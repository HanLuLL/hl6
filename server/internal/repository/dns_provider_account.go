package repository

import (
	"time"

	"hl6-server/internal/model"
)

func (r *Repository) ListDNSProviderAccounts() ([]model.DNSProviderAccount, error) {
	var accounts []model.DNSProviderAccount
	err := r.DB.Order("id ASC").Find(&accounts).Error
	return accounts, err
}

func (r *Repository) FindDNSProviderAccount(id uint) (*model.DNSProviderAccount, error) {
	var account model.DNSProviderAccount
	err := r.DB.First(&account, id).Error
	return &account, err
}

func (r *Repository) CreateDNSProviderAccount(account *model.DNSProviderAccount) error {
	return r.DB.Create(account).Error
}

func (r *Repository) UpdateDNSProviderAccount(account *model.DNSProviderAccount) error {
	return r.DB.Save(account).Error
}

func (r *Repository) DeleteDNSProviderAccount(id uint) error {
	return r.DB.Delete(&model.DNSProviderAccount{}, id).Error
}

func (r *Repository) CountDomainsByAccount(accountID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Domain{}).Where("provider_account_id = ?", accountID).Count(&count).Error
	return count, err
}

// UpdateDNSProviderAccountVerified marks the account as verified (status=active, clears errors).
func (r *Repository) UpdateDNSProviderAccountVerified(id uint) error {
	now := time.Now()
	return r.DB.Model(&model.DNSProviderAccount{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":              model.DNSProviderAccountStatusActive,
		"last_verified_at":    &now,
		"last_error_category": "",
		"last_error_message":  "",
	}).Error
}

// UpdateDNSProviderAccountError records a connectivity failure on the account.
func (r *Repository) UpdateDNSProviderAccountError(id uint, errCategory, errMessage string) error {
	return r.DB.Model(&model.DNSProviderAccount{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_error_category": errCategory,
		"last_error_message":  errMessage,
	}).Error
}

// SetDNSProviderAccountStatus updates the status field of an account.
func (r *Repository) SetDNSProviderAccountStatus(id uint, status string) error {
	return r.DB.Model(&model.DNSProviderAccount{}).Where("id = ?", id).
		Update("status", status).Error
}
