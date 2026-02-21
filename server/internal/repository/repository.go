package repository

import (
	"encoding/json"
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
	err := r.DB.Preload("Group").Where("logto_id = ?", logtoID).First(&user).Error
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
	err := r.DB.Preload("Group").Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&users).Error
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

func (r *Repository) HasDuplicateRecord(subdomainID uint, recordType, content string) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type = ? AND content = ?", subdomainID, recordType, content).Count(&count).Error
	return count > 0, err
}

func (r *Repository) HasDuplicateRecordExcluding(subdomainID uint, recordType, content string, excludeID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ? AND type = ? AND content = ? AND id != ?", subdomainID, recordType, content, excludeID).Count(&count).Error
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

func (r *Repository) DeductCredits(tx *gorm.DB, userID uint, amount model.Credit, descriptionKey string, descriptionParams json.RawMessage) error {
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
		UserID:            userID,
		Amount:            -amount,
		Type:              "deduct",
		DescriptionKey:    descriptionKey,
		DescriptionParams: descriptionParams,
		BalanceAfter:      balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) GrantCredits(tx *gorm.DB, userID uint, amount model.Credit, descriptionKey string, descriptionParams json.RawMessage) error {
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
		UserID:            userID,
		Amount:            amount,
		Type:              "grant",
		DescriptionKey:    descriptionKey,
		DescriptionParams: descriptionParams,
		BalanceAfter:      balance.Balance,
	}
	return tx.Create(&txn).Error
}

func (r *Repository) RefundCredits(tx *gorm.DB, userID uint, amount model.Credit, descriptionKey string, descriptionParams json.RawMessage) error {
	var balance model.CreditBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&balance).Error; err != nil {
		return err
	}
	balance.Balance += amount
	if err := tx.Save(&balance).Error; err != nil {
		return err
	}
	txn := model.CreditTransaction{
		UserID:            userID,
		Amount:            amount,
		Type:              "refund",
		DescriptionKey:    descriptionKey,
		DescriptionParams: descriptionParams,
		BalanceAfter:      balance.Balance,
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
	var users, domains, subdomains, dnsRecords, userGroups int64
	r.DB.Model(&model.User{}).Count(&users)
	r.DB.Model(&model.Domain{}).Where("is_active = ?", true).Count(&domains)
	r.DB.Model(&model.Subdomain{}).Count(&subdomains)
	r.DB.Model(&model.DNSRecord{}).Count(&dnsRecords)
	r.DB.Model(&model.UserGroup{}).Count(&userGroups)
	stats["users"] = users
	stats["domains"] = domains
	stats["subdomains"] = subdomains
	stats["dns_records"] = dnsRecords
	stats["user_groups"] = userGroups
	return stats, nil
}

// UserGroup
type UserGroupWithCount struct {
	model.UserGroup
	UserCount int64 `json:"user_count"`
}

func (r *Repository) ListUserGroups() ([]UserGroupWithCount, error) {
	var results []UserGroupWithCount
	err := r.DB.Table("user_groups").
		Select("user_groups.*, COALESCE(COUNT(users.id), 0) as user_count").
		Joins("LEFT JOIN users ON users.group_id = user_groups.id").
		Group("user_groups.id").
		Order("user_groups.id ASC").
		Scan(&results).Error
	return results, err
}

func (r *Repository) FindUserGroup(id uint) (*model.UserGroup, error) {
	var group model.UserGroup
	err := r.DB.First(&group, id).Error
	return &group, err
}

func (r *Repository) CreateUserGroup(group *model.UserGroup) error {
	return r.DB.Create(group).Error
}

func (r *Repository) UpdateUserGroup(group *model.UserGroup) error {
	return r.DB.Save(group).Error
}

func (r *Repository) DeleteUserGroup(id uint) error {
	return r.DB.Delete(&model.UserGroup{}, id).Error
}

func (r *Repository) GetDefaultUserGroup() (*model.UserGroup, error) {
	var group model.UserGroup
	err := r.DB.Where("is_default = ?", true).First(&group).Error
	return &group, err
}

func (r *Repository) SetDefaultUserGroup(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserGroup{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserGroup{}).Where("id = ?", id).Update("is_default", true).Error
	})
}

func (r *Repository) CountUserGroups() (int64, error) {
	var count int64
	err := r.DB.Model(&model.UserGroup{}).Count(&count).Error
	return count, err
}

func (r *Repository) MigrateUsersToGroup(fromGroupID, toGroupID uint) error {
	return r.DB.Model(&model.User{}).Where("group_id = ?", fromGroupID).Update("group_id", toGroupID).Error
}

func (r *Repository) UpdateUserGroupID(userID, groupID uint) error {
	return r.DB.Model(&model.User{}).Where("id = ?", userID).Update("group_id", groupID).Error
}

// DomainGroupAccess
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

// SystemConfig
func (r *Repository) GetSystemConfig(key string) (string, error) {
	var cfg model.SystemConfig
	err := r.DB.Where("\"key\" = ?", key).First(&cfg).Error
	if err != nil {
		return "", err
	}
	return cfg.Value, nil
}

func (r *Repository) SetSystemConfig(key, value string) error {
	var cfg model.SystemConfig
	err := r.DB.Where("\"key\" = ?", key).First(&cfg).Error
	if err != nil {
		return r.DB.Create(&model.SystemConfig{Key: key, Value: value}).Error
	}
	cfg.Value = value
	return r.DB.Save(&cfg).Error
}

func (r *Repository) GetAllSystemConfigs() (map[string]string, error) {
	var configs []model.SystemConfig
	if err := r.DB.Find(&configs).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	return result, nil
}
