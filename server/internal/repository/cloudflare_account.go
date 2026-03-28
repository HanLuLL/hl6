package repository

import "hl6-server/internal/model"

func (r *Repository) ListCloudflareAccounts() ([]model.CloudflareAccount, error) {
	var accounts []model.CloudflareAccount
	err := r.DB.Order("id ASC").Find(&accounts).Error
	return accounts, err
}

func (r *Repository) FindCloudflareAccount(id uint) (*model.CloudflareAccount, error) {
	var account model.CloudflareAccount
	err := r.DB.First(&account, id).Error
	return &account, err
}

func (r *Repository) CreateCloudflareAccount(account *model.CloudflareAccount) error {
	return r.DB.Create(account).Error
}

func (r *Repository) UpdateCloudflareAccount(account *model.CloudflareAccount) error {
	return r.DB.Save(account).Error
}

func (r *Repository) DeleteCloudflareAccount(id uint) error {
	return r.DB.Delete(&model.CloudflareAccount{}, id).Error
}

func (r *Repository) CountDomainsByAccount(accountID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Domain{}).Where("cloudflare_account_id = ?", accountID).Count(&count).Error
	return count, err
}
