package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"

	"gorm.io/gorm"
)

type CloudflareTaskWorker struct {
	repo         *repository.Repository
	cfg          *config.Config
	pollInterval time.Duration
	batchSize    int
}

func NewCloudflareTaskWorker(repo *repository.Repository, cfg *config.Config, pollInterval time.Duration, batchSize int) *CloudflareTaskWorker {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 20
	}
	return &CloudflareTaskWorker{
		repo:         repo,
		cfg:          cfg,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

func (w *CloudflareTaskWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *CloudflareTaskWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		w.processBatch(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *CloudflareTaskWorker) processBatch(ctx context.Context) {
	if err := w.repo.RequeueStaleRunningCloudflareTasks(5 * time.Minute); err != nil {
		log.Printf("cloudflare task worker requeue stale tasks failed: %v", err)
	}

	tasks, err := w.repo.AcquireDueCloudflareTasks(w.batchSize)
	if err != nil {
		log.Printf("cloudflare task worker acquire failed: %v", err)
		return
	}
	for i := range tasks {
		w.processOne(ctx, tasks[i])
	}
}

func (w *CloudflareTaskWorker) processOne(ctx context.Context, task model.CloudflareTask) {
	payload, err := parseTaskPayload(task.Payload)
	if err != nil {
		w.markTaskDead(task, err)
		return
	}

	err = w.executeTask(ctx, task, payload)
	if err == nil {
		if markErr := w.repo.MarkCloudflareTaskSucceeded(task.ID); markErr != nil {
			log.Printf("cloudflare task %d mark succeeded failed: %v", task.ID, markErr)
			nextRetryAt := time.Now().Add(computeRetryDelay(task.Attempts))
			_ = w.repo.MarkCloudflareTaskRetry(task.ID, truncateError(markErr.Error(), 1000), nextRetryAt)
		}
		return
	}

	errMsg := truncateError(err.Error(), 1000)
	dead := task.Attempts >= task.MaxAttempts
	if dead {
		w.markTaskDead(task, err)
		return
	}
	w.markDNSRecordSyncFailed(task, errMsg, false)

	nextRetryAt := time.Now().Add(computeRetryDelay(task.Attempts))
	if markErr := w.repo.MarkCloudflareTaskRetry(task.ID, errMsg, nextRetryAt); markErr != nil {
		log.Printf("cloudflare task %d mark retry failed: %v", task.ID, markErr)
	}
}

func (w *CloudflareTaskWorker) executeTask(ctx context.Context, task model.CloudflareTask, payload *model.CloudflareTaskPayload) error {
	cf, err := w.cloudflareForAccount(payload.CloudflareAccountID)
	if err != nil {
		return err
	}

	switch task.Action {
	case model.CloudflareTaskActionCreateDNSRecord:
		record, shouldRun, err := w.shouldExecuteDNSRecordTask(task)
		if err != nil {
			return err
		}
		if !shouldRun {
			return nil
		}
		cfID, err := cf.CreateRecord(
			ctx,
			payload.ZoneID,
			payload.RecordType,
			payload.Name,
			payload.Content,
			payload.TTL,
			payload.Proxied,
		)
		if err != nil {
			return err
		}
		if err := w.repo.UpdateDNSRecordSyncState(nil, task.ResourceID, "synced", nil, "", &cfID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Record was deleted locally after CF create; cleanup remote side to avoid orphan records.
				if delErr := cf.DeleteRecord(ctx, payload.ZoneID, cfID); delErr != nil {
					return fmt.Errorf("record deleted locally after create, cleanup failed: %w", delErr)
				}
				return nil
			}
			return err
		}
		record.CloudflareRecordID = cfID
		return nil
	case model.CloudflareTaskActionUpdateDNSRecord:
		record, shouldRun, err := w.shouldExecuteDNSRecordTask(task)
		if err != nil {
			return err
		}
		if !shouldRun {
			return nil
		}
		recordID := strings.TrimSpace(payload.RecordID)
		if recordID == "" && record != nil {
			recordID = strings.TrimSpace(record.CloudflareRecordID)
		}
		if recordID == "" {
			return errors.New("cloudflare record id is empty")
		}
		if err := cf.UpdateRecord(
			ctx,
			payload.ZoneID,
			recordID,
			payload.RecordType,
			payload.Name,
			payload.Content,
			payload.TTL,
			payload.Proxied,
		); err != nil {
			return err
		}
		if err := w.repo.UpdateDNSRecordSyncState(nil, task.ResourceID, "synced", nil, "", &recordID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Local record disappeared while update was executing; delete remote record for consistency.
				if delErr := cf.DeleteRecord(ctx, payload.ZoneID, recordID); delErr != nil {
					return fmt.Errorf("record deleted locally after update, cleanup failed: %w", delErr)
				}
				return nil
			}
			return err
		}
		return nil
	case model.CloudflareTaskActionDeleteDNSRecord:
		recordID := strings.TrimSpace(payload.RecordID)
		if recordID == "" {
			if strings.TrimSpace(payload.RecordType) == "" || strings.TrimSpace(payload.Name) == "" || strings.TrimSpace(payload.Content) == "" {
				return nil
			}
			existingID, err := cf.FindRecord(ctx, payload.ZoneID, payload.RecordType, payload.Name, payload.Content)
			if err != nil {
				if errors.Is(err, ErrCloudflareRecordNotFound) {
					return nil
				}
				return err
			}
			recordID = existingID
		}
		return cf.DeleteRecord(ctx, payload.ZoneID, recordID)
	default:
		return fmt.Errorf("unsupported cloudflare task action %q", task.Action)
	}
}

func (w *CloudflareTaskWorker) shouldExecuteDNSRecordTask(task model.CloudflareTask) (*model.DNSRecord, bool, error) {
	if task.ResourceType != "dns_record" {
		return nil, true, nil
	}

	record, err := w.repo.FindDNSRecord(task.ResourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if record.SyncOperationID == nil || *record.SyncOperationID != task.ID {
		return record, false, nil
	}
	if record.SyncStatus == "pending_delete" || record.SyncStatus == "sync_dead" {
		return record, false, nil
	}
	return record, true, nil
}

func (w *CloudflareTaskWorker) cloudflareForAccount(accountID uint) (*CloudflareService, error) {
	if accountID == 0 {
		return nil, errors.New("cloudflare account id is required")
	}
	account, err := w.repo.FindCloudflareAccount(accountID)
	if err != nil {
		return nil, err
	}
	token := crypto.DecryptOrPlaintext(account.ApiToken, w.cfg.EncryptionKey)
	return NewCloudflareService(token)
}

func parseTaskPayload(raw json.RawMessage) (*model.CloudflareTaskPayload, error) {
	var payload model.CloudflareTaskPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if payload.CloudflareAccountID == 0 {
		return nil, errors.New("cloudflare_account_id is required")
	}
	if strings.TrimSpace(payload.ZoneID) == "" {
		return nil, errors.New("zone_id is required")
	}
	return &payload, nil
}

func (w *CloudflareTaskWorker) markTaskDead(task model.CloudflareTask, err error) {
	errMsg := truncateError(err.Error(), 1000)
	w.markDNSRecordSyncFailed(task, errMsg, true)
	if markErr := w.repo.MarkCloudflareTaskDead(task.ID, errMsg); markErr != nil {
		log.Printf("cloudflare task %d mark dead failed: %v", task.ID, markErr)
	}
}

func (w *CloudflareTaskWorker) markDNSRecordSyncFailed(task model.CloudflareTask, errMsg string, dead bool) {
	if task.ResourceType != "dns_record" {
		return
	}
	if task.Action == model.CloudflareTaskActionDeleteDNSRecord {
		return
	}
	opID := task.ID
	syncStatus := "sync_retry"
	if dead {
		syncStatus = "sync_dead"
	}
	_ = w.repo.UpdateDNSRecordSyncState(nil, task.ResourceID, syncStatus, &opID, errMsg, nil)
}

func computeRetryDelay(attempts int) time.Duration {
	if attempts < 1 {
		return 2 * time.Second
	}
	seconds := 1 << (attempts - 1)
	if seconds > 1800 {
		seconds = 1800
	}
	return time.Duration(seconds) * time.Second
}

func truncateError(msg string, maxLen int) string {
	trimmed := strings.TrimSpace(msg)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen]
}
