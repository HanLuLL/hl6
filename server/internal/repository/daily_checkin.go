package repository

import "hl6-server/internal/model"

func (r *Repository) HasDailyCheckinClaim(userID uint, checkinDate string) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DailyCheckinClaim{}).
		Where("user_id = ? AND checkin_date = ?", userID, checkinDate).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
