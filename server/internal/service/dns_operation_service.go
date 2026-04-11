package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"

	"gorm.io/gorm"
)

const (
	singleOperationTimeout       = 3 * time.Second
	compensationOperationTimeout = 8 * time.Second
	singleOperationAttempts      = 2
	defaultBatchMaxAttempts      = 3
	defaultBatchAsyncThreshold   = 200
	retryBackoffMilliseconds     = 180
)

type operationRequestIDKey struct{}

type OperationResult struct {
	HTTPStatus int         `json:"http_status"`
	Message    string      `json:"message"`
	MessageKey string      `json:"message_key,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Retryable  bool        `json:"retryable"`
}

type CreateRecordInput struct {
	Subdomain *model.Subdomain
	Record    *model.DNSRecord
}

type UpdateRecordInput struct {
	Subdomain    *model.Subdomain
	Record       *model.DNSRecord
	NewContent   string
	NewTTL       int
	NewProxied   bool
	BeforeDBSave func(tx *gorm.DB, record *model.DNSRecord) error
}

type DeleteRecordInput struct {
	Subdomain *model.Subdomain
	Record    *model.DNSRecord
}

type BatchDeleteItem struct {
	RecordID          uint
	SubdomainFQDN     string
	Provider          string
	ProviderAccountID uint
	ZoneID            string
	ProviderRecordID  string
	RecordType        string
	Name              string
	Content           string
	TTL               int
	Proxied           bool
}

type BatchDeleteFailure struct {
	RecordID         uint   `json:"record_id"`
	SubdomainFQDN    string `json:"subdomain_fqdn"`
	RecordType       string `json:"record_type"`
	RecordContent    string `json:"record_content"`
	ProviderRecordID string `json:"provider_record_id"`
	Error            string `json:"error"`
}

type BatchDeleteResult struct {
	Total     int                  `json:"total"`
	Succeeded int                  `json:"succeeded"`
	Failed    int                  `json:"failed"`
	Failures  []BatchDeleteFailure `json:"failures"`
	Async     bool                 `json:"async,omitempty"`
	JobID     uint                 `json:"job_id,omitempty"`
}

type DNSOperationService struct {
	repo                *repository.Repository
	cfg                 *config.Config
	batchAsyncThreshold int
}

func NewDNSOperationService(repo *repository.Repository, cfg *config.Config) *DNSOperationService {
	threshold := defaultBatchAsyncThreshold
	if cfg != nil && cfg.DNSBatchThreshold > 0 {
		threshold = cfg.DNSBatchThreshold
	}
	svc := &DNSOperationService{
		repo:                repo,
		cfg:                 cfg,
		batchAsyncThreshold: threshold,
	}
	go svc.resumePendingBulkJobs()
	return svc
}

func (s *DNSOperationService) resumePendingBulkJobs() {
	if s.repo == nil {
		return
	}
	_ = s.repo.ResetRunningDNSBulkJobsToPending()
	jobs, err := s.repo.ListPendingDNSBulkJobs(200)
	if err != nil {
		return
	}
	for _, job := range jobs {
		go s.runBulkDeleteJob(job.ID)
	}
}

func (s *DNSOperationService) ExecuteIdempotent(ctx context.Context, scope, idempotencyKey string, exec func(context.Context) (OperationResult, error)) OperationResult {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return OperationResult{
			HTTPStatus: 400,
			Message:    "missing idempotency key",
			MessageKey: "error.invalidRequestBody",
			Retryable:  false,
		}
	}
	req, created, err := s.repo.TryCreateDNSOperationRequest(scope, key)
	if err != nil {
		return OperationResult{
			HTTPStatus: 500,
			Message:    "failed to persist idempotency request",
			MessageKey: "error.databaseError",
			Retryable:  true,
		}
	}

	if !created {
		if req.Status == model.DNSOperationRequestStatusRunning {
			return OperationResult{
				HTTPStatus: 409,
				Message:    "operation is in progress",
				MessageKey: "error.cloudflareOperationInProgress",
				Retryable:  true,
			}
		}
		return decodeOperationResult(req)
	}

	execCtx := withOperationRequestID(ctx, req.ID)
	result, execErr := exec(execCtx)
	if execErr != nil {
		result = OperationResult{
			HTTPStatus: 500,
			Message:    execErr.Error(),
			MessageKey: "error.databaseError",
			Retryable:  true,
		}
	}
	if result.HTTPStatus == 0 {
		result.HTTPStatus = 500
		if result.Message == "" {
			result.Message = "empty operation result"
		}
		if result.MessageKey == "" {
			result.MessageKey = "error.databaseError"
		}
		result.Retryable = true
	}

	status := model.DNSOperationRequestStatusSucceeded
	if result.HTTPStatus >= 400 {
		status = model.DNSOperationRequestStatusFailed
	}
	if completeErr := s.repo.CompleteDNSOperationRequest(req.ID, status, result.HTTPStatus, result.Message, result.MessageKey, result.Retryable, result.Data); completeErr != nil {
		return OperationResult{
			HTTPStatus: 500,
			Message:    "failed to persist operation result",
			MessageKey: "error.databaseError",
			Retryable:  true,
		}
	}
	return result
}

func decodeOperationResult(req *model.DNSOperationRequest) OperationResult {
	res := OperationResult{
		HTTPStatus: req.HTTPStatus,
		Message:    req.Message,
		MessageKey: req.MessageKey,
		Retryable:  req.Retryable,
	}
	if res.HTTPStatus == 0 {
		if req.Status == model.DNSOperationRequestStatusSucceeded {
			res.HTTPStatus = 200
		} else {
			res.HTTPStatus = 500
		}
	}
	if len(req.ResponseData) > 0 {
		var data interface{}
		if err := json.Unmarshal(req.ResponseData, &data); err == nil {
			res.Data = data
		}
	}
	return res
}

func (s *DNSOperationService) CreateRecordAtomic(ctx context.Context, input CreateRecordInput, beforePersist func(tx *gorm.DB) error) error {
	if input.Subdomain == nil || input.Record == nil {
		return errors.New("create record input is incomplete")
	}
	requestID := operationRequestIDFromCtx(ctx)
	client, provider, err := s.providerClientForAccount(input.Subdomain.Domain.ProviderAccountID, input.Subdomain.Domain.Provider)
	if err != nil {
		return err
	}

	opCtx, cancel := newOperationCtx(ctx)
	defer cancel()

	providerRecordID, err := retryValue(opCtx, singleOperationAttempts, func(callCtx context.Context) (string, error) {
		return client.CreateRecord(callCtx,
			input.Subdomain.Domain.ProviderZoneID,
			input.Record.Type,
			input.Record.Name,
			input.Record.Content,
			input.Record.TTL,
			input.Record.Proxied,
		)
	})
	if err != nil {
		s.logEvent(requestID, "create_provider", false, err.Error(), map[string]interface{}{"record": input.Record.Name}, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, "")
		return err
	}

	input.Record.ProviderRecordID = strings.TrimSpace(providerRecordID)
	if dbErr := s.repo.Transaction(func(tx *gorm.DB) error {
		if beforePersist != nil {
			if err := beforePersist(tx); err != nil {
				return err
			}
		}
		return tx.Create(input.Record).Error
	}); dbErr != nil {
		compensateCtx, compensateCancel := newCompensationCtx()
		defer compensateCancel()
		rollbackErr := retryError(compensateCtx, singleOperationAttempts, func(callCtx context.Context) error {
			return client.DeleteRecord(callCtx, input.Subdomain.Domain.ProviderZoneID, providerRecordID)
		})
		s.logEvent(requestID, "create_db", false, dbErr.Error(), map[string]interface{}{"provider_record_id": providerRecordID}, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
		if rollbackErr != nil {
			s.logEvent(requestID, "create_compensate_delete", false, rollbackErr.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
			return fmt.Errorf("database create failed (%v), provider rollback failed (%w)", dbErr, rollbackErr)
		}
		s.logEvent(requestID, "create_compensate_delete", true, "rollback succeeded", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
		return fmt.Errorf("database create failed after provider create: %w", dbErr)
	}

	s.logEvent(requestID, "create_done", true, "ok", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
	return nil
}

func (s *DNSOperationService) UpdateRecordAtomic(ctx context.Context, input UpdateRecordInput) error {
	if input.Subdomain == nil || input.Record == nil {
		return errors.New("update record input is incomplete")
	}
	requestID := operationRequestIDFromCtx(ctx)
	client, provider, err := s.providerClientForAccount(input.Subdomain.Domain.ProviderAccountID, input.Subdomain.Domain.Provider)
	if err != nil {
		return err
	}

	oldRecord := *input.Record
	recordID := strings.TrimSpace(input.Record.ProviderRecordID)
	if recordID == "" {
		return errors.New("provider record id is empty")
	}

	opCtx, cancel := newOperationCtx(ctx)
	defer cancel()

	if err := retryError(opCtx, singleOperationAttempts, func(callCtx context.Context) error {
		return client.UpdateRecord(callCtx,
			input.Subdomain.Domain.ProviderZoneID,
			recordID,
			input.Record.Type,
			input.Record.Name,
			input.NewContent,
			input.NewTTL,
			input.NewProxied,
		)
	}); err != nil {
		s.logEvent(requestID, "update_provider", false, err.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
		return err
	}

	if dbErr := s.repo.Transaction(func(tx *gorm.DB) error {
		input.Record.Content = input.NewContent
		input.Record.TTL = input.NewTTL
		input.Record.Proxied = input.NewProxied
		if input.BeforeDBSave != nil {
			if err := input.BeforeDBSave(tx, input.Record); err != nil {
				return err
			}
		}
		return tx.Save(input.Record).Error
	}); dbErr != nil {
		compensateCtx, compensateCancel := newCompensationCtx()
		defer compensateCancel()
		rollbackErr := retryError(compensateCtx, singleOperationAttempts, func(callCtx context.Context) error {
			return client.UpdateRecord(callCtx,
				input.Subdomain.Domain.ProviderZoneID,
				recordID,
				oldRecord.Type,
				oldRecord.Name,
				oldRecord.Content,
				oldRecord.TTL,
				oldRecord.Proxied,
			)
		})
		s.logEvent(requestID, "update_db", false, dbErr.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
		if rollbackErr != nil {
			s.logEvent(requestID, "update_compensate", false, rollbackErr.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
			return fmt.Errorf("database update failed (%v), provider rollback failed (%w)", dbErr, rollbackErr)
		}
		s.logEvent(requestID, "update_compensate", true, "rollback succeeded", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
		return fmt.Errorf("database update failed after provider update: %w", dbErr)
	}

	s.logEvent(requestID, "update_done", true, "ok", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
	return nil
}

func (s *DNSOperationService) DeleteRecordAtomic(ctx context.Context, input DeleteRecordInput) error {
	if input.Subdomain == nil || input.Record == nil {
		return errors.New("delete record input is incomplete")
	}
	item := BatchDeleteItem{
		RecordID:          input.Record.ID,
		SubdomainFQDN:     input.Subdomain.FQDN,
		Provider:          input.Subdomain.Domain.Provider,
		ProviderAccountID: input.Subdomain.Domain.ProviderAccountID,
		ZoneID:            input.Subdomain.Domain.ProviderZoneID,
		ProviderRecordID:  input.Record.ProviderRecordID,
		RecordType:        input.Record.Type,
		Name:              input.Record.Name,
		Content:           input.Record.Content,
		TTL:               input.Record.TTL,
		Proxied:           input.Record.Proxied,
	}
	opCtx, cancel := newOperationCtx(ctx)
	defer cancel()
	_, err := s.deleteSingleWithRetry(opCtx, item, singleOperationAttempts)
	return err
}

func (s *DNSOperationService) DeleteRecordsBatch(ctx context.Context, items []BatchDeleteItem, maxAttempts int) BatchDeleteResult {
	if maxAttempts <= 0 {
		maxAttempts = defaultBatchMaxAttempts
	}
	if len(items) > s.batchAsyncThreshold {
		jobID, err := s.enqueueBulkDeleteJob(items, maxAttempts)
		if err != nil {
			return BatchDeleteResult{
				Total:    len(items),
				Failed:   len(items),
				Failures: []BatchDeleteFailure{{Error: err.Error()}},
			}
		}
		return BatchDeleteResult{
			Total: len(items),
			Async: true,
			JobID: jobID,
		}
	}
	result := BatchDeleteResult{Total: len(items), Failures: make([]BatchDeleteFailure, 0)}
	for _, item := range items {
		success, err := s.deleteSingleWithRetry(ctx, item, maxAttempts)
		if success {
			result.Succeeded++
			continue
		}
		result.Failed++
		failure := BatchDeleteFailure{
			RecordID:         item.RecordID,
			SubdomainFQDN:    item.SubdomainFQDN,
			RecordType:       item.RecordType,
			RecordContent:    item.Content,
			ProviderRecordID: item.ProviderRecordID,
		}
		if err != nil {
			failure.Error = err.Error()
		}
		result.Failures = append(result.Failures, failure)
	}
	return result
}

func (s *DNSOperationService) enqueueBulkDeleteJob(items []BatchDeleteItem, maxAttempts int) (uint, error) {
	now := time.Now()
	job := &model.DNSBulkJob{
		Scope:       "dns_delete_batch",
		Status:      model.DNSBulkJobStatusPending,
		TotalItems:  len(items),
		MaxAttempts: maxAttempts,
		StartedAt:   &now,
	}
	jobItems := make([]model.DNSBulkJobItem, 0, len(items))
	for _, item := range items {
		jobItems = append(jobItems, model.DNSBulkJobItem{
			RecordID:          item.RecordID,
			SubdomainFQDN:     item.SubdomainFQDN,
			Provider:          item.Provider,
			ProviderAccountID: item.ProviderAccountID,
			ZoneID:            item.ZoneID,
			ProviderRecordID:  item.ProviderRecordID,
			RecordType:        item.RecordType,
			Name:              item.Name,
			Content:           item.Content,
			TTL:               item.TTL,
			Proxied:           item.Proxied,
			Status:            model.DNSBulkJobItemStatusPending,
		})
	}
	if err := s.repo.CreateDNSBulkJobWithItems(job, jobItems); err != nil {
		return 0, err
	}
	go s.runBulkDeleteJob(job.ID)
	return job.ID, nil
}

func (s *DNSOperationService) runBulkDeleteJob(jobID uint) {
	job, err := s.repo.FindDNSBulkJob(jobID)
	if err != nil {
		return
	}
	startedAt := time.Now()
	started, err := s.repo.TryStartDNSBulkJob(jobID, startedAt)
	if err != nil || !started {
		return
	}
	items, err := s.repo.ListAllDNSBulkJobItems(jobID)
	if err != nil {
		finishedAt := time.Now()
		_ = s.repo.UpdateDNSBulkJob(jobID, map[string]interface{}{
			"status":      model.DNSBulkJobStatusFailed,
			"finished_at": &finishedAt,
			"message":     fmt.Sprintf("load job items failed: %v", err),
		})
		return
	}
	maxAttempts := job.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = defaultBatchMaxAttempts
	}

	succeeded := 0
	failed := 0
	for _, item := range items {
		_ = s.repo.UpdateDNSBulkJobItem(item.ID, map[string]interface{}{
			"status": model.DNSBulkJobItemStatusRunning,
		})
		batchItem := BatchDeleteItem{
			RecordID:          item.RecordID,
			SubdomainFQDN:     item.SubdomainFQDN,
			Provider:          item.Provider,
			ProviderAccountID: item.ProviderAccountID,
			ZoneID:            item.ZoneID,
			ProviderRecordID:  item.ProviderRecordID,
			RecordType:        item.RecordType,
			Name:              item.Name,
			Content:           item.Content,
			TTL:               item.TTL,
			Proxied:           item.Proxied,
		}
		success, attempts, deleteErr := s.deleteSingleWithRetryDetailed(context.Background(), batchItem, maxAttempts)
		finishedAt := time.Now()
		if success {
			succeeded++
			_ = s.repo.UpdateDNSBulkJobItem(item.ID, map[string]interface{}{
				"status":      model.DNSBulkJobItemStatusSucceeded,
				"attempts":    attempts,
				"last_error":  "",
				"finished_at": &finishedAt,
			})
		} else {
			failed++
			msg := "delete failed"
			if deleteErr != nil {
				msg = deleteErr.Error()
			}
			_ = s.repo.UpdateDNSBulkJobItem(item.ID, map[string]interface{}{
				"status":      model.DNSBulkJobItemStatusFailed,
				"attempts":    attempts,
				"last_error":  msg,
				"finished_at": &finishedAt,
			})
		}
		_ = s.repo.UpdateDNSBulkJob(jobID, map[string]interface{}{
			"succeeded_items": succeeded,
			"failed_items":    failed,
		})
	}

	finalStatus := model.DNSBulkJobStatusSucceeded
	if failed > 0 {
		finalStatus = model.DNSBulkJobStatusFailed
	}
	finishedAt := time.Now()
	_ = s.repo.UpdateDNSBulkJob(jobID, map[string]interface{}{
		"status":      finalStatus,
		"finished_at": &finishedAt,
		"message":     fmt.Sprintf("completed: succeeded=%d failed=%d", succeeded, failed),
	})
}

func (s *DNSOperationService) deleteSingleWithRetry(ctx context.Context, item BatchDeleteItem, maxAttempts int) (bool, error) {
	success, _, err := s.deleteSingleWithRetryDetailed(ctx, item, maxAttempts)
	return success, err
}

func (s *DNSOperationService) deleteSingleWithRetryDetailed(ctx context.Context, item BatchDeleteItem, maxAttempts int) (bool, int, error) {
	if maxAttempts <= 0 {
		maxAttempts = defaultBatchMaxAttempts
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var lastErr error
	attemptUsed := 0
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptUsed = attempt
		if err := s.deleteSingle(ctx, item); err == nil {
			return true, attemptUsed, nil
		} else {
			lastErr = err
		}
		if attempt == maxAttempts || !shouldRetryProviderError(lastErr) {
			break
		}
		backoff := time.Duration(1<<(attempt-1)) * time.Duration(retryBackoffMilliseconds) * time.Millisecond
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, attemptUsed, ctx.Err()
		case <-timer.C:
		}
	}
	return false, attemptUsed, lastErr
}

func (s *DNSOperationService) deleteSingle(ctx context.Context, item BatchDeleteItem) error {
	requestID := operationRequestIDFromCtx(ctx)
	client, provider, err := s.providerClientForAccount(item.ProviderAccountID, item.Provider)
	if err != nil {
		return err
	}

	callCtx, cancel := newOperationCtx(ctx)
	defer cancel()

	recordID := strings.TrimSpace(item.ProviderRecordID)
	providerDeleted := false
	if recordID == "" {
		if strings.TrimSpace(item.RecordType) != "" && strings.TrimSpace(item.Name) != "" && strings.TrimSpace(item.Content) != "" {
			recordID, err = retryValue(callCtx, singleOperationAttempts, func(c context.Context) (string, error) {
				return client.FindRecord(c, item.ZoneID, item.RecordType, item.Name, item.Content)
			})
			if err != nil {
				if isProviderRecordNotFoundErr(err) {
					recordID = ""
				} else {
					s.logEvent(requestID, "delete_find_provider", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, "")
					return err
				}
			}
		}
	}

	if recordID != "" {
		if err := retryError(callCtx, singleOperationAttempts, func(c context.Context) error {
			return client.DeleteRecord(c, item.ZoneID, recordID)
		}); err != nil {
			s.logEvent(requestID, "delete_provider", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
			return err
		}
		providerDeleted = true

		// When provider_record_id is stale (e.g. previous compensation recreated the record with a new ID),
		// deleting by old ID may be a no-op on provider side. Re-check by logical key and delete again if needed.
		if strings.TrimSpace(item.RecordType) != "" && strings.TrimSpace(item.Name) != "" && strings.TrimSpace(item.Content) != "" {
			remainingID, findErr := retryValue(callCtx, singleOperationAttempts, func(c context.Context) (string, error) {
				return client.FindRecord(c, item.ZoneID, item.RecordType, item.Name, item.Content)
			})
			if findErr != nil {
				if !isProviderRecordNotFoundErr(findErr) {
					s.logEvent(requestID, "delete_verify_provider", false, findErr.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
					return findErr
				}
			} else {
				remainingID = strings.TrimSpace(remainingID)
				if remainingID != "" {
					if err := retryError(callCtx, singleOperationAttempts, func(c context.Context) error {
						return client.DeleteRecord(c, item.ZoneID, remainingID)
					}); err != nil {
						s.logEvent(requestID, "delete_provider_by_lookup", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, remainingID)
						return err
					}
					recordID = remainingID
				}
			}
		}
	}

	if item.RecordID > 0 {
		if err := s.repo.DeleteDNSRecord(item.RecordID); err != nil {
			s.logEvent(requestID, "delete_db", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
			if providerDeleted {
				if strings.TrimSpace(item.RecordType) == "" || strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Content) == "" {
					rollbackErr := errors.New("insufficient record data for provider rollback create")
					s.logEvent(requestID, "delete_compensate_recreate", false, rollbackErr.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
					return fmt.Errorf("delete local record failed (%v), provider rollback recreate failed (%w)", err, rollbackErr)
				}
				compensateCtx, compensateCancel := newCompensationCtx()
				defer compensateCancel()
				recreatedProviderRecordID, rollbackErr := retryValue(compensateCtx, singleOperationAttempts, func(c context.Context) (string, error) {
					return client.CreateRecord(c, item.ZoneID, item.RecordType, item.Name, item.Content, item.TTL, item.Proxied)
				})
				if rollbackErr != nil {
					s.logEvent(requestID, "delete_compensate_recreate", false, rollbackErr.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
					return fmt.Errorf("delete local record failed (%v), provider rollback recreate failed (%w)", err, rollbackErr)
				}
				recreatedProviderRecordID = strings.TrimSpace(recreatedProviderRecordID)
				detail := map[string]interface{}{
					"old_provider_record_id": recordID,
					"new_provider_record_id": recreatedProviderRecordID,
				}
				if recreatedProviderRecordID != "" && recreatedProviderRecordID != strings.TrimSpace(item.ProviderRecordID) {
					if updateErr := s.repo.GetDB().Model(&model.DNSRecord{}).
						Where("id = ?", item.RecordID).
						Update("provider_record_id", recreatedProviderRecordID).Error; updateErr != nil {
						s.logEvent(requestID, "delete_compensate_update_provider_id", false, updateErr.Error(), detail, provider, item.ZoneID, item.RecordID, recreatedProviderRecordID)
						return fmt.Errorf("delete local record failed (%v), provider rollback succeeded but provider_record_id update failed (%w)", err, updateErr)
					}
				}
				s.logEvent(requestID, "delete_compensate_recreate", true, "rollback succeeded", detail, provider, item.ZoneID, item.RecordID, recreatedProviderRecordID)
				return fmt.Errorf("delete local record failed after provider delete: %w", err)
			}
			return fmt.Errorf("delete local record failed: %w", err)
		}
	}

	s.logEvent(requestID, "delete_done", true, "ok", nil, provider, item.ZoneID, item.RecordID, recordID)
	return nil
}

func (s *DNSOperationService) providerClientForAccount(accountID uint, providerHint string) (DNSProviderClient, string, error) {
	if accountID == 0 {
		return nil, "", errors.New("provider account id is required")
	}
	account, err := s.repo.FindDNSProviderAccount(accountID)
	if err != nil {
		return nil, "", err
	}
	rawCredentials := strings.TrimSpace(account.Credentials)
	credentials, err := ParseProviderCredentials(account.Provider, crypto.DecryptOrPlaintext(rawCredentials, s.cfg.EncryptionKey))
	if err != nil {
		return nil, "", err
	}

	provider := model.NormalizeProvider(account.Provider)
	if provider == "" {
		provider = model.DNSProviderCloudflare
	}
	if hint := model.NormalizeProvider(providerHint); hint != "" {
		provider = hint
	}
	client, err := BuildProviderClient(provider, credentials)
	if err != nil {
		return nil, "", err
	}
	return client, provider, nil
}

func retryError(ctx context.Context, attempts int, fn func(context.Context) error) error {
	_, err := retryValue(ctx, attempts, func(c context.Context) (struct{}, error) {
		return struct{}{}, fn(c)
	})
	return err
}

func newOperationCtx(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, singleOperationTimeout)
}

func newCompensationCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), compensationOperationTimeout)
}

func withOperationRequestID(parent context.Context, requestID uint) context.Context {
	if requestID == 0 {
		return parent
	}
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, operationRequestIDKey{}, requestID)
}

func operationRequestIDFromCtx(ctx context.Context) *uint {
	if ctx == nil {
		return nil
	}
	value := ctx.Value(operationRequestIDKey{})
	requestID, ok := value.(uint)
	if !ok || requestID == 0 {
		return nil
	}
	id := requestID
	return &id
}

func retryValue[T any](ctx context.Context, attempts int, fn func(context.Context) (T, error)) (T, error) {
	if attempts < 1 {
		attempts = 1
	}
	var zero T
	var lastErr error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		value, err := fn(ctx)
		if err == nil {
			return value, nil
		}
		lastErr = err
		if i == attempts-1 || !shouldRetryProviderError(err) {
			break
		}
		timer := time.NewTimer(time.Duration((i+1)*retryBackoffMilliseconds) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, lastErr
}

func shouldRetryProviderError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := strings.ToLower(err.Error())
	transientTokens := []string{
		"timeout",
		"tempor",
		"connection reset",
		"connection refused",
		"eof",
		"429",
		"500",
		"502",
		"503",
		"504",
		"rate limit",
	}
	for _, token := range transientTokens {
		if strings.Contains(msg, token) {
			return true
		}
	}
	return false
}

func isProviderRecordNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrCloudflareRecordNotFound) ||
		errors.Is(err, ErrDNSPodRecordNotFound) ||
		errors.Is(err, ErrAliDNSRecordNotFound) ||
		errors.Is(err, ErrHuaweiDNSRecordNotFound) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	notFoundTokens := []string{
		"record not found",
		"record does not exist",
		"recordidinvalid",
		"cloudflare record not found",
		"dnspod record not found",
		"aliyun dns record not found",
		"huawei cloud dns record not found",
	}
	for _, token := range notFoundTokens {
		if strings.Contains(msg, token) {
			return true
		}
	}
	return false
}

func (s *DNSOperationService) logEvent(requestID *uint, step string, success bool, message string, detail interface{}, provider, zone string, recordID uint, providerRecordID string) {
	var raw json.RawMessage
	if detail != nil {
		if encoded, err := json.Marshal(detail); err == nil {
			raw = encoded
		}
	}
	_ = s.repo.CreateDNSOperationEvent(&model.DNSOperationEvent{
		RequestID:     requestID,
		Scope:         "dns_operation",
		Step:          step,
		Success:       success,
		Message:       message,
		Detail:        raw,
		Provider:      provider,
		ProviderZone:  zone,
		RecordID:      recordID,
		ProviderRecID: providerRecordID,
	})
}
