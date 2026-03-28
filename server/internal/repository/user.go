package repository

import "hl6-server/internal/model"

func (r *Repository) FindUserByExternalID(externalID string) (*model.User, error) {
	var user model.User
	err := r.DB.Preload("Group").Where("external_id = ?", externalID).First(&user).Error
	return &user, err
}

func (r *Repository) FindUserByID(id uint) (*model.User, error) {
	var user model.User
	err := r.DB.Preload("Group").First(&user, id).Error
	return &user, err
}

func (r *Repository) CreateUser(user *model.User) error {
	return r.DB.Create(user).Error
}

func (r *Repository) UpdateUser(user *model.User) error {
	return r.DB.Save(user).Error
}

func (r *Repository) ListUsers(page, perPage int, search, banStatus string) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	q := r.DB.Model(&model.User{})
	if search != "" {
		like := "%" + escapeLike(search) + "%"
		q = q.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}

	switch banStatus {
	case "active":
		q = q.Where("is_banned = ?", false)
	case "banned":
		q = q.Where("is_banned = ?", true)
	}

	q.Count(&total)
	err := q.Preload("Group").Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&users).Error
	return users, total, err
}

func (r *Repository) CountUsers() (int64, error) {
	var total int64
	err := r.DB.Model(&model.User{}).Count(&total).Error
	return total, err
}

func (r *Repository) CountUnbannedAdmins() (int64, error) {
	var total int64
	err := r.DB.Model(&model.User{}).
		Where("role = ? AND is_banned = ?", "admin", false).
		Count(&total).Error
	return total, err
}
