package repository

import (
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
)

// ---- AI Model Config ----

// ListAIModelConfigs 返回所有 AI 模型配置。
func (r *Repository) ListAIModelConfigs() ([]model.AIModelConfig, error) {
	var configs []model.AIModelConfig
	err := r.DB.Order("id ASC").Find(&configs).Error
	return configs, err
}

// FindAIModelConfig 按 ID 查找 AI 模型配置。
func (r *Repository) FindAIModelConfig(id uint) (*model.AIModelConfig, error) {
	var config model.AIModelConfig
	err := r.DB.First(&config, id).Error
	return &config, err
}

// GetDefaultAIModelConfig 获取默认且启用的 AI 模型配置。
func (r *Repository) GetDefaultAIModelConfig() (*model.AIModelConfig, error) {
	var config model.AIModelConfig
	err := r.DB.Where("is_default = ? AND is_enabled = ?", true, true).First(&config).Error
	return &config, err
}

// CreateAIModelConfig 创建 AI 模型配置。
func (r *Repository) CreateAIModelConfig(config *model.AIModelConfig) error {
	return r.DB.Create(config).Error
}

// UpdateAIModelConfig 更新 AI 模型配置。
func (r *Repository) UpdateAIModelConfig(config *model.AIModelConfig) error {
	return r.DB.Save(config).Error
}

// DeleteAIModelConfig 删除 AI 模型配置。
func (r *Repository) DeleteAIModelConfig(id uint) error {
	return r.DB.Delete(&model.AIModelConfig{}, id).Error
}

// ClearDefaultAIModelConfigs 清除所有默认标记（切换默认模型时使用）。
func (r *Repository) ClearDefaultAIModelConfigs() error {
	return r.DB.Model(&model.AIModelConfig{}).Where("is_default = ?", true).Update("is_default", false).Error
}

// ---- Prompt Template ----

// ListPromptTemplates 返回所有提示词模板，按 sort_order 排序。
func (r *Repository) ListPromptTemplates() ([]model.AuditPromptTemplate, error) {
	var templates []model.AuditPromptTemplate
	err := r.DB.Order("sort_order ASC, id ASC").Find(&templates).Error
	return templates, err
}

// FindPromptTemplate 按 ID 查找提示词模板。
func (r *Repository) FindPromptTemplate(id uint) (*model.AuditPromptTemplate, error) {
	var t model.AuditPromptTemplate
	err := r.DB.First(&t, id).Error
	return &t, err
}

// GetHighestPriorityPromptTemplate 获取最高优先级（sort_order 最小）的启用模板。
func (r *Repository) GetHighestPriorityPromptTemplate() (*model.AuditPromptTemplate, error) {
	var t model.AuditPromptTemplate
	err := r.DB.Where("is_enabled = ?", true).Order("sort_order ASC, id ASC").First(&t).Error
	return &t, err
}

// CreatePromptTemplate 创建提示词模板。
func (r *Repository) CreatePromptTemplate(t *model.AuditPromptTemplate) error {
	return r.DB.Create(t).Error
}

// UpdatePromptTemplate 更新提示词模板。
func (r *Repository) UpdatePromptTemplate(t *model.AuditPromptTemplate) error {
	return r.DB.Save(t).Error
}

// DeletePromptTemplate 删除提示词模板。
func (r *Repository) DeletePromptTemplate(id uint) error {
	return r.DB.Delete(&model.AuditPromptTemplate{}, id).Error
}

// CountPromptTemplates 统计提示词模板数量。
func (r *Repository) CountPromptTemplates() (int64, error) {
	var count int64
	err := r.DB.Model(&model.AuditPromptTemplate{}).Count(&count).Error
	return count, err
}

// ---- AI Review ----

// CreateAuditAIReview 创建 AI 审查记录。
func (r *Repository) CreateAuditAIReview(review *model.AuditAIReview) error {
	return r.DB.Create(review).Error
}

// UpdateAuditAIReviewFinalAction 更新审查记录的最终处置。
func (r *Repository) UpdateAuditAIReviewFinalAction(id uint, action string) error {
	return r.DB.Model(&model.AuditAIReview{}).Where("id = ?", id).Update("final_action", action).Error
}

// ListAuditAIReviews 分页查询 AI 审查记录。
func (r *Repository) ListAuditAIReviews(page, perPage int, filter AuditAIReviewFilter) ([]model.AuditAIReview, int64, error) {
	q := r.DB.Model(&model.AuditAIReview{})
	if len(filter.Judgments) > 0 {
		q = q.Where("ai_judgment IN ?", filter.Judgments)
	}
	if len(filter.AdminStatuses) > 0 {
		q = q.Where("admin_review_status IN ?", filter.AdminStatuses)
	}
	if filter.FQDN != "" {
		q = q.Where("fqdn ILIKE ?", "%"+filter.FQDN+"%")
	}
	if !filter.From.IsZero() {
		q = q.Where("created_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		q = q.Where("created_at <= ?", filter.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reviews []model.AuditAIReview
	offset := (page - 1) * perPage
	err := q.Order("id DESC").Offset(offset).Limit(perPage).Find(&reviews).Error
	return reviews, total, err
}

// FindAuditAIReview 按 ID 查找 AI 审查记录。
func (r *Repository) FindAuditAIReview(id uint) (*model.AuditAIReview, error) {
	var review model.AuditAIReview
	err := r.DB.First(&review, id).Error
	return &review, err
}

// UpdateAuditAIReviewAdminReview 更新管理员二次审核状态。
func (r *Repository) UpdateAuditAIReviewAdminReview(id uint, status, note string, reviewerID uint) error {
	now := time.Now()
	return r.DB.Model(&model.AuditAIReview{}).Where("id = ?", id).Updates(map[string]interface{}{
		"admin_review_status": status,
		"admin_reviewed_by":   reviewerID,
		"admin_reviewed_at":   now,
		"admin_note":          note,
	}).Error
}

// AuditAIReviewFilter AI 审查记录筛选条件。
type AuditAIReviewFilter struct {
	Judgments     []string
	AdminStatuses []string
	FQDN          string
	From          time.Time
	To            time.Time
}

// ---- User Appeal ----

// CreateUserAppeal 创建用户申诉。
func (r *Repository) CreateUserAppeal(appeal *model.UserAppeal) error {
	return r.DB.Create(appeal).Error
}

// ListUserAppeals 分页查询申诉记录（管理员）。
func (r *Repository) ListUserAppeals(page, perPage int) ([]model.UserAppeal, int64, error) {
	var total int64
	if err := r.DB.Model(&model.UserAppeal{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var appeals []model.UserAppeal
	offset := (page - 1) * perPage
	err := r.DB.Order("id DESC").Offset(offset).Limit(perPage).Find(&appeals).Error
	return appeals, total, err
}

// ListUserAppealsByUser 查询用户的申诉记录。
func (r *Repository) ListUserAppealsByUser(userID uint) ([]model.UserAppeal, error) {
	var appeals []model.UserAppeal
	err := r.DB.Where("user_id = ?", userID).Order("id DESC").Find(&appeals).Error
	return appeals, err
}

// FindUserAppeal 按 ID 查找申诉。
func (r *Repository) FindUserAppeal(id uint) (*model.UserAppeal, error) {
	var appeal model.UserAppeal
	err := r.DB.First(&appeal, id).Error
	return &appeal, err
}

// UpdateUserAppealReview 更新申诉审核结果。
func (r *Repository) UpdateUserAppealReview(id uint, status, reply string, reviewerID uint) error {
	now := time.Now()
	return r.DB.Model(&model.UserAppeal{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      status,
		"reviewed_by": reviewerID,
		"reviewed_at": now,
		"reply":       reply,
	}).Error
}

// HasPendingAppealByUser 检查用户是否有待处理的申诉。
func (r *Repository) HasPendingAppealByUser(userID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.UserAppeal{}).Where("user_id = ? AND status = ?", userID, model.AppealStatusPending).Count(&count).Error
	return count > 0, err
}

// ---- AI Audit Statistics ----

// GetAIAuditStats 获取 AI 审计统计概览。
func (r *Repository) GetAIAuditStats() (map[string]interface{}, error) {
	var totalReviews int64
	var pendingReviews int64
	var violationCount int64
	var cleanCount int64

	r.DB.Model(&model.AuditAIReview{}).Count(&totalReviews)
	r.DB.Model(&model.AuditAIReview{}).Where("admin_review_status = ?", model.AdminReviewPending).Count(&pendingReviews)
	r.DB.Model(&model.AuditAIReview{}).Where("ai_judgment = ?", model.AIJudgmentViolation).Count(&violationCount)
	r.DB.Model(&model.AuditAIReview{}).Where("ai_judgment = ?", model.AIJudgmentClean).Count(&cleanCount)

	var pendingAppeals int64
	r.DB.Model(&model.UserAppeal{}).Where("status = ?", model.AppealStatusPending).Count(&pendingAppeals)

	return map[string]interface{}{
		"total_reviews":    totalReviews,
		"pending_reviews":  pendingReviews,
		"violation_count":  violationCount,
		"clean_count":      cleanCount,
		"pending_appeals":  pendingAppeals,
	}, nil
}

// Ensure DB interface compatibility.
var _ *gorm.DB
