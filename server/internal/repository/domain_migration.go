package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"hl6-server/internal/model"
)

// CreateMigrationTask creates a new migration task and assigns the next queue_seq
// for the domain atomically. Uses FOR UPDATE to serialize concurrent creates.
func (r *Repository) CreateMigrationTask(task *model.DomainDNSMigrationTask) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the domain row to serialize queue_seq assignment
		var domain model.Domain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", task.DomainID).First(&domain).Error; err != nil {
			return fmt.Errorf("lock domain for migration: %w", err)
		}

		// Count existing tasks to get next sequence number
		var maxSeq int64
		tx.Model(&model.DomainDNSMigrationTask{}).
			Where("domain_id = ?", task.DomainID).
			Select("COALESCE(MAX(queue_seq), 0)").Scan(&maxSeq)
		task.QueueSeq = maxSeq + 1

		return tx.Create(task).Error
	})
}

// FindMigrationTask finds a migration task by ID.
func (r *Repository) FindMigrationTask(taskID uint) (*model.DomainDNSMigrationTask, error) {
	var task model.DomainDNSMigrationTask
	if err := r.DB.Preload("Items").First(&task, taskID).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListMigrationTasks lists migration tasks for a domain with optional status filter and pagination.
func (r *Repository) ListMigrationTasks(domainID uint, status string, page, perPage int) ([]model.DomainDNSMigrationTask, int64, error) {
	var tasks []model.DomainDNSMigrationTask
	var total int64

	q := r.DB.Model(&model.DomainDNSMigrationTask{}).Where("domain_id = ?", domainID)
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	if err := q.Order("queue_seq DESC").Offset(offset).Limit(perPage).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// ListMigrationItems lists items for a task with pagination.
func (r *Repository) ListMigrationItems(taskID uint, page, perPage int) ([]model.DomainDNSMigrationItem, int64, error) {
	var items []model.DomainDNSMigrationItem
	var total int64

	q := r.DB.Model(&model.DomainDNSMigrationItem{}).Where("task_id = ?", taskID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	if err := q.Order("id ASC").Offset(offset).Limit(perPage).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListFailedMigrationItems returns all failed items for a given task.
func (r *Repository) ListFailedMigrationItems(taskID uint) ([]model.DomainDNSMigrationItem, error) {
	var items []model.DomainDNSMigrationItem
	if err := r.DB.Where("task_id = ? AND status = ?", taskID, model.MigrationItemStatusFailed).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// TryStartMigrationTask atomically transitions a pending task to running for a domain
// if no other task is already running. Returns true if successfully started.
func (r *Repository) TryStartMigrationTask(taskID, domainID uint) (bool, error) {
	startedAt := time.Now()
	result := r.DB.Model(&model.DomainDNSMigrationTask{}).
		Where("id = ? AND domain_id = ? AND status = ?", taskID, domainID, model.MigrationTaskStatusPending).
		Updates(map[string]interface{}{
			"status":     model.MigrationTaskStatusRunning,
			"started_at": &startedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// UpdateMigrationTask updates arbitrary fields on a migration task.
func (r *Repository) UpdateMigrationTask(taskID uint, updates map[string]interface{}) error {
	return r.DB.Model(&model.DomainDNSMigrationTask{}).Where("id = ?", taskID).Updates(updates).Error
}

// UpdateMigrationItem updates arbitrary fields on a migration item.
func (r *Repository) UpdateMigrationItem(itemID uint, updates map[string]interface{}) error {
	return r.DB.Model(&model.DomainDNSMigrationItem{}).Where("id = ?", itemID).Updates(updates).Error
}

// CreateMigrationItems bulk-inserts migration items for a task.
func (r *Repository) CreateMigrationItems(items []model.DomainDNSMigrationItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.DB.Create(&items).Error
}

// FindRunningMigrationTask returns the currently running migration task for a domain, if any.
func (r *Repository) FindRunningMigrationTask(domainID uint) (*model.DomainDNSMigrationTask, error) {
	var task model.DomainDNSMigrationTask
	err := r.DB.Where("domain_id = ? AND status = ?", domainID, model.MigrationTaskStatusRunning).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// FindNextPendingMigrationTask returns the next pending task for a domain (lowest queue_seq).
func (r *Repository) FindNextPendingMigrationTask(domainID uint) (*model.DomainDNSMigrationTask, error) {
	var task model.DomainDNSMigrationTask
	err := r.DB.Where("domain_id = ? AND status = ?", domainID, model.MigrationTaskStatusPending).
		Order("queue_seq ASC").First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// FindPendingMigrationTasksAll returns all pending migration tasks (for worker resumption).
func (r *Repository) FindPendingMigrationTasksAll(limit int) ([]model.DomainDNSMigrationTask, error) {
	var tasks []model.DomainDNSMigrationTask
	if err := r.DB.Where("status = ?", model.MigrationTaskStatusPending).
		Order("id ASC").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// ResetRunningMigrationTasksToPending resets any tasks stuck in running back to pending
// (called on server startup to recover from crash).
func (r *Repository) ResetRunningMigrationTasksToPending() error {
	return r.DB.Model(&model.DomainDNSMigrationTask{}).
		Where("status = ?", model.MigrationTaskStatusRunning).
		Updates(map[string]interface{}{
			"status":     model.MigrationTaskStatusPending,
			"started_at": nil,
		}).Error
}

// UpdateDomainMigrationState updates the migration-related fields on a domain.
func (r *Repository) UpdateDomainMigrationState(domainID uint, state string, readOnly bool, lastTaskID *uint) error {
	updates := map[string]interface{}{
		"migration_state":        state,
		"migration_read_only":    readOnly,
		"last_migration_task_id": lastTaskID,
	}
	return r.DB.Model(&model.Domain{}).Where("id = ?", domainID).Updates(updates).Error
}

// CountMigrationQueueForDomain returns the number of pending+running tasks for a domain.
func (r *Repository) CountMigrationQueueForDomain(domainID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.DomainDNSMigrationTask{}).
		Where("domain_id = ? AND status IN ?", domainID, []string{
			model.MigrationTaskStatusPending,
			model.MigrationTaskStatusRunning,
		}).Count(&count).Error
	return count, err
}
