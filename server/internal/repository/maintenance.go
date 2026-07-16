package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
)

func (r *Repository) CreateDatabaseBackup(backup *model.DatabaseBackup) error {
	if backup == nil || backup.CreatedByUserID == 0 || backup.Filename == "" || backup.ChecksumSHA256 == "" || backup.StoragePath == "" {
		return ErrInvalidNewUserInput
	}
	return r.DB.Create(backup).Error
}

func (r *Repository) FindDatabaseBackup(id uint) (*model.DatabaseBackup, error) {
	var backup model.DatabaseBackup
	if err := r.DB.First(&backup, id).Error; err != nil {
		return nil, err
	}
	return &backup, nil
}

func (r *Repository) CreateDatabaseRestoreJob(job *model.DatabaseRestoreJob) error {
	if job == nil || job.CreatedByUserID == 0 || job.ChallengeHash == "" {
		return ErrInvalidNewUserInput
	}
	return r.DB.Create(job).Error
}

func (r *Repository) UpdateDatabaseRestoreJob(job *model.DatabaseRestoreJob) error {
	if job == nil || job.ID == 0 {
		return ErrInvalidNewUserInput
	}
	return r.DB.Save(job).Error
}

func (r *Repository) ListDatabaseRestoreJobs(page, perPage int) ([]model.DatabaseRestoreJob, int64, error) {
	var jobs []model.DatabaseRestoreJob
	var total int64
	query := r.DB.Model(&model.DatabaseRestoreJob{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (r *Repository) MarkDatabaseRestoreJobRunning(job *model.DatabaseRestoreJob, backupID uint) error {
	if job == nil || job.ID == 0 || backupID == 0 {
		return ErrInvalidNewUserInput
	}
	now := time.Now().UTC()
	job.Status = model.DatabaseRestoreStatusRunning
	job.PreRestoreBackupID = &backupID
	job.StartedAt = &now
	result := r.DB.Model(&model.DatabaseRestoreJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"status":                job.Status,
		"pre_restore_backup_id": backupID,
		"started_at":            now,
		"failure_detail":        "",
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("database restore job no longer exists")
	}
	return nil
}

func (r *Repository) FinishDatabaseRestoreJob(job *model.DatabaseRestoreJob, status, validationResult, failureDetail string) error {
	if job == nil || job.ID == 0 {
		return ErrInvalidNewUserInput
	}
	if status != model.DatabaseRestoreStatusSucceeded && status != model.DatabaseRestoreStatusFailed {
		return errors.New("invalid database restore status")
	}
	now := time.Now().UTC()
	job.Status = status
	job.ValidationResult = validationResult
	job.FailureDetail = failureDetail
	job.FinishedAt = &now
	result := r.DB.Model(&model.DatabaseRestoreJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"status":            status,
		"validation_result": validationResult,
		"failure_detail":    failureDetail,
		"finished_at":       now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("database restore job no longer exists")
	}
	return nil
}

func (r *Repository) DeleteExpiredDatabaseBackups(before time.Time) error {
	return r.DB.Where("expires_at IS NOT NULL AND expires_at < ?", before).Delete(&model.DatabaseBackup{}).Error
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
