package repository

import "hl6-server/internal/model"

func (r *Repository) ListSubdomainsByUser(userID uint) ([]model.Subdomain, error) {
	var subs []model.Subdomain
	err := r.DB.Where("user_id = ?", userID).Preload("Domain").Order("created_at DESC").Find(&subs).Error
	return subs, err
}

func (r *Repository) ListSubdomainsByUserWithRecords(userID uint) ([]model.Subdomain, error) {
	var subs []model.Subdomain
	err := r.DB.Where("user_id = ?", userID).
		Preload("Domain").
		Preload("DNSRecords").
		Order("created_at DESC").
		Find(&subs).Error
	return subs, err
}

func (r *Repository) FindSubdomain(id uint) (*model.Subdomain, error) {
	var sub model.Subdomain
	err := r.DB.Preload("Domain").Preload("DNSRecords").First(&sub, id).Error
	return &sub, err
}

func (r *Repository) FindSubdomainByName(domainID uint, name string) (*model.Subdomain, error) {
	var sub model.Subdomain
	err := r.DB.Where("domain_id = ? AND name = ?", domainID, name).First(&sub).Error
	return &sub, err
}

func (r *Repository) CreateSubdomain(sub *model.Subdomain) error {
	return r.DB.Create(sub).Error
}

func (r *Repository) DeleteSubdomain(id uint) error {
	return r.DB.Delete(&model.Subdomain{}, id).Error
}

func (r *Repository) ListSubdomainsByDomain(domainID uint) ([]model.Subdomain, error) {
	var subs []model.Subdomain
	err := r.DB.Where("domain_id = ?", domainID).Preload("Domain").Preload("DNSRecords").Find(&subs).Error
	return subs, err
}
