package model

import "time"

const (
	BrandingAssetTypeLogoWebP   = "logo_webp"
	BrandingAssetTypeFaviconICO = "favicon_ico"
)

type BrandingAsset struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	AssetType string    `json:"asset_type" gorm:"uniqueIndex;not null"`
	Data      []byte    `json:"-" gorm:"type:bytea;not null"`
	Size      int       `json:"size" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
