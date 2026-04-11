package repository

import "hl6-server/internal/model"

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
