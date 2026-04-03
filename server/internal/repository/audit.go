package repository

import "hl6-server/internal/model"

func (r *Repository) CreateAuditLog(log *model.AuditLog) error {
	return r.DB.Create(log).Error
}

func (r *Repository) ListAuditLogs(page, perPage int, operator, action string) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	q := r.DB.Model(&model.AuditLog{})

	if operator != "" {
		like := "%" + escapeLike(operator) + "%"
		q = q.Joins("LEFT JOIN users ON users.id = audit_logs.user_id").
			Where("users.email ILIKE ?", like)
	}

	if action != "" {
		like := "%" + escapeLike(action) + "%"
		q = q.Where("audit_logs.action ILIKE ?", like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset((page - 1) * perPage).Limit(perPage).Order("audit_logs.created_at DESC").Preload("User").Find(&logs).Error
	return logs, total, err
}
