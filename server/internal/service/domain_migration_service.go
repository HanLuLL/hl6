package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"
)

// MigrationWorkerConcurrency is the number of goroutines that can process migration tasks concurrently
// (across different domains; same domain is always serialized).
const MigrationWorkerConcurrency = 5

// CreateMigrationInput is the input for creating a new migration task.
type CreateMigrationInput struct {
	DomainID              uint
	TriggeredBy           uint
	TargetProviderAccountID uint
	TargetProviderZoneID  string
	Reason                string
}

// CreateMigrationResult is the result of creating a migration task.
type CreateMigrationResult struct {
	TaskID              uint   `json:"task_id"`
	Status              string `json:"status"`
	QueuePosition       int    `json:"queue_position"`
	DomainMigrationState string `json:"domain_migration_state"`
}

// RetryFailuresResult is the result of retrying failed migration items.
type RetryFailuresResult struct {
	RetriedItems       int    `json:"retried_items"`
	RemainingFailed    int    `json:"remaining_failed_items"`
	Status             string `json:"status"`
}

// CleanupSourceResult is the result of cleaning up source provider records.
type CleanupSourceResult struct {
	CleanupTotal     int `json:"cleanup_total"`
	CleanupSucceeded int `json:"cleanup_succeeded"`
	CleanupFailed    int `json:"cleanup_failed"`
}

// DomainMigrationService orchestrates DNS provider migrations.
type DomainMigrationService struct {
	repo *repository.Repository
	cfg  *config.Config
}

// NewDomainMigrationService creates a new DomainMigrationService and starts the background worker.
func NewDomainMigrationService(repo *repository.Repository, cfg *config.Config) *DomainMigrationService {
	svc := &DomainMigrationService{repo: repo, cfg: cfg}
	go svc.resumePendingTasks()
	return svc
}

// Repo returns the underlying repository for use by handlers.
func (s *DomainMigrationService) Repo() *repository.Repository {
	return s.repo
}

// resumePendingTasks resets any stuck-running tasks to pending and resumes pending tasks on startup.
// Tasks are grouped by domain and only the first (lowest queue_seq) task per domain is started,
// preserving serial execution order within each domain.
func (s *DomainMigrationService) resumePendingTasks() {
	if s.repo == nil {
		return
	}
	_ = s.repo.ResetRunningMigrationTasksToPending()
	tasks, err := s.repo.FindPendingMigrationTasksAll(200)
	if err != nil {
		return
	}
	// Only start the first pending task per domain (lowest queue_seq).
	// Subsequent tasks will be picked up by finishTask → FindNextPendingMigrationTask.
	seen := make(map[uint]bool, len(tasks))
	for _, task := range tasks {
		if seen[task.DomainID] {
			continue
		}
		seen[task.DomainID] = true
		go s.runTask(task.ID)
	}
}

// CreateMigration creates a new migration task for a domain and immediately switches the domain's
// active provider to the target (per FR-016). If a running task exists, the new task is queued.
func (s *DomainMigrationService) CreateMigration(ctx context.Context, input CreateMigrationInput) (*CreateMigrationResult, error) {
	domain, err := s.repo.FindDomain(input.DomainID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	// Validate target account
	targetAccount, err := s.repo.FindDNSProviderAccount(input.TargetProviderAccountID)
	if err != nil {
		return nil, fmt.Errorf("target provider account not found: %w", err)
	}
	if targetAccount.Status == model.DNSProviderAccountStatusDisabled {
		return nil, errors.New("target provider account is disabled")
	}
	if input.TargetProviderZoneID == "" {
		return nil, errors.New("target_provider_zone_id is required")
	}

	// Snapshot source provider info
	sourceProvider := domain.Provider
	sourceAccountID := domain.ProviderAccountID
	sourceZoneID := domain.ProviderZoneID

	task := &model.DomainDNSMigrationTask{
		DomainID:        input.DomainID,
		Status:          model.MigrationTaskStatusPending,
		TriggeredBy:     input.TriggeredBy,
		SourceProvider:  sourceProvider,
		SourceAccountID: sourceAccountID,
		SourceZoneID:    sourceZoneID,
		TargetProvider:  targetAccount.Provider,
		TargetAccountID: input.TargetProviderAccountID,
		TargetZoneID:    input.TargetProviderZoneID,
		Reason:          input.Reason,
	}

	if err := s.repo.CreateMigrationTask(task); err != nil {
		return nil, fmt.Errorf("create migration task: %w", err)
	}

	// Immediately switch domain to target provider (FR-016)
	if err := s.repo.DB.Model(domain).Updates(map[string]interface{}{
		"provider":            targetAccount.Provider,
		"provider_account_id": input.TargetProviderAccountID,
		"provider_zone_id":    input.TargetProviderZoneID,
	}).Error; err != nil {
		return nil, fmt.Errorf("switch domain provider: %w", err)
	}

	// Determine if running immediately or queued
	runningTask, _ := s.repo.FindRunningMigrationTask(input.DomainID)
	taskStatus := model.MigrationTaskStatusPending
	queuePos := 1
	newDomainState := model.DomainMigrationStateQueued

	if runningTask == nil {
		// No running task — start immediately
		taskStatus = model.MigrationTaskStatusRunning
		queuePos = 0
		newDomainState = model.DomainMigrationStateRunning
		taskIDPtr := task.ID
		if err := s.repo.UpdateDomainMigrationState(input.DomainID, model.DomainMigrationStateRunning, true, &taskIDPtr); err != nil {
			log.Printf("update domain migration state error: %v", err)
		}
		go s.runTask(task.ID)
	} else {
		taskIDPtr := task.ID
		_ = s.repo.UpdateDomainMigrationState(input.DomainID, model.DomainMigrationStateQueued, domain.MigrationReadOnly, &taskIDPtr)
	}

	return &CreateMigrationResult{
		TaskID:               task.ID,
		Status:               taskStatus,
		QueuePosition:        queuePos,
		DomainMigrationState: newDomainState,
	}, nil
}

// runTask executes a migration task, processing all pending DNS records.
func (s *DomainMigrationService) runTask(taskID uint) {
	task, err := s.repo.FindMigrationTask(taskID)
	if err != nil {
		log.Printf("migration worker: load task %d: %v", taskID, err)
		return
	}

	// Mark task as running
	startedAt := time.Now()
	if err := s.repo.UpdateMigrationTask(taskID, map[string]interface{}{
		"status":     model.MigrationTaskStatusRunning,
		"started_at": &startedAt,
	}); err != nil {
		log.Printf("migration worker: start task %d: %v", taskID, err)
		return
	}
	if err := s.repo.UpdateDomainMigrationState(task.DomainID, model.DomainMigrationStateRunning, true, &taskID); err != nil {
		log.Printf("migration worker: update domain state %d: %v", task.DomainID, err)
	}

	// Load all DNS records for the domain
	records, err := s.loadDomainRecords(task.DomainID)
	if err != nil {
		s.finishTask(task, model.MigrationTaskStatusFailed, "failed to load dns records: "+err.Error())
		return
	}

	// Create migration items if not already created
	items, _, _ := s.repo.ListMigrationItems(taskID, 1, 10000)
	if len(items) == 0 && len(records) > 0 {
		items = make([]model.DomainDNSMigrationItem, 0, len(records))
		for _, rec := range records {
			items = append(items, model.DomainDNSMigrationItem{
				TaskID:                 taskID,
				DNSRecordID:            rec.ID,
				RecordType:             rec.Type,
				Name:                   rec.Name,
				Content:                rec.Content,
				TTL:                    rec.TTL,
				Proxied:                rec.Proxied,
				SourceProviderRecordID: rec.ProviderRecordID,
				Status:                 model.MigrationItemStatusPending,
			})
		}
		if err := s.repo.CreateMigrationItems(items); err != nil {
			s.finishTask(task, model.MigrationTaskStatusFailed, "failed to create migration items: "+err.Error())
			return
		}
		_ = s.repo.UpdateMigrationTask(taskID, map[string]interface{}{"total_items": len(items)})
		// Reload items
		items, _, _ = s.repo.ListMigrationItems(taskID, 1, 10000)
	}

	// Build target provider client
	targetClient, err := s.buildClientForAccount(task.TargetAccountID, task.TargetProvider)
	if err != nil {
		s.finishTask(task, model.MigrationTaskStatusFailed, "failed to build target provider client: "+err.Error())
		return
	}

	succeeded := 0
	failed := 0
	for i := range items {
		item := &items[i]
		if item.Status == model.MigrationItemStatusSucceeded || item.Status == model.MigrationItemStatusSkipped {
			succeeded++
			continue
		}

		_ = s.repo.UpdateMigrationItem(item.ID, map[string]interface{}{
			"status":   model.MigrationItemStatusRunning,
			"attempts": item.Attempts + 1,
		})

		opCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		targetID, createErr := targetClient.CreateRecord(opCtx,
			task.TargetZoneID,
			item.RecordType,
			item.Name,
			item.Content,
			item.TTL,
			item.Proxied,
		)
		cancel()

		finishedAt := time.Now()
		if createErr != nil {
			errCat, _ := ClassifyProviderError(createErr)
			failed++
			_ = s.repo.UpdateMigrationItem(item.ID, map[string]interface{}{
				"status":              model.MigrationItemStatusFailed,
				"last_error_category": string(errCat),
				"last_error_message":  createErr.Error(),
				"finished_at":         &finishedAt,
				"attempts":            item.Attempts + 1,
			})
		} else {
			succeeded++
			updates := map[string]interface{}{
				"status":                   model.MigrationItemStatusSucceeded,
				"target_provider_record_id": targetID,
				"last_error_category":       "",
				"last_error_message":        "",
				"finished_at":               &finishedAt,
				"attempts":                  item.Attempts + 1,
			}
			_ = s.repo.UpdateMigrationItem(item.ID, updates)
			// Also update the dns_record with the new provider_record_id
			_ = s.repo.DB.Model(&model.DNSRecord{}).Where("id = ?", item.DNSRecordID).
				Update("provider_record_id", targetID)
		}

		_ = s.repo.UpdateMigrationTask(taskID, map[string]interface{}{
			"succeeded_items": succeeded,
			"failed_items":    failed,
		})
	}

	finalStatus := model.MigrationTaskStatusSucceeded
	if failed > 0 && succeeded == 0 {
		finalStatus = model.MigrationTaskStatusFailed
	} else if failed > 0 {
		finalStatus = model.MigrationTaskStatusPartialFailed
	}
	s.finishTask(task, finalStatus, fmt.Sprintf("completed: succeeded=%d failed=%d", succeeded, failed))
}

func (s *DomainMigrationService) finishTask(task *model.DomainDNSMigrationTask, status, message string) {
	finishedAt := time.Now()
	_ = s.repo.UpdateMigrationTask(task.ID, map[string]interface{}{
		"status":      status,
		"finished_at": &finishedAt,
	})

	// Update domain migration state
	domainState := model.DomainMigrationStateIdle
	switch status {
	case model.MigrationTaskStatusPartialFailed:
		domainState = model.DomainMigrationStatePartialFailed
	case model.MigrationTaskStatusFailed:
		domainState = model.DomainMigrationStateFailed
	}

	taskIDPtr := task.ID
	_ = s.repo.UpdateDomainMigrationState(task.DomainID, domainState, false, &taskIDPtr)

	// Check for next queued task on this domain
	next, _ := s.repo.FindNextPendingMigrationTask(task.DomainID)
	if next != nil {
		go s.runTask(next.ID)
	}

	log.Printf("migration task %d finished: %s — %s", task.ID, status, message)
}

// RetryFailures retries all failed items in a migration task.
func (s *DomainMigrationService) RetryFailures(ctx context.Context, taskID uint) (*RetryFailuresResult, error) {
	task, err := s.repo.FindMigrationTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	failedItems, err := s.repo.ListFailedMigrationItems(taskID)
	if err != nil {
		return nil, fmt.Errorf("list failed items: %w", err)
	}
	if len(failedItems) == 0 {
		return &RetryFailuresResult{Status: task.Status}, nil
	}

	targetClient, err := s.buildClientForAccount(task.TargetAccountID, task.TargetProvider)
	if err != nil {
		return nil, fmt.Errorf("build target client: %w", err)
	}

	retried := 0
	for i := range failedItems {
		item := &failedItems[i]

		opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		// Try upsert: find existing then update or create
		existingID, findErr := targetClient.FindRecord(opCtx, task.TargetZoneID, item.RecordType, item.Name, item.Content)
		cancel()

		finishedAt := time.Now()
		if findErr == nil && existingID != "" {
			// Conflict: overwrite with current value
			opCtx2, cancel2 := context.WithTimeout(ctx, 30*time.Second)
			updateErr := targetClient.UpdateRecord(opCtx2, task.TargetZoneID, existingID, item.RecordType, item.Name, item.Content, item.TTL, item.Proxied)
			cancel2()
			if updateErr != nil {
				errCat, _ := ClassifyProviderError(updateErr)
				_ = s.repo.UpdateMigrationItem(item.ID, map[string]interface{}{
					"last_error_category": string(errCat),
					"last_error_message":  updateErr.Error(),
					"attempts":            gorm.Expr("attempts + 1"),
				})
				continue
			}
			overwriteBefore, _ := json.Marshal(map[string]string{"provider_record_id": existingID})
			overwriteAfter, _ := json.Marshal(map[string]string{"provider_record_id": existingID, "content": item.Content})
			_ = s.repo.UpdateMigrationItem(item.ID, map[string]interface{}{
				"status":                   model.MigrationItemStatusSucceeded,
				"target_provider_record_id": existingID,
				"conflict_overwritten":      true,
				"overwrite_before":          overwriteBefore,
				"overwrite_after":           overwriteAfter,
				"last_error_category":       "",
				"last_error_message":        "",
				"finished_at":               &finishedAt,
				"attempts":                  gorm.Expr("attempts + 1"),
			})
			retried++
			continue
		}

		opCtx3, cancel3 := context.WithTimeout(ctx, 30*time.Second)
		targetID, createErr := targetClient.CreateRecord(opCtx3, task.TargetZoneID, item.RecordType, item.Name, item.Content, item.TTL, item.Proxied)
		cancel3()
		if createErr != nil {
			errCat, _ := ClassifyProviderError(createErr)
			_ = s.repo.UpdateMigrationItem(item.ID, map[string]interface{}{
				"last_error_category": string(errCat),
				"last_error_message":  createErr.Error(),
				"attempts":            gorm.Expr("attempts + 1"),
			})
			continue
		}
		_ = s.repo.UpdateMigrationItem(item.ID, map[string]interface{}{
			"status":                   model.MigrationItemStatusSucceeded,
			"target_provider_record_id": targetID,
			"last_error_category":       "",
			"last_error_message":        "",
			"finished_at":               &finishedAt,
			"attempts":                  gorm.Expr("attempts + 1"),
		})
		_ = s.repo.DB.Model(&model.DNSRecord{}).Where("id = ?", item.DNSRecordID).
			Update("provider_record_id", targetID)
		retried++
	}

	// Recount failures
	remaining, _ := s.repo.ListFailedMigrationItems(taskID)
	remainingCount := len(remaining)

	newStatus := task.Status
	if remainingCount == 0 {
		newStatus = model.MigrationTaskStatusSucceeded
		finishedAt := time.Now()
		taskIDPtr := task.ID
		_ = s.repo.UpdateMigrationTask(taskID, map[string]interface{}{
			"status":      newStatus,
			"finished_at": &finishedAt,
		})
		_ = s.repo.UpdateDomainMigrationState(task.DomainID, model.DomainMigrationStateIdle, false, &taskIDPtr)
	} else {
		_ = s.repo.UpdateMigrationTask(taskID, map[string]interface{}{
			"retried_items": gorm.Expr("retried_items + ?", retried),
			"failed_items":  remainingCount,
		})
	}

	return &RetryFailuresResult{
		RetriedItems:    retried,
		RemainingFailed: remainingCount,
		Status:          newStatus,
	}, nil
}

// CleanupSource deletes all platform-managed records from the source provider.
func (s *DomainMigrationService) CleanupSource(ctx context.Context, taskID uint, confirmDomainName, confirmPhrase string) (*CleanupSourceResult, error) {
	task, err := s.repo.FindMigrationTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Validate confirm fields
	domain, err := s.repo.FindDomain(task.DomainID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(confirmDomainName), strings.TrimSpace(domain.Name)) {
		return nil, &ProviderError{
			Category: ErrCategoryInvalidRequest,
			Err:      errors.New("confirm_domain_name does not match domain name"),
		}
	}
	if strings.TrimSpace(confirmPhrase) != "CLEANUP" {
		return nil, &ProviderError{
			Category: ErrCategoryInvalidRequest,
			Err:      errors.New("confirm_phrase must be CLEANUP"),
		}
	}

	sourceClient, err := s.buildClientForAccount(task.SourceAccountID, task.SourceProvider)
	if err != nil {
		return nil, fmt.Errorf("build source client: %w", err)
	}

	items, _, err := s.repo.ListMigrationItems(taskID, 1, 10000)
	if err != nil {
		return nil, fmt.Errorf("list migration items: %w", err)
	}

	total := 0
	succeeded := 0
	failed := 0
	for _, item := range items {
		sourceID := strings.TrimSpace(item.SourceProviderRecordID)
		if sourceID == "" {
			continue
		}
		total++
		opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := sourceClient.DeleteRecord(opCtx, task.SourceZoneID, sourceID)
		cancel()
		if err != nil {
			failed++
			log.Printf("migration %d cleanup source record %s: %v", taskID, sourceID, err)
		} else {
			succeeded++
		}
	}

	return &CleanupSourceResult{
		CleanupTotal:     total,
		CleanupSucceeded: succeeded,
		CleanupFailed:    failed,
	}, nil
}

func (s *DomainMigrationService) buildClientForAccount(accountID uint, providerHint string) (DNSProviderClient, error) {
	account, err := s.repo.FindDNSProviderAccount(accountID)
	if err != nil {
		return nil, err
	}
	rawCredentials := strings.TrimSpace(account.Credentials)
	provider := model.NormalizeProvider(account.Provider)
	if provider == "" {
		provider = providerHint
	}
	if hint := model.NormalizeProvider(providerHint); hint != "" {
		provider = hint
	}
	credentials, err := ParseProviderCredentials(provider, crypto.DecryptOrPlaintext(rawCredentials, s.cfg.EncryptionKey))
	if err != nil {
		return nil, err
	}
	return BuildProviderClient(provider, credentials)
}

func (s *DomainMigrationService) loadDomainRecords(domainID uint) ([]model.DNSRecord, error) {
	var subdomains []model.Subdomain
	if err := s.repo.DB.Where("domain_id = ?", domainID).Find(&subdomains).Error; err != nil {
		return nil, err
	}

	var records []model.DNSRecord
	for _, sub := range subdomains {
		var recs []model.DNSRecord
		if err := s.repo.DB.Where("subdomain_id = ?", sub.ID).Find(&recs).Error; err != nil {
			return nil, err
		}
		records = append(records, recs...)
	}
	return records, nil
}
