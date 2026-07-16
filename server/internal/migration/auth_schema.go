package migration

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"hl6-server/internal/model"
)

const localAuthEnabledConfigKey = "auth.local.enabled"

// InstallAuthSchema is deliberately additive. The destructive removal of
// legacy OIDC schema happens only through the explicit cutover command.
func InstallAuthSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("auth schema database is nil")
	}
	return db.AutoMigrate(
		&model.UserCredential{},
		&model.AuthToken{},
		&model.AuthSecurityEvent{},
		&model.DatabaseBackup{},
		&model.DatabaseRestoreJob{},
	)
}

// EnsureLocalAuthDefault preserves the explicit v1-to-v2 cutover for existing
// installations while making a brand-new installation usable without a
// migration command. Once written, the setting is never changed implicitly.
func EnsureLocalAuthDefault(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("local authentication default database is nil")
	}

	var existing model.SystemConfig
	err := db.Where("\"key\" = ?", localAuthEnabledConfigKey).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load local authentication setting: %w", err)
	}

	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return fmt.Errorf("count users for local authentication default: %w", err)
	}
	value := "false"
	if userCount == 0 {
		value = "true"
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&model.SystemConfig{Key: localAuthEnabledConfigKey, Value: value}).Error; err != nil {
		return fmt.Errorf("create local authentication default: %w", err)
	}
	return nil
}
