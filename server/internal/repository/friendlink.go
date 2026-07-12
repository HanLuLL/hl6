package repository

import (
	"hl6-server/internal/model"

	"gorm.io/gorm"
)

// ListPublicFriendLinks 返回前台可见（启用）的友链，按 sort_order 升序、id 升序排列。
func (r *Repository) ListPublicFriendLinks() ([]model.FriendLink, error) {
	var links []model.FriendLink
	err := r.DB.Where("is_active = ?", true).
		Order("sort_order ASC, id ASC").
		Find(&links).Error
	return links, err
}

// ListAllFriendLinks 返回全部友链（含禁用），供后台管理使用。
func (r *Repository) ListAllFriendLinks() ([]model.FriendLink, error) {
	var links []model.FriendLink
	err := r.DB.Order("sort_order ASC, id ASC").Find(&links).Error
	return links, err
}

// FindFriendLink 按 ID 查询友链。
func (r *Repository) FindFriendLink(id uint) (*model.FriendLink, error) {
	var link model.FriendLink
	err := r.DB.First(&link, id).Error
	return &link, err
}

// CreateFriendLink 创建友链。
func (r *Repository) CreateFriendLink(link *model.FriendLink) error {
	return r.DB.Create(link).Error
}

// UpdateFriendLink 保存友链修改。
func (r *Repository) UpdateFriendLink(link *model.FriendLink) error {
	return r.DB.Save(link).Error
}

// DeleteFriendLink 删除友链。
func (r *Repository) DeleteFriendLink(id uint) error {
	return r.DB.Delete(&model.FriendLink{}, id).Error
}

// ReorderFriendLinks 批量更新排序（key=ID, value=SortOrder）。
func (r *Repository) ReorderFriendLinks(orders map[uint]int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		for id, order := range orders {
			if err := tx.Model(&model.FriendLink{}).Where("id = ?", id).Update("sort_order", order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
