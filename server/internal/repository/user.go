package repository

import "hl6-server/internal/model"

type UserWithCredits struct {
	model.User
	Credits model.Credit `json:"credits" gorm:"column:credits"`
}

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

func (r *Repository) ListUsers(page, perPage int, search, banStatus, role string, groupID *uint, inviter string) ([]UserWithCredits, int64, error) {
	var users []UserWithCredits
	var total int64
	q := r.DB.Model(&model.User{}).
		Select("users.*, COALESCE(credit_balances.balance, 0) AS credits").
		Joins("LEFT JOIN credit_balances ON credit_balances.user_id = users.id")

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

	switch role {
	case "user", "admin":
		q = q.Where("role = ?", role)
	}

	if groupID != nil {
		q = q.Where("group_id = ?", *groupID)
	}

	if inviter != "" {
		like := "%" + escapeLike(inviter) + "%"
		q = q.Where(`
EXISTS (
  SELECT 1
  FROM user_referrals ur
  JOIN users inviters ON inviters.id = ur.inviter_id
  WHERE ur.invitee_id = users.id
    AND (inviters.email ILIKE ? OR inviters.name ILIKE ?)
)`, like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
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
