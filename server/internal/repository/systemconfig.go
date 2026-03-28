package repository

import (
	"hl6-server/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) GetSystemConfig(key string) (string, error) {
	cfg, err := r.FindSystemConfig(key)
	if err != nil {
		return "", err
	}
	return cfg.Value, nil
}

func (r *Repository) FindSystemConfig(key string) (*model.SystemConfig, error) {
	var cfg model.SystemConfig
	result := r.DB.Where("\"key\" = ?", key).Limit(1).Find(&cfg)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &cfg, nil
}

func (r *Repository) SetSystemConfig(key, value string) error {
	var cfg model.SystemConfig
	result := r.DB.Where("\"key\" = ?", key).Limit(1).Find(&cfg)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return r.DB.Create(&model.SystemConfig{Key: key, Value: value}).Error
	}
	cfg.Value = value
	return r.DB.Save(&cfg).Error
}

func (r *Repository) GetSystemConfigsByKeys(keys []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(keys) == 0 {
		return result, nil
	}

	var configs []model.SystemConfig
	if err := r.DB.Where("\"key\" IN ?", keys).Find(&configs).Error; err != nil {
		return nil, err
	}
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	return result, nil
}
