package repository

import (
	"hl6-server/internal/model"

	"gorm.io/gorm"
)

func (r *Repository) ListDomains(activeOnly bool) ([]model.Domain, error) {
	var domains []model.Domain
	q := r.DB
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	err := q.Order("name ASC").Find(&domains).Error
	return domains, err
}

func (r *Repository) FindDomain(id uint) (*model.Domain, error) {
	var domain model.Domain
	err := r.DB.First(&domain, id).Error
	return &domain, err
}

func (r *Repository) CreateDomain(domain *model.Domain) error {
	return r.DB.Create(domain).Error
}

func (r *Repository) UpdateDomain(domain *model.Domain) error {
	return r.DB.Save(domain).Error
}

func (r *Repository) CountSubdomainsByDomain(domainID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Subdomain{}).Where("domain_id = ?", domainID).Count(&count).Error
	return count, err
}

func (r *Repository) DeleteDomain(tx *gorm.DB, domainID uint) error {
	if err := tx.Where("domain_id = ?", domainID).Delete(&model.DomainGroupAccess{}).Error; err != nil {
		return err
	}
	return tx.Delete(&model.Domain{}, domainID).Error
}

func (r *Repository) ListDomainGroupAccess(domainID uint) ([]model.DomainGroupAccess, error) {
	var accesses []model.DomainGroupAccess
	err := r.DB.Preload("Group").Where("domain_id = ?", domainID).Find(&accesses).Error
	return accesses, err
}

func (r *Repository) ListAllDomainGroupAccess() (map[uint][]model.DomainGroupAccess, error) {
	var accesses []model.DomainGroupAccess
	if err := r.DB.Preload("Group").Order("domain_id ASC").Find(&accesses).Error; err != nil {
		return nil, err
	}
	result := make(map[uint][]model.DomainGroupAccess)
	for _, a := range accesses {
		result[a.DomainID] = append(result[a.DomainID], a)
	}
	return result, nil
}

func (r *Repository) ReplaceDomainGroupAccess(tx *gorm.DB, domainID uint, accesses []model.DomainGroupAccess) error {
	if err := tx.Where("domain_id = ?", domainID).Delete(&model.DomainGroupAccess{}).Error; err != nil {
		return err
	}
	for i := range accesses {
		accesses[i].DomainID = domainID
		accesses[i].ID = 0
		if err := tx.Create(&accesses[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) FindDomainGroupAccess(domainID, groupID uint) (*model.DomainGroupAccess, error) {
	var access model.DomainGroupAccess
	err := r.DB.Where("domain_id = ? AND group_id = ?", domainID, groupID).First(&access).Error
	return &access, err
}

type DomainWithGroupCost struct {
	model.Domain
	GroupCreditCost model.Credit `json:"credit_cost"`
}

func (r *Repository) ListDomainsForGroup(groupID uint) ([]DomainWithGroupCost, error) {
	var results []DomainWithGroupCost
	err := r.DB.Table("domains").
		Select("domains.*, domain_group_accesses.credit_cost as group_credit_cost").
		Joins("INNER JOIN domain_group_accesses ON domain_group_accesses.domain_id = domains.id").
		Where("domain_group_accesses.group_id = ? AND domains.is_active = ?", groupID, true).
		Order("domains.name ASC").
		Scan(&results).Error
	return results, err
}

func (r *Repository) DeleteDomainGroupAccessByGroup(tx *gorm.DB, groupID uint) error {
	return tx.Where("group_id = ?", groupID).Delete(&model.DomainGroupAccess{}).Error
}
