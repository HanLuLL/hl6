package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
	"hl6-server/pkg/audit"
)

func (r *Repository) CreateAuditLog(log *model.AuditLog) error {
	return r.DB.Create(log).Error
}

func (r *Repository) CreateAuditLogTx(tx *gorm.DB, log *model.AuditLog) error {
	if tx == nil {
		return r.CreateAuditLog(log)
	}
	return tx.Create(log).Error
}

func (r *Repository) ListAuditLogs(page, perPage int, operator, action string) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	q := r.DB.Model(&model.AuditLog{})

	if operator != "" {
		like := "%" + escapeLike(operator) + "%"
		q = q.Joins("LEFT JOIN users ON users.id = audit_logs.user_id").
			Where("users.email ILIKE ?", like)
	}

	if action != "" {
		like := "%" + escapeLike(action) + "%"
		q = q.Where("audit_logs.action ILIKE ?", like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset((page-1)*perPage).Limit(perPage).Order("audit_logs.created_at DESC").Preload("User").Find(&logs).Error
	return logs, total, err
}

// --- AuditRule CRUD ---

func (r *Repository) ListAuditRules() ([]model.AuditRule, error) {
	var rules []model.AuditRule
	err := r.DB.Order("created_at DESC").Find(&rules).Error
	return rules, err
}

func (r *Repository) ListEnabledAuditRules() ([]model.AuditRule, error) {
	var rules []model.AuditRule
	err := r.DB.Where("enabled = ?", true).Order("created_at ASC").Find(&rules).Error
	return rules, err
}

func (r *Repository) FindAuditRule(id uint) (*model.AuditRule, error) {
	var rule model.AuditRule
	err := r.DB.First(&rule, id).Error
	return &rule, err
}

func (r *Repository) CreateAuditRule(rule *model.AuditRule) error {
	return r.DB.Create(rule).Error
}

func (r *Repository) UpdateAuditRule(rule *model.AuditRule) error {
	return r.DB.Save(rule).Error
}

func (r *Repository) DeleteAuditRule(id uint) error {
	return r.DB.Delete(&model.AuditRule{}, id).Error
}

func (r *Repository) DomainExists(id uint) bool {
	var count int64
	r.DB.Model(&model.Domain{}).Where("id = ?", id).Count(&count)
	return count > 0
}

// --- Subdomain 状态更新 ---

func (r *Repository) UpdateSubdomainStatusFunc(tx *gorm.DB, subdomainID uint, status, reason string, suspendedAt *time.Time) error {
	updates := map[string]interface{}{
		"status":           status,
		"suspended_reason": reason,
		"updated_at":       time.Now(),
	}
	if suspendedAt != nil {
		updates["suspended_at"] = suspendedAt
	} else {
		updates["suspended_at"] = nil
	}
	return tx.Model(&model.Subdomain{}).Where("id = ?", subdomainID).Updates(updates).Error
}

func (r *Repository) RestoreSubdomainStatus(tx *gorm.DB, subdomainID uint) error {
	now := time.Now()
	return tx.Model(&model.Subdomain{}).Where("id = ?", subdomainID).Updates(map[string]interface{}{
		"status":           model.SubdomainStatusActive,
		"suspended_reason": "",
		"suspended_at":     nil,
		"updated_at":       now,
	}).Error
}

func (r *Repository) UpdateDNSRecordStatus(tx *gorm.DB, recordID uint, status, providerRecordID string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if providerRecordID != "" {
		updates["provider_record_id"] = providerRecordID
	}
	return tx.Model(&model.DNSRecord{}).Where("id = ?", recordID).Updates(updates).Error
}

// --- 巡检目标查询 ---

func (r *Repository) ListActiveScanTargets(offset, limit int) ([]model.AuditScanTarget, error) {
	var targets []model.AuditScanTarget
	err := r.DB.Raw(`
		SELECT DISTINCT s.id, s.fqdn
		FROM subdomains s
		JOIN dns_records dr ON dr.subdomain_id = s.id
		WHERE s.status = ?
		  AND dr.status = ?
		  AND dr.type IN ?
		ORDER BY s.id ASC
		OFFSET ? LIMIT ?
	`, model.SubdomainStatusActive, model.DNSRecordStatusActive, audit.ScannableRecordTypes, offset, limit).Scan(&targets).Error
	return targets, err
}

func (r *Repository) CountActiveScanTargets() (int64, error) {
	var count int64
	err := r.DB.Raw(`
		SELECT COUNT(DISTINCT s.id)
		FROM subdomains s
		JOIN dns_records dr ON dr.subdomain_id = s.id
		WHERE s.status = ?
		  AND dr.status = ?
		  AND dr.type IN ?
	`, model.SubdomainStatusActive, model.DNSRecordStatusActive, audit.ScannableRecordTypes).Scan(&count).Error
	return count, err
}

func (r *Repository) ListActiveSubdomainsByUser(userID uint) ([]model.Subdomain, error) {
	var subs []model.Subdomain
	err := r.DB.Where("user_id = ? AND status = ?", userID, model.SubdomainStatusActive).
		Preload("Domain").
		Preload("DNSRecords").
		Find(&subs).Error
	return subs, err
}

// --- SubdomainScan CRUD ---

func (r *Repository) CreateSubdomainScan(scan *model.SubdomainScan) error {
	return r.DB.Create(scan).Error
}

func (r *Repository) FindLatestCleanScanHash(subdomainID uint) (string, time.Time, error) {
	var scan model.SubdomainScan
	err := r.DB.Where("subdomain_id = ? AND status = ?", subdomainID, model.ScanStatusClean).
		Order("created_at DESC").
		Limit(1).
		First(&scan).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}
	return scan.ContentHash, scan.CreatedAt, nil
}

func (r *Repository) HasEnabledAuditRuleUpdatedSince(t time.Time) (bool, error) {
	if t.IsZero() {
		return true, nil
	}
	var count int64
	err := r.DB.Model(&model.AuditRule{}).
		Where("enabled = ? AND updated_at > ?", true, t).
		Count(&count).Error
	return count > 0, err
}

type AuditScanListFilter struct {
	Statuses    []string
	SubdomainID *uint
	RuleIDs     []uint
	FQDN        string
	UserEmail   string
	From        *time.Time
	To          *time.Time
}

type AuditRuleHitStats struct {
	RuleID      uint
	HitCount7d  int64
	LastHitAt   *time.Time
	LastHitFQDN string
}

func (r *Repository) AdminListScans(page, perPage int, filter AuditScanListFilter) ([]model.SubdomainScan, int64, error) {
	q := r.DB.Model(&model.SubdomainScan{})
	if len(filter.Statuses) > 0 {
		q = q.Where("status IN ?", filter.Statuses)
	}
	if filter.SubdomainID != nil {
		q = q.Where("subdomain_id = ?", *filter.SubdomainID)
	}
	if filter.FQDN != "" {
		like := "%" + escapeLike(filter.FQDN) + "%"
		q = q.Where("fqdn ILIKE ?", like)
	}
	if len(filter.RuleIDs) > 0 {
		q = q.Where("matched_rule_id IN ?", filter.RuleIDs)
	}
	if filter.UserEmail != "" {
		like := "%" + escapeLike(filter.UserEmail) + "%"
		q = q.Where(`subdomain_id IN (
			SELECT s.id FROM subdomains s JOIN users u ON u.id = s.user_id WHERE u.email ILIKE ?
		)`, like)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var scans []model.SubdomainScan
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	err := q.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&scans).Error
	if scans == nil {
		scans = []model.SubdomainScan{}
	}
	return scans, total, err
}

func (r *Repository) ListAuditRuleHitStats(ruleIDs []uint) (map[uint]AuditRuleHitStats, error) {
	if len(ruleIDs) == 0 {
		return map[uint]AuditRuleHitStats{}, nil
	}
	since := time.Now().Add(-7 * 24 * time.Hour)
	type row struct {
		MatchedRuleID uint
		HitCount7d    int64
	}
	var counts []row
	if err := r.DB.Model(&model.SubdomainScan{}).
		Select("matched_rule_id, COUNT(*) AS hit_count7d").
		Where("status = ? AND matched_rule_id IN ? AND created_at >= ?", model.ScanStatusViolation, ruleIDs, since).
		Group("matched_rule_id").
		Scan(&counts).Error; err != nil {
		return nil, err
	}

	out := make(map[uint]AuditRuleHitStats, len(ruleIDs))
	for _, id := range ruleIDs {
		out[id] = AuditRuleHitStats{RuleID: id}
	}
	for _, c := range counts {
		s := out[c.MatchedRuleID]
		s.HitCount7d = c.HitCount7d
		out[c.MatchedRuleID] = s
	}

	for _, id := range ruleIDs {
		var scan model.SubdomainScan
		err := r.DB.Where("matched_rule_id = ? AND status = ?", id, model.ScanStatusViolation).
			Order("created_at DESC").Limit(1).First(&scan).Error
		if err == nil {
			s := out[id]
			t := scan.CreatedAt
			s.LastHitAt = &t
			s.LastHitFQDN = scan.FQDN
			out[id] = s
		}
	}
	return out, nil
}

func (r *Repository) FindSubdomainScan(id uint) (*model.SubdomainScan, error) {
	var scan model.SubdomainScan
	err := r.DB.First(&scan, id).Error
	return &scan, err
}

func (r *Repository) ListScansBySubdomain(subdomainID uint, page, perPage int) ([]model.SubdomainScan, int64, error) {
	q := r.DB.Model(&model.SubdomainScan{}).Where("subdomain_id = ?", subdomainID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var scans []model.SubdomainScan
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	err := q.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&scans).Error
	if scans == nil {
		scans = []model.SubdomainScan{}
	}
	return scans, total, err
}

func (r *Repository) ListDNSRecordsBySubdomainWithStatus(subdomainID uint, status string) ([]model.DNSRecord, error) {
	q := r.DB.Where("subdomain_id = ?", subdomainID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var records []model.DNSRecord
	err := q.Order("type ASC, name ASC").Find(&records).Error
	return records, err
}

func (r *Repository) ListSubdomainsByIDs(ids []uint) ([]model.Subdomain, error) {
	if len(ids) == 0 {
		return []model.Subdomain{}, nil
	}
	var subs []model.Subdomain
	err := r.DB.Where("id IN ?", ids).Preload("DNSRecords").Find(&subs).Error
	return subs, err
}

// --- Audit exemption pending ---

func (r *Repository) FindActiveExemptionPending(subdomainID, ruleID uint) (*model.AuditExemptionPending, error) {
	var item model.AuditExemptionPending
	err := r.DB.Where(
		"subdomain_id = ? AND rule_id = ? AND status IN ?",
		subdomainID, ruleID,
		[]string{model.AuditExemptionStatusPending, model.AuditExemptionStatusRechecking},
	).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) hasBlockingExemption(subdomainID, ruleID uint) (bool, error) {
	var count int64
	err := r.DB.Model(&model.AuditExemptionPending{}).
		Where("subdomain_id = ? AND rule_id = ? AND status IN ?",
			subdomainID, ruleID,
			[]string{model.AuditExemptionStatusPending, model.AuditExemptionStatusRechecking},
		).Count(&count).Error
	return count > 0, err
}

func (r *Repository) CreateExemptionPending(subdomainID, ruleID uint, recheckAt time.Time) error {
	exists, err := r.hasBlockingExemption(subdomainID, ruleID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return r.DB.Create(&model.AuditExemptionPending{
		SubdomainID: subdomainID,
		RuleID:      ruleID,
		RecheckAt:   recheckAt,
		Status:      model.AuditExemptionStatusPending,
	}).Error
}

func (r *Repository) ClaimDueExemptions(limit int) ([]model.AuditExemptionPending, error) {
	if limit <= 0 {
		limit = 50
	}
	var claimed []model.AuditExemptionPending
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var due []model.AuditExemptionPending
		if err := tx.Where("status = ? AND recheck_at <= ?", model.AuditExemptionStatusPending, time.Now()).
			Order("recheck_at ASC").
			Limit(limit).
			Find(&due).Error; err != nil {
			return err
		}
		for _, item := range due {
			res := tx.Model(&model.AuditExemptionPending{}).
				Where("id = ? AND status = ?", item.ID, model.AuditExemptionStatusPending).
				Update("status", model.AuditExemptionStatusRechecking)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				continue
			}
			claimed = append(claimed, item)
		}
		return nil
	})
	return claimed, err
}

func (r *Repository) CompleteExemptionPending(subdomainID, ruleID uint) error {
	res := r.DB.Model(&model.AuditExemptionPending{}).
		Where("subdomain_id = ? AND rule_id = ? AND status = ?",
			subdomainID, ruleID, model.AuditExemptionStatusRechecking).
		Update("status", model.AuditExemptionStatusCompleted)
	return res.Error
}

func (r *Repository) FindSubdomainFQDNByID(id uint) (string, error) {
	var fqdn string
	err := r.DB.Model(&model.Subdomain{}).Where("id = ?", id).Pluck("fqdn", &fqdn).Error
	return fqdn, err
}
