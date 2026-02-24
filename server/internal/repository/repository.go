package repository

import (
	"encoding/json"
	"fmt"
	"hl6-server/internal/model"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// escapeLike escapes LIKE/ILIKE pattern characters so user input is treated literally.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

type Repository struct {
	DB *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{DB: db}
}

// GetDB returns the underlying *gorm.DB for cases that need direct access.
func (r *Repository) GetDB() *gorm.DB {
	return r.DB
}

// User
func (r *Repository) FindUserByExternalID(externalID string) (*model.User, error) {
	var user model.User
	err := r.DB.Preload("Group").Where("external_id = ?", externalID).First(&user).Error
	return &user, err
}

func (r *Repository) FindUserByID(id uint) (*model.User, error) {
	var user model.User
	err := r.DB.Preload("Group").First(&user, id).Error
	return &user, err
}

func (r *Repository) CreateUser(user *model.User) error {
	return r.DB.Create(user).Error
}

func (r *Repository) UpdateUser(user *model.User) error {
	return r.DB.Save(user).Error
}

func (r *Repository) ListUsers(page, perPage int, search ...string) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	q := r.DB.Model(&model.User{})
	if len(search) > 0 && search[0] != "" {
		like := "%" + escapeLike(search[0]) + "%"
		q = q.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}
	q.Count(&total)
	err := q.Preload("Group").Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&users).Error
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

func (r *Repository) ListSubdomainsByDomain(domainID uint) ([]model.Subdomain, error) {
	var subs []model.Subdomain
	err := r.DB.Where("domain_id = ?", domainID).Preload("Domain").Preload("DNSRecords").Find(&subs).Error
	return subs, err
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

func (r *Repository) CountDNSRecordsBySubdomain(subdomainID uint) (int64, error) {
	var count int64
	err := r.DB.Model(&model.DNSRecord{}).Where("subdomain_id = ?", subdomainID).Count(&count).Error
	return count, err
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

func (r *Repository) ListAuditLogs(page, perPage int, search ...string) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	q := r.DB.Model(&model.AuditLog{})
	if len(search) > 0 && search[0] != "" {
		like := "%" + escapeLike(search[0]) + "%"
		q = q.Where("action ILIKE ? OR resource ILIKE ?", like, like)
	}
	q.Count(&total)
	err := q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Preload("User").Find(&logs).Error
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

// CloudflareAccount
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

// Notification

type NotificationWithRead struct {
	model.Notification
	IsRead bool `json:"is_read"`
}

func (r *Repository) ListNotificationsForUser(userID, groupID uint, userCreatedAt string, offset, limit int) ([]NotificationWithRead, int64, error) {
	var results []NotificationWithRead
	var total int64

	visibilityWhere := `(
		(n.target_type = 'users' AND n.target_ids @> to_jsonb(?::bigint))
		OR (n.target_type = 'groups' AND n.target_ids @> to_jsonb(?::bigint) AND (n.visible_to_new = true OR n.created_at >= ?))
		OR (n.target_type = 'all' AND (n.visible_to_new = true OR n.created_at >= ?))
	)`

	// Count
	countSQL := `SELECT COUNT(*) FROM notifications n WHERE ` + visibilityWhere
	if err := r.DB.Raw(countSQL, userID, groupID, userCreatedAt, userCreatedAt).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []NotificationWithRead{}, 0, nil
	}

	// Query with read status and complex ordering
	querySQL := `SELECT n.id, n.title, n.content, n.type, n.target_type, n.target_ids, n.visible_to_new, n.created_by, n.created_at,
		CASE WHEN nr.id IS NOT NULL THEN true ELSE false END as is_read
		FROM notifications n
		LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?
		WHERE ` + visibilityWhere + `
		ORDER BY
			CASE WHEN nr.id IS NULL THEN 0 ELSE 1 END ASC,
			CASE
				WHEN nr.id IS NULL THEN
					CASE n.type WHEN 'urgent' THEN 0 WHEN 'pinned' THEN 1 ELSE 2 END
				ELSE
					CASE n.type WHEN 'pinned' THEN 0 ELSE 1 END
			END ASC,
			n.created_at DESC
		OFFSET ? LIMIT ?`

	err := r.DB.Raw(querySQL, userID, userID, groupID, userCreatedAt, userCreatedAt, offset, limit).Scan(&results).Error
	if results == nil {
		results = []NotificationWithRead{}
	}
	return results, total, err
}

func (r *Repository) FindNotificationForUser(id, userID, groupID uint, userCreatedAt string) (*NotificationWithRead, error) {
	var result NotificationWithRead

	visibilityWhere := `(
		(n.target_type = 'users' AND n.target_ids @> to_jsonb(?::bigint))
		OR (n.target_type = 'groups' AND n.target_ids @> to_jsonb(?::bigint) AND (n.visible_to_new = true OR n.created_at >= ?))
		OR (n.target_type = 'all' AND (n.visible_to_new = true OR n.created_at >= ?))
	)`

	querySQL := `SELECT n.*,
		CASE WHEN nr.id IS NOT NULL THEN true ELSE false END as is_read
		FROM notifications n
		LEFT JOIN notification_reads nr ON nr.notification_id = n.id AND nr.user_id = ?
		WHERE n.id = ? AND ` + visibilityWhere

	err := r.DB.Raw(querySQL, userID, id, userID, groupID, userCreatedAt, userCreatedAt).Scan(&result).Error
	if result.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &result, err
}

func (r *Repository) HasUnreadNotifications(userID, groupID uint, userCreatedAt string) (bool, error) {
	var count int64

	querySQL := `SELECT COUNT(*) FROM notifications n
		WHERE NOT EXISTS (SELECT 1 FROM notification_reads nr WHERE nr.notification_id = n.id AND nr.user_id = ?)
		AND (
			(n.target_type = 'users' AND n.target_ids @> to_jsonb(?::bigint))
			OR (n.target_type = 'groups' AND n.target_ids @> to_jsonb(?::bigint) AND (n.visible_to_new = true OR n.created_at >= ?))
			OR (n.target_type = 'all' AND (n.visible_to_new = true OR n.created_at >= ?))
		)`

	err := r.DB.Raw(querySQL, userID, userID, groupID, userCreatedAt, userCreatedAt).Scan(&count).Error
	return count > 0, err
}

func (r *Repository) MarkNotificationRead(notificationID, userID uint) error {
	read := model.NotificationRead{
		NotificationID: notificationID,
		UserID:         userID,
	}
	return r.DB.Where("notification_id = ? AND user_id = ?", notificationID, userID).
		FirstOrCreate(&read).Error
}

func (r *Repository) CreateNotification(n *model.Notification) error {
	return r.DB.Create(n).Error
}

var imageURLRegexp = regexp.MustCompile(`/api/v1/notifications/images/(\d+)`)

func (r *Repository) CreateNotificationWithImages(n *model.Notification) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(n).Error; err != nil {
			return err
		}

		// Link referenced images to this notification
		matches := imageURLRegexp.FindAllStringSubmatch(n.Content, -1)
		var imageIDs []uint
		for _, m := range matches {
			id, err := strconv.ParseUint(m[1], 10, 64)
			if err != nil {
				continue
			}
			imageIDs = append(imageIDs, uint(id))
		}
		if len(imageIDs) > 0 {
			if err := tx.Model(&model.NotificationImage{}).
				Where("id IN ? AND notification_id IS NULL", imageIDs).
				Update("notification_id", n.ID).Error; err != nil {
				return fmt.Errorf("failed to link images: %w", err)
			}
		}

		return nil
	})
}

func (r *Repository) DeleteNotification(id uint) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notification_id = ?", id).Delete(&model.NotificationRead{}).Error; err != nil {
			return err
		}
		if err := tx.Where("notification_id = ?", id).Delete(&model.NotificationImage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Notification{}, id).Error
	})
}

func (r *Repository) FindNotification(id uint) (*model.Notification, error) {
	var n model.Notification
	err := r.DB.Preload("Creator").First(&n, id).Error
	return &n, err
}

func (r *Repository) ListNotificationsAdmin(page, perPage int) ([]model.Notification, int64, error) {
	var notifications []model.Notification
	var total int64
	q := r.DB.Model(&model.Notification{})
	q.Count(&total)
	err := q.Preload("Creator").Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&notifications).Error
	return notifications, total, err
}

func (r *Repository) CreateNotificationImage(img *model.NotificationImage) error {
	return r.DB.Create(img).Error
}

func (r *Repository) FindNotificationImage(id uint) (*model.NotificationImage, error) {
	var img model.NotificationImage
	err := r.DB.First(&img, id).Error
	return &img, err
}

// GetNotificationTargetUserIDs resolves target user IDs based on target_type
func (r *Repository) GetNotificationTargetUserIDs(n *model.Notification) ([]uint, error) {
	switch n.TargetType {
	case "users":
		var ids []uint
		if err := json.Unmarshal(n.TargetIDs, &ids); err != nil {
			return nil, err
		}
		return ids, nil
	case "groups":
		var groupIDs []uint
		if err := json.Unmarshal(n.TargetIDs, &groupIDs); err != nil {
			return nil, err
		}
		var userIDs []uint
		err := r.DB.Model(&model.User{}).Where("group_id IN ?", groupIDs).Pluck("id", &userIDs).Error
		return userIDs, err
	case "all":
		// Return empty slice to signal "all users"
		return nil, nil
	default:
		return nil, nil
	}
}
