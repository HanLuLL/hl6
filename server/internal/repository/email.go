package repository

import (
	"hl6-server/internal/model"

	"gorm.io/gorm"
)

// CreateEmailLog 创建邮件发送记录。
func (r *Repository) CreateEmailLog(log *model.EmailLog) error {
	return r.DB.Create(log).Error
}

// UpdateEmailLog 更新邮件发送记录。
func (r *Repository) UpdateEmailLog(log *model.EmailLog) error {
	return r.DB.Save(log).Error
}

// ListEmailLogs 分页查询邮件发送记录。
func (r *Repository) ListEmailLogs(page, perPage int, emailType, status string) ([]model.EmailLog, int64, error) {
	var logs []model.EmailLog
	var total int64

	query := r.DB.Model(&model.EmailLog{})
	if emailType != "" {
		query = query.Where("email_type = ?", emailType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// FindEmailLog 根据 ID 查找邮件记录。
func (r *Repository) FindEmailLog(id uint) (*model.EmailLog, error) {
	var log model.EmailLog
	if err := r.DB.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// ListFailedEmailLogs 查询重试次数未超限的失败邮件。
func (r *Repository) ListFailedEmailLogs(maxRetries int) ([]model.EmailLog, error) {
	var logs []model.EmailLog
	if err := r.DB.Where("status = ? AND retry_count < ?", model.EmailStatusFailed, maxRetries).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// DeleteOldEmailLogs 删除超过指定天数的邮件日志。
func (r *Repository) DeleteOldEmailLogs(olderThanDays int) (int64, error) {
	result := r.DB.Exec(`
		DELETE FROM email_logs
		WHERE created_at < NOW() - INTERVAL '1 day' * ?
	`, olderThanDays)
	return result.RowsAffected, result.Error
}
