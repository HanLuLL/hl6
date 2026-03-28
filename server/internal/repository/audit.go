package repository

import "hl6-server/internal/model"

func (r *Repository) CreateAuditLog(log *model.AuditLog) error {
	return r.DB.Create(log).Error
}

func (r *Repository) ListAuditLogs(page, perPage int, search ...string) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	q := r.DB.Model(&model.AuditLog{})
	if len(search) > 0 && search[0] != "" {
		like := "%" + escapeLike(search[0]) + "%"
		q = q.Where("action ILIKE ? OR resource ILIKE ?", like, like)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Preload("User").Find(&logs).Error
	return logs, total, err
}
