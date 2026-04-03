package repository

import (
	"time"

	"hl6-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) EnqueueCloudflareTask(tx *gorm.DB, task *model.CloudflareTask) error {
	db := r.DB
	if tx != nil {
		db = tx
	}

	if task.Status == "" {
		task.Status = model.CloudflareTaskStatusPending
	}
	if task.MaxAttempts <= 0 {
		task.MaxAttempts = 8
	}
	if task.NextRetryAt.IsZero() {
		task.NextRetryAt = time.Now()
	}

	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(task).Error; err != nil {
		return err
	}

	if task.ID != 0 {
		return nil
	}

	var existing model.CloudflareTask
	if err := db.Where("idempotency_key = ?", task.IdempotencyKey).First(&existing).Error; err != nil {
		return err
	}
	*task = existing
	return nil
}

func (r *Repository) AcquireDueCloudflareTasks(limit int) ([]model.CloudflareTask, error) {
	if limit <= 0 {
		limit = 1
	}

	var tasks []model.CloudflareTask
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ? AND next_retry_at <= ?", []string{model.CloudflareTaskStatusPending, model.CloudflareTaskStatusRetry}, now).
			Order("next_retry_at ASC, id ASC").
			Limit(limit).
			Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(tasks))
		for i := range tasks {
			ids = append(ids, tasks[i].ID)
		}

		if err := tx.Model(&model.CloudflareTask{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":     model.CloudflareTaskStatusRunning,
				"attempts":   gorm.Expr("attempts + 1"),
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		for i := range tasks {
			tasks[i].Status = model.CloudflareTaskStatusRunning
			tasks[i].Attempts++
		}
		return nil
	})
	return tasks, err
}

func (r *Repository) RequeueStaleRunningCloudflareTasks(staleAfter time.Duration) error {
	if staleAfter <= 0 {
		staleAfter = 5 * time.Minute
	}
	deadline := time.Now().Add(-staleAfter)
	return r.DB.Model(&model.CloudflareTask{}).
		Where("status = ? AND updated_at < ?", model.CloudflareTaskStatusRunning, deadline).
		Updates(map[string]interface{}{
			"status":        model.CloudflareTaskStatusRetry,
			"next_retry_at": time.Now(),
			"last_error":    "requeued stale running task",
		}).Error
}

func (r *Repository) ListDeadCloudflareTasks(limit int) ([]model.CloudflareTask, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var tasks []model.CloudflareTask
	err := r.DB.Where("status = ?", model.CloudflareTaskStatusDead).
		Order("updated_at DESC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

func (r *Repository) MarkCloudflareTaskSucceeded(taskID uint) error {
	return r.DB.Model(&model.CloudflareTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        model.CloudflareTaskStatusSucceeded,
			"last_error":    "",
			"next_retry_at": time.Now(),
		}).Error
}

func (r *Repository) MarkCloudflareTaskRetry(taskID uint, lastError string, nextRetryAt time.Time) error {
	return r.DB.Model(&model.CloudflareTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        model.CloudflareTaskStatusRetry,
			"last_error":    lastError,
			"next_retry_at": nextRetryAt,
		}).Error
}

func (r *Repository) MarkCloudflareTaskDead(taskID uint, lastError string) error {
	return r.DB.Model(&model.CloudflareTask{}).
		Where("id = ?", taskID).
		Updates(map[string]interface{}{
			"status":        model.CloudflareTaskStatusDead,
			"last_error":    lastError,
			"next_retry_at": time.Now(),
		}).Error
}

func (r *Repository) UpdateDNSRecordSyncState(tx *gorm.DB, recordID uint, status string, operationID *uint, syncError string, cloudflareRecordID *string) error {
	db := r.DB
	if tx != nil {
		db = tx
	}

	updates := map[string]interface{}{
		"sync_status":       status,
		"sync_operation_id": operationID,
		"sync_error":        syncError,
	}
	if cloudflareRecordID != nil {
		updates["cloudflare_record_id"] = *cloudflareRecordID
	}

	result := db.Model(&model.DNSRecord{}).
		Where("id = ?", recordID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
