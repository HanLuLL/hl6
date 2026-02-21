package repository

import (
	"hl6-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{DB: db}
}

// User
func (r *Repository) FindUserByLogtoID(logtoID string) (*model.User, error) {
	var user model.User
	err := r.DB.Where("logto_id = ?", logtoID).First(&user).Error
	return &user, err
}

func (r *Repository) CreateUser(user *model.User) error {
	return r.DB.Create(user).Error
}

func (r *Repository) UpdateUser(user *model.User) error {
	return r.DB.Save(user).Error
}

func (r *Repository) ListUsers(page, perPage int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	r.DB.Model(&model.User{}).Count(&total)
	err := r.DB.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&users).Error
	return users, total, err
}

// Domain
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

// Subdomain
func (r *Repository) ListSubdomainsByUser(userID uint) ([]model.Subdomain, error) {
	var subs []model.Subdomain
	err := r.DB.Where("user_id = ?", userID).Preload("Domain").Order("created_at DESC").Find(&subs).Error
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

// DNS Records
func (r *Repository) ListDNSRecords(subdomainID uint) ([]model.DNSRecord, error) {
	var records []model.DNSRecord
	err := r.DB.Where("subdomain_id = ?", subdomainID).Order("type ASC, name ASC").Find(&records).Error
	return records, err
}

func (r *Repository) FindDNSRecord(id uint) (*model.DNSRecord, error) {
	var record model.DNSRecord
	err := r.DB.First(&record, id).Error
	return &record, err
}

func (r *Repository) CreateDNSRecord(record *model.DNSRecord) error {
	return r.DB.Create(record).Error
}

func (r *Repository) UpdateDNSRecord(record *model.DNSRecord) error {
	return r.DB.Save(record).Error
}

func (r *Repository) DeleteDNSRecord(id uint) error {
	return r.DB.Delete(&model.DNSRecord{}, id).Error
}

func (r *Repository) HasNonCNAMERecords(subdomainID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type != ?", subdomainID, "CNAME").Count(&count).Error
	return count > 0, err
}

func (r *Repository) HasCNAMERecord(subdomainID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type = ?", subdomainID, "CNAME").Count(&count).Error
	return count > 0, err
}

// Credits
func (r *Repository) GetCreditBalance(userID uint) (*model.CreditBalance, error) {
	var balance model.CreditBalance
	err := r.DB.Where("user_id = ?", userID).First(&balance).Error
	return &balance, err
}

func (r *Repository) EnsureCreditBalance(userID uint) (*model.CreditBalance, error) {
	var balance model.CreditBalance
	err := r.DB.Where("user_id = ?", userID).FirstOrCreate(&balance, model.CreditBalance{UserID: userID, Balance: 0}).Error
	return &balance, err
}

func (r *Repository) DeductCredits(tx *gorm.DB, userID uint, amount int, description string) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		return err
	}
	if balance.Balance < amount {
		return gorm.ErrInvalidData
	}
	balance.Balance -= amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:       userID,
		Amount:       -amount,
		Type:         "deduct",
		Description:  description,
		BalanceAfter: balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) GrantCredits(tx *gorm.DB, userID uint, amount int, description string) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		balance = model.CreditBalance{UserID: userID, Balance: 0}
		if err := tx.Create(&balance).Error; err != nil {
			return err
		}
	}
	balance.Balance += amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:       userID,
		Amount:       amount,
		Type:         "grant",
		Description:  description,
		BalanceAfter: balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) RefundCredits(tx *gorm.DB, userID uint, amount int, description string) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		return err
	}
	balance.Balance += amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:       userID,
		Amount:       amount,
		Type:         "refund",
		Description:  description,
		BalanceAfter: balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) ListTransactions(userID uint, page, perPage int) ([]model.CreditTransaction, int64, error) {
	var txns []model.CreditTransaction
	var total int64
	q := r.DB.Model(&model.CreditTransaction{}).Where("user_id = ?", userID)
	q.Count(&total)
	err := q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&txns).Error
	return txns, total, err
}

// Audit Log
func (r *Repository) CreateAuditLog(log *model.AuditLog) error {
	return r.DB.Create(log).Error
}

func (r *Repository) ListAuditLogs(page, perPage int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	r.DB.Model(&model.AuditLog{}).Count(&total)
	err := r.DB.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Preload("User").Find(&logs).Error
	return logs, total, err
}

// Stats
func (r *Repository) GetStats() (map[string]int64, error) {
	stats := make(map[string]int64)
	var users, domains, subdomains, dnsRecords int64
	r.DB.Model(&model.User{}).Count(&users)
	r.DB.Model(&model.Domain{}).Where("is_active = ?", true).Count(&domains)
	r.DB.Model(&model.Subdomain{}).Count(&subdomains)
	r.DB.Model(&model.DNSRecord{}).Count(&dnsRecords)
	stats["users"] = users
	stats["domains"] = domains
	stats["subdomains"] = subdomains
	stats["dns_records"] = dnsRecords
	return stats, nil
}
