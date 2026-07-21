package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
)

// RealnameListFilter 实名申请列表筛选条件。
type RealnameListFilter struct {
	Statuses []string
	UserID   *uint
	Provider string
	From     *time.Time
	To       *time.Time
}

// CreateRealnameApplication 创建实名申请单。
func (r *Repository) CreateRealnameApplication(app *model.RealnameApplication) error {
	return r.DB.Create(app).Error
}

// UpdateRealnameApplication 保存申请单全量字段。
func (r *Repository) UpdateRealnameApplication(app *model.RealnameApplication) error {
	return r.DB.Save(app).Error
}

// UpdateRealnameApplicationStatus 仅更新状态和拒绝原因等少量字段。
func (r *Repository) UpdateRealnameApplicationStatus(id uint, status, rejectReason string) error {
	updates := map[string]interface{}{
		"status":         status,
		"reject_reason":  rejectReason,
		"updated_at":     time.Now(),
	}
	return r.DB.Model(&model.RealnameApplication{}).Where("id = ?", id).Updates(updates).Error
}

// FindRealnameApplication 按 ID 查询申请单。
func (r *Repository) FindRealnameApplication(id uint) (*model.RealnameApplication, error) {
	var app model.RealnameApplication
	err := r.DB.First(&app, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &app, err
}

// FindRealnameApplicationForUpdate 加锁查询（事务内调用）。
func (r *Repository) FindRealnameApplicationForUpdate(tx *gorm.DB, id uint) (*model.RealnameApplication, error) {
	var app model.RealnameApplication
	err := tx.Set("gorm:query_option", "FOR UPDATE").First(&app, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &app, err
}

// FindRealnameApplicationByOrder 通过 PaymentOrder.ID 反查申请单。
func (r *Repository) FindRealnameApplicationByOrder(orderID uint) (*model.RealnameApplication, error) {
	var app model.RealnameApplication
	err := r.DB.Where("order_id = ?", orderID).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &app, err
}

// FindLatestRealnameApplication 查询用户最近一条申请单。
func (r *Repository) FindLatestRealnameApplication(userID uint) (*model.RealnameApplication, error) {
	var app model.RealnameApplication
	err := r.DB.Where("user_id = ?", userID).Order("created_at DESC").First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &app, err
}

// ListRealnameApplicationsByUser 查询用户全部申请单历史（分页）。
func (r *Repository) ListRealnameApplicationsByUser(userID uint, page, perPage int) ([]model.RealnameApplication, int64, error) {
	q := r.DB.Model(&model.RealnameApplication{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	var apps []model.RealnameApplication
	err := q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&apps).Error
	if apps == nil {
		apps = []model.RealnameApplication{}
	}
	return apps, total, err
}

// HasPendingRealnameApplication 判断用户是否存在未终态的申请单
// （pending_payment/paid/verifying 视为未终态，可避免重复申请）。
func (r *Repository) HasPendingRealnameApplication(userID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.RealnameApplication{}).
		Where("user_id = ? AND status IN ?", userID, []string{
			model.RealnameAppStatusPendingPayment,
			model.RealnameAppStatusPaid,
			model.RealnameAppStatusVerifying,
		}).Count(&count).Error
	return count > 0, err
}

// AdminListRealnameApplications 管理员列表查询。
func (r *Repository) AdminListRealnameApplications(page, perPage int, filter RealnameListFilter) ([]model.RealnameApplication, int64, error) {
	q := r.DB.Model(&model.RealnameApplication{})
	if len(filter.Statuses) > 0 {
		q = q.Where("status IN ?", filter.Statuses)
	}
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.Provider != "" {
		q = q.Where("provider = ?", filter.Provider)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	var apps []model.RealnameApplication
	err := q.Preload("User").Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&apps).Error
	if apps == nil {
		apps = []model.RealnameApplication{}
	}
	return apps, total, err
}

// UpdateUserRealnameStatus 更新用户的实名状态字段（事务内调用）。
func (r *Repository) UpdateUserRealnameStatus(tx *gorm.DB, userID uint, status, realnameName string, verifiedAt *time.Time) error {
	updates := map[string]interface{}{
		"realname_status":      status,
		"realname_name":        realnameName,
		"updated_at":           time.Now(),
	}
	if verifiedAt != nil {
		updates["realname_verified_at"] = *verifiedAt
	}
	return tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

// CountRealnameApplicationsByStatus 按状态统计申请数。
func (r *Repository) CountRealnameApplicationsByStatus() (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	err := r.DB.Model(&model.RealnameApplication{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Status] = r.Count
	}
	return out, nil
}

// CountVerifiedRealnameUsers 统计已实名用户数。
func (r *Repository) CountVerifiedRealnameUsers() (int64, error) {
	var count int64
	err := r.DB.Model(&model.User{}).Where("realname_status = ?", model.RealnameStatusVerified).Count(&count).Error
	return count, err
}
