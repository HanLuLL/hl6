package repository

import (
	"time"

	"hl6-server/internal/model"

	"gorm.io/gorm/clause"
)

func (r *Repository) FindBrandingAssetByType(assetType string) (*model.BrandingAsset, error) {
	var asset model.BrandingAsset
	err := r.DB.Where("asset_type = ?", assetType).First(&asset).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *Repository) ListBrandingAssets(assetTypes []string) ([]model.BrandingAsset, error) {
	var assets []model.BrandingAsset
	if len(assetTypes) == 0 {
		return assets, nil
	}
	err := r.DB.Select("id", "asset_type", "size", "created_at", "updated_at").
		Where("asset_type IN ?", assetTypes).
		Find(&assets).Error
	return assets, err
}

func (r *Repository) UpsertBrandingAsset(assetType string, data []byte) error {
	now := time.Now()
	asset := model.BrandingAsset{
		AssetType: assetType,
		Data:      data,
		Size:      len(data),
		UpdatedAt: now,
	}
	return r.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "asset_type"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"data":       data,
			"size":       len(data),
			"updated_at": now,
		}),
	}).Create(&asset).Error
}

func (r *Repository) DeleteBrandingAssets(assetTypes []string) error {
	if len(assetTypes) == 0 {
		return nil
	}
	return r.DB.Where("asset_type IN ?", assetTypes).Delete(&model.BrandingAsset{}).Error
}
