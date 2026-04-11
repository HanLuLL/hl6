package repository

import (
	"hl6-server/internal/model"
	"time"

	"gorm.io/gorm"
)

func (r *Repository) CreateDNSBulkJobWithItems(job *model.DNSBulkJob, items []model.DNSBulkJobItem) error {
	if job == nil {
		return gorm.ErrInvalidData
	}
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			items[i].JobID = job.ID
			if items[i].Status == "" {
				items[i].Status = model.DNSBulkJobItemStatusPending
			}
		}
		return tx.Create(&items).Error
	})
}

func (r *Repository) FindDNSBulkJob(id uint) (*model.DNSBulkJob, error) {
	var job model.DNSBulkJob
	if err := r.DB.First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *Repository) TryStartDNSBulkJob(id uint, startedAt time.Time) (bool, error) {
	res := r.DB.Model(&model.DNSBulkJob{}).
		Where("id = ? AND status = ?", id, model.DNSBulkJobStatusPending).
		Updates(map[string]interface{}{
			"status":     model.DNSBulkJobStatusRunning,
			"started_at": &startedAt,
			"message":    "running",
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *Repository) ResetRunningDNSBulkJobsToPending() error {
	return r.DB.Model(&model.DNSBulkJob{}).
		Where("status = ?", model.DNSBulkJobStatusRunning).
		Updates(map[string]interface{}{
			"status":  model.DNSBulkJobStatusPending,
			"message": "rescheduled after restart",
		}).Error
}

func (r *Repository) ListPendingDNSBulkJobs(limit int) ([]model.DNSBulkJob, error) {
	if limit <= 0 {
		limit = 100
	}
	var jobs []model.DNSBulkJob
	err := r.DB.Where("status = ?", model.DNSBulkJobStatusPending).
		Order("id ASC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

func (r *Repository) UpdateDNSBulkJob(id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.DB.Model(&model.DNSBulkJob{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) UpdateDNSBulkJobItem(id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return r.DB.Model(&model.DNSBulkJobItem{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) ListAllDNSBulkJobItems(jobID uint) ([]model.DNSBulkJobItem, error) {
	var items []model.DNSBulkJobItem
	err := r.DB.Where("job_id = ?", jobID).Order("id ASC").Find(&items).Error
	return items, err
}

func (r *Repository) ListDNSBulkJobItems(jobID uint, page, perPage int, status string) ([]model.DNSBulkJobItem, int64, error) {
	var (
		items []model.DNSBulkJobItem
		total int64
	)
	q := r.DB.Model(&model.DNSBulkJobItem{}).Where("job_id = ?", jobID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id ASC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&items).Error
	if items == nil {
		items = []model.DNSBulkJobItem{}
	}
	return items, total, err
}
