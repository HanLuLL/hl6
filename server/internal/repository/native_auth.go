package repository

import (
	"errors"
	"time"

	"hl6-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNativeAuthCodeInvalid = errors.New("native auth code is invalid or expired")
var ErrNativeAuthRequestInvalid = errors.New("native auth request is invalid or expired")

func (r *Repository) CreateNativeAuthCode(codeHash, externalID string, expiresAt time.Time) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", time.Now()).Delete(&model.NativeAuthCode{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.NativeAuthCode{
			CodeHash:   codeHash,
			ExternalID: externalID,
			ExpiresAt:  expiresAt,
		}).Error
	})
}

func (r *Repository) ConsumeNativeAuthCode(codeHash string) (string, error) {
	var externalID string
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var code model.NativeAuthCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code_hash = ? AND used_at IS NULL AND expires_at > ?", codeHash, time.Now()).
			First(&code).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNativeAuthCodeInvalid
			}
			return err
		}
		now := time.Now()
		if err := tx.Model(&code).Update("used_at", &now).Error; err != nil {
			return err
		}
		externalID = code.ExternalID
		return nil
	})
	return externalID, err
}

func (r *Repository) CreateNativeAuthRequest(requestHash, redirectURI string, expiresAt time.Time) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", time.Now()).Delete(&model.NativeAuthRequest{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.NativeAuthRequest{
			RequestHash: requestHash,
			RedirectURI: redirectURI,
			ExpiresAt:   expiresAt,
		}).Error
	})
}

func (r *Repository) ConsumeNativeAuthRequest(requestHash string) (string, error) {
	var redirectURI string
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var request model.NativeAuthRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("request_hash = ? AND used_at IS NULL AND expires_at > ?", requestHash, time.Now()).
			First(&request).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNativeAuthRequestInvalid
			}
			return err
		}
		now := time.Now()
		if err := tx.Model(&request).Update("used_at", &now).Error; err != nil {
			return err
		}
		redirectURI = request.RedirectURI
		return nil
	})
	return redirectURI, err
}
