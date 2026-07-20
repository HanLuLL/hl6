package repository

import (
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
)

// SystemLogFilter defines filter criteria for querying system logs.
type SystemLogFilter struct {
	Levels    []string
	Modules   []string
	Search    string
	From      *time.Time
	To        *time.Time
	UserID    *uint
	IPAddress string
}

// CreateSystemLog creates a new system log entry.
func (r *Repository) CreateSystemLog(log *model.SystemLog) error {
	return r.DB.Create(log).Error
}

// CreateSystemLogTx creates a new system log entry within a transaction.
func (r *Repository) CreateSystemLogTx(tx *gorm.DB, log *model.SystemLog) error {
	if tx == nil {
		return r.CreateSystemLog(log)
	}
	return tx.Create(log).Error
}

// ListSystemLogs retrieves system logs with pagination and filtering.
func (r *Repository) ListSystemLogs(page, perPage int, filter SystemLogFilter) ([]model.SystemLog, int64, error) {
	var logs []model.SystemLog
	var total int64

	q := r.DB.Model(&model.SystemLog{})

	// Apply level filter
	if len(filter.Levels) > 0 {
		q = q.Where("level IN ?", filter.Levels)
	}

	// Apply module filter
	if len(filter.Modules) > 0 {
		q = q.Where("module IN ?", filter.Modules)
	}

	// Apply search filter (searches in message and module)
	if filter.Search != "" {
		like := "%" + escapeLike(filter.Search) + "%"
		q = q.Where("message ILIKE ? OR module ILIKE ?", like, like)
	}

	// Apply time range filter
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}

	// Get total count
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}

	err := q.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&logs).Error
	if logs == nil {
		logs = []model.SystemLog{}
	}

	return logs, total, err
}

// GetSystemLog retrieves a single system log by ID.
func (r *Repository) GetSystemLog(id uint) (*model.SystemLog, error) {
	var log model.SystemLog
	err := r.DB.First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// DeleteSystemLogsOlderThan deletes system logs older than the specified time.
func (r *Repository) DeleteSystemLogsOlderThan(t time.Time) (int64, error) {
	result := r.DB.Where("created_at < ?", t).Delete(&model.SystemLog{})
	return result.RowsAffected, result.Error
}

// GetSystemLogModules retrieves distinct modules from system logs.
func (r *Repository) GetSystemLogModules() ([]string, error) {
	var modules []string
	err := r.DB.Model(&model.SystemLog{}).
		Distinct("module").
		Order("module ASC").
		Pluck("module", &modules).Error
	return modules, err
}

// GetSystemLogStats returns statistics about system logs.
func (r *Repository) GetSystemLogStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count by level
	type levelCount struct {
		Level string
		Count int64
	}
	var levelCounts []levelCount
	if err := r.DB.Model(&model.SystemLog{}).
		Select("level, COUNT(*) as count").
		Group("level").
		Scan(&levelCounts).Error; err != nil {
		return nil, err
	}
	for _, lc := range levelCounts {
		stats["level_"+lc.Level] = lc.Count
	}

	// Total count
	var total int64
	if err := r.DB.Model(&model.SystemLog{}).Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	// Count today
	today := time.Now().Truncate(24 * time.Hour)
	var todayCount int64
	if err := r.DB.Model(&model.SystemLog{}).
		Where("created_at >= ?", today).
		Count(&todayCount).Error; err != nil {
		return nil, err
	}
	stats["today"] = todayCount

	return stats, nil
}