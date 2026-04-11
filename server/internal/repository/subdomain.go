package repository

import (
	"time"

	"hl6-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

func (r *Repository) LockSubdomainForUpdate(tx *gorm.DB, id uint) (*model.Subdomain, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}

	var sub model.Subdomain
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&sub, id).Error
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

type AdminClaimedSubdomainDTO struct {
	ID             uint      `json:"id"`
	DomainID       uint      `json:"domain_id"`
	UserID         uint      `json:"user_id"`
	FQDN           string    `json:"fqdn"`
	DomainName     string    `json:"domain_name"`
	UserEmail      string    `json:"user_email"`
	DNSRecordCount int64     `json:"dns_record_count"`
	CreatedAt      time.Time `json:"created_at"`
}

func (r *Repository) AdminListClaimedSubdomains(page, perPage int, search string) ([]AdminClaimedSubdomainDTO, int64, error) {
	var results []AdminClaimedSubdomainDTO
	var total int64

	countQ := r.DB.Model(&model.Subdomain{}).
		Joins("JOIN users ON users.id = subdomains.user_id").
		Joins("JOIN domains ON domains.id = subdomains.domain_id")

	if search != "" {
		like := "%" + escapeLike(search) + "%"
		countQ = countQ.Where("subdomains.fqdn ILIKE ? OR users.email ILIKE ? OR domains.name ILIKE ?", like, like, like)
	}

	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := r.DB.Table("subdomains").
		Select(`
subdomains.id,
subdomains.domain_id,
subdomains.user_id,
subdomains.fqdn,
subdomains.created_at,
domains.name as domain_name,
users.email as user_email,
COUNT(dns_records.id) as dns_record_count
`).
		Joins("JOIN domains ON domains.id = subdomains.domain_id").
		Joins("JOIN users ON users.id = subdomains.user_id").
		Joins("LEFT JOIN dns_records ON dns_records.subdomain_id = subdomains.id")

	if search != "" {
		like := "%" + escapeLike(search) + "%"
		q = q.Where("subdomains.fqdn ILIKE ? OR users.email ILIKE ? OR domains.name ILIKE ?", like, like, like)
	}

	err := q.Group("subdomains.id, subdomains.domain_id, subdomains.user_id, subdomains.fqdn, subdomains.created_at, domains.name, users.email").
		Order("subdomains.created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Scan(&results).Error
	if err != nil {
		return nil, 0, err
	}
	if results == nil {
		results = []AdminClaimedSubdomainDTO{}
	}
	return results, total, nil
}
