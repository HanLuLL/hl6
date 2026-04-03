package repository

import "hl6-server/internal/model"

func (r *Repository) FindUserByReferralCode(code string) (*model.User, error) {
	var user model.User
	err := r.DB.Where("referral_code = ?", code).First(&user).Error
	return &user, err
}

func (r *Repository) ListReferralsByInviter(inviterID uint, page, perPage int) ([]model.UserReferral, int64, error) {
	var referrals []model.UserReferral
	var total int64
	q := r.DB.Model(&model.UserReferral{}).Where("inviter_id = ?", inviterID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Invitee").Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&referrals).Error
	return referrals, total, err
}

func (r *Repository) GetReferralInvitersForUsers(userIDs []uint) (map[uint]*model.User, error) {
	if len(userIDs) == 0 {
		return make(map[uint]*model.User), nil
	}
	var referrals []model.UserReferral
	if err := r.DB.Where("invitee_id IN ?", userIDs).Preload("Inviter").Find(&referrals).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]*model.User)
	for i := range referrals {
		result[referrals[i].InviteeID] = &referrals[i].Inviter
	}
	return result, nil
}
