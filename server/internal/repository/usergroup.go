package repository

import (
	"hl6-server/internal/model"

	"gorm.io/gorm"
)

type UserGroupWithCount struct {
	model.UserGroup
	UserCount int64 `json:"user_count"`
}

func (r *Repository) ListUserGroups() ([]UserGroupWithCount, error) {
	var results []UserGroupWithCount
	err := r.DB.Table("user_groups").
		Select("user_groups.*, COALESCE(COUNT(users.id), 0) as user_count").
		Joins("LEFT JOIN users ON users.group_id = user_groups.id").
		Group("user_groups.id").
		Order("user_groups.id ASC").
		Scan(&results).Error
	return results, err
}

func (r *Repository) FindUserGroup(id uint) (*model.UserGroup, error) {
	var group model.UserGroup
	err := r.DB.First(&group, id).Error
	return &group, err
}

func (r *Repository) CreateUserGroup(group *model.UserGroup) error {
	return r.DB.Create(group).Error
}

func (r *Repository) UpdateUserGroup(group *model.UserGroup) error {
	return r.DB.Save(group).Error
}

func (r *Repository) DeleteUserGroup(id uint) error {
	return r.DB.Delete(&model.UserGroup{}, id).Error
}

func (r *Repository) GetDefaultUserGroup() (*model.UserGroup, error) {
	var group model.UserGroup
	err := r.DB.Where("is_default = ?", true).First(&group).Error
	return &group, err
}

func (r *Repository) SetDefaultUserGroup(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserGroup{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserGroup{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

func (r *Repository) CountUserGroups() (int64, error) {
	var count int64
	err := r.DB.Model(&model.UserGroup{}).Count(&count).Error
	return count, err
}

func (r *Repository) MigrateUsersToGroup(fromGroupID, toGroupID uint) error {
	return r.DB.Model(&model.User{}).Where("group_id = ?", fromGroupID).Update("group_id", toGroupID).Error
}

func (r *Repository) UpdateUserGroupID(userID, groupID uint) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("group_id", groupID).Error
}
