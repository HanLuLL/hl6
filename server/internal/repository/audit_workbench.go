package repository

import (
	"time"

	"hl6-server/internal/model"
	"hl6-server/pkg/audit"
)

// AuditSummary 工作台顶部统计。
type AuditSummary struct {
	DeletedCount      int64 `json:"deleted_count"`
	CurrentViolation  int64 `json:"current_violation"`
	NeverScannedCount int64 `json:"never_scanned_count"`
	EnabledRulesCount int64 `json:"enabled_rules_count"`
}

func (r *Repository) GetAuditSummary() (*AuditSummary, error) {
	var summary AuditSummary

	if err := r.DB.Model(&model.AuditLog{}).
		Where("action IN ?", []string{"audit_release_subdomain", "audit_delete_dns"}).
		Count(&summary.DeletedCount).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Raw(`
		SELECT COUNT(*)
		FROM subdomains s
		JOIN LATERAL (
			SELECT ss.status
			FROM subdomain_scans ss
			WHERE ss.subdomain_id = s.id
			ORDER BY ss.created_at DESC
			LIMIT 1
		) latest_scan ON true
		WHERE s.status = ?
		  AND latest_scan.status = ?
	`, model.SubdomainStatusActive, model.ScanStatusViolation).Scan(&summary.CurrentViolation).Error; err != nil {
		return nil, err
	}
	if err := r.DB.Model(&model.AuditRule{}).
		Where("enabled = ?", true).
		Count(&summary.EnabledRulesCount).Error; err != nil {
		return nil, err
	}

	err := r.DB.Raw(`
		SELECT COUNT(DISTINCT s.id)
		FROM subdomains s
		JOIN dns_records dr ON dr.subdomain_id = s.id
			AND dr.status = ? AND dr.type IN ?
		WHERE s.status = ?
		  AND NOT EXISTS (SELECT 1 FROM subdomain_scans ss WHERE ss.subdomain_id = s.id)
	`, model.DNSRecordStatusActive, audit.ScannableRecordTypes, model.SubdomainStatusActive).Scan(&summary.NeverScannedCount).Error
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// AuditWorkbenchScanBrief 最新扫描摘要。
type AuditWorkbenchScanBrief struct {
	ID             uint      `json:"id"`
	Status         string    `json:"status"`
	HTTPStatusCode int       `json:"http_status_code"`
	CreatedAt      time.Time `json:"created_at"`
}

// AuditWorkbenchViolationBrief 最新违规摘要。
type AuditWorkbenchViolationBrief struct {
	ID               uint                    `json:"id"`
	MatchedRuleID    *uint                   `json:"matched_rule_id"`
	MatchedRuleName  string                  `json:"matched_rule_name"`
	MatchedSnippet   string                  `json:"matched_snippet"`
	CreatedAt        time.Time               `json:"created_at"`
	MatchedRules     model.MatchedRulesSlice `json:"matched_rules"`
}

// AuditWorkbenchItem 工作台列表行。
type AuditWorkbenchItem struct {
	SubdomainID      uint                          `json:"subdomain_id"`
	FQDN             string                        `json:"fqdn"`
	DomainID         uint                          `json:"domain_id"`
	DomainName       string                        `json:"domain_name"`
	UserID           uint                          `json:"user_id"`
	UserEmail        string                        `json:"user_email"`
	Status           string                        `json:"status"`
	SuspendedReason  string                        `json:"suspended_reason,omitempty"`
	SuspendedAt      *time.Time                    `json:"suspended_at,omitempty"`
	DNSRecordCount   int64                         `json:"dns_record_count"`
	LatestScan       *AuditWorkbenchScanBrief      `json:"latest_scan"`
	LatestViolation  *AuditWorkbenchViolationBrief `json:"latest_violation"`
	ViolationCount7d int64                         `json:"violation_count_7d"`
	ContentChanged   bool                          `json:"content_changed"`
	Scannable        bool                          `json:"scannable"`
}

// AuditCasesFilter 子域名列表筛选。
type AuditCasesFilter struct {
	Statuses       []string
	ScanStatus     string
	RuleIDs        []uint
	DomainIDs      []uint
	GroupID        *uint
	Search         string
	UserEmail      string
	FQDN           string
	SuspendedFrom  *time.Time
	SuspendedTo    *time.Time
	ScanFrom       *time.Time
	ScanTo         *time.Time
	Sort           string
	RecordTypes    []string
}

func (r *Repository) ListAuditCases(page, perPage int, filter AuditCasesFilter) ([]AuditWorkbenchItem, int64, error) {
	base := `
FROM subdomains s
JOIN domains d ON d.id = s.domain_id
JOIN users u ON u.id = s.user_id
LEFT JOIN LATERAL (
	SELECT ss.id, ss.status, ss.http_status_code, ss.created_at
	FROM subdomain_scans ss WHERE ss.subdomain_id = s.id
	ORDER BY ss.created_at DESC LIMIT 1
) latest_scan ON true
LEFT JOIN LATERAL (
	SELECT ss.id, ss.matched_rule_id, ss.matched_snippet, ss.created_at, ss.matched_rules,
		ar.name AS matched_rule_name
	FROM subdomain_scans ss
	LEFT JOIN audit_rules ar ON ar.id = ss.matched_rule_id
	WHERE ss.subdomain_id = s.id
	  AND ss.id = latest_scan.id
	  AND latest_scan.status = 'violation'
) latest_violation ON true
`
	args := []interface{}{}
	where := "WHERE 1=1"

	if len(filter.Statuses) > 0 {
		where += " AND s.status IN ?"
		args = append(args, filter.Statuses)
	}
	if len(filter.DomainIDs) > 0 {
		where += " AND s.domain_id IN ?"
		args = append(args, filter.DomainIDs)
	}
	if filter.GroupID != nil {
		where += " AND u.group_id = ?"
		args = append(args, *filter.GroupID)
	}
	if filter.Search != "" {
		like := "%" + escapeLike(filter.Search) + "%"
		where += " AND (s.fqdn ILIKE ? OR u.email ILIKE ?)"
		args = append(args, like, like)
	} else {
		if filter.UserEmail != "" {
			where += " AND u.email ILIKE ?"
			args = append(args, "%"+escapeLike(filter.UserEmail)+"%")
		}
		if filter.FQDN != "" {
			where += " AND s.fqdn ILIKE ?"
			args = append(args, "%"+escapeLike(filter.FQDN)+"%")
		}
	}
	if filter.SuspendedFrom != nil {
		where += " AND s.suspended_at >= ?"
		args = append(args, *filter.SuspendedFrom)
	}
	if filter.SuspendedTo != nil {
		where += " AND s.suspended_at <= ?"
		args = append(args, *filter.SuspendedTo)
	}
	if filter.ScanFrom != nil {
		where += " AND latest_scan.created_at >= ?"
		args = append(args, *filter.ScanFrom)
	}
	if filter.ScanTo != nil {
		where += " AND latest_scan.created_at <= ?"
		args = append(args, *filter.ScanTo)
	}
	if len(filter.RuleIDs) > 0 {
		where += " AND latest_violation.matched_rule_id IN ?"
		args = append(args, filter.RuleIDs)
	}
	where = appendAuditSiteScanFilter(where, filter.ScanStatus)
	if len(filter.RecordTypes) > 0 {
		where += ` AND EXISTS (
			SELECT 1 FROM dns_records dr
			WHERE dr.subdomain_id = s.id AND dr.status = ? AND dr.type IN ?
		)`
		args = append(args, model.DNSRecordStatusActive, filter.RecordTypes)
	}

	countSQL := "SELECT COUNT(*) " + base + where
	var total int64
	if err := r.DB.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := "latest_scan.created_at DESC NULLS LAST"
	switch filter.Sort {
	case "suspended_at_desc":
		orderBy = "s.suspended_at DESC NULLS LAST"
	case "fqdn_asc":
		orderBy = "s.fqdn ASC"
	}

	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	selectSQL := `
SELECT
	s.id AS subdomain_id,
	s.fqdn,
	s.domain_id,
	d.name AS domain_name,
	s.user_id,
	u.email AS user_email,
	s.status,
	s.suspended_reason,
	s.suspended_at,
	(SELECT COUNT(*) FROM dns_records dr WHERE dr.subdomain_id = s.id) AS dns_record_count,
	latest_scan.id AS latest_scan_id,
	latest_scan.status AS latest_scan_status,
	latest_scan.http_status_code AS latest_scan_http_status_code,
	latest_scan.created_at AS latest_scan_created_at,
	latest_violation.id AS latest_violation_id,
	latest_violation.matched_rule_id,
	latest_violation.matched_rule_name,
	latest_violation.matched_snippet,
	latest_violation.created_at AS latest_violation_created_at,
	latest_violation.matched_rules,
	(SELECT COUNT(*) FROM subdomain_scans ss
	 WHERE ss.subdomain_id = s.id AND ss.status = ? AND ss.created_at >= ?) AS violation_count_7d,
	EXISTS (
		SELECT 1 FROM dns_records dr
		WHERE dr.subdomain_id = s.id
		  AND dr.status = ?
		  AND dr.type IN ?
	) AS scannable,
		(SELECT COUNT(*) FROM subdomain_scans WHERE subdomain_id = s.id AND status = ?) >= 2
		AND (SELECT content_hash FROM subdomain_scans WHERE subdomain_id = s.id AND status = ? ORDER BY created_at DESC LIMIT 1)
		<> (SELECT content_hash FROM subdomain_scans WHERE subdomain_id = s.id AND status = ? ORDER BY created_at DESC OFFSET 1 LIMIT 1)
	AS content_changed
` + base + where + " ORDER BY " + orderBy + " OFFSET ? LIMIT ?"

	since7d := time.Now().Add(-7 * 24 * time.Hour)
	// SELECT 子句中的 ? 在 base/where 之前，参数须先于 lateral join 与筛选条件。
	queryArgs := []interface{}{
		model.ScanStatusViolation, since7d,
		model.DNSRecordStatusActive, audit.ScannableRecordTypes,
		model.ScanStatusClean, model.ScanStatusClean, model.ScanStatusClean,
	}
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, offset, perPage)

	type row struct {
		SubdomainID              uint
		FQDN                     string
		DomainID                 uint
		DomainName               string
		UserID                   uint
		UserEmail                string
		Status                   string
		SuspendedReason          string
		SuspendedAt              *time.Time
		DNSRecordCount           int64
		LatestScanID             *uint
		LatestScanStatus         *string
		LatestScanHTTPStatusCode *int
		LatestScanCreatedAt      *time.Time
		LatestViolationID        *uint
		MatchedRuleID            *uint
		MatchedRuleName          *string
		MatchedSnippet           *string
		LatestViolationCreatedAt *time.Time
		MatchedRules             model.MatchedRulesSlice
		ViolationCount7d         int64
		Scannable                bool
		ContentChanged           bool
	}

	var rows []row
	if err := r.DB.Raw(selectSQL, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]AuditWorkbenchItem, 0, len(rows))
	for _, rw := range rows {
		item := AuditWorkbenchItem{
			SubdomainID:      rw.SubdomainID,
			FQDN:             rw.FQDN,
			DomainID:         rw.DomainID,
			DomainName:       rw.DomainName,
			UserID:           rw.UserID,
			UserEmail:        rw.UserEmail,
			Status:           rw.Status,
			SuspendedReason:  rw.SuspendedReason,
			SuspendedAt:      rw.SuspendedAt,
			DNSRecordCount:   rw.DNSRecordCount,
			ViolationCount7d: rw.ViolationCount7d,
			ContentChanged:   rw.ContentChanged,
			Scannable:        rw.Scannable,
		}
		if rw.LatestScanID != nil {
			item.LatestScan = &AuditWorkbenchScanBrief{
				ID:             *rw.LatestScanID,
				Status:         derefString(rw.LatestScanStatus),
				HTTPStatusCode: derefInt(rw.LatestScanHTTPStatusCode),
				CreatedAt:      derefTime(rw.LatestScanCreatedAt),
			}
		}
		if rw.LatestViolationID != nil {
			item.LatestViolation = &AuditWorkbenchViolationBrief{
				ID:              *rw.LatestViolationID,
				MatchedRuleID:   rw.MatchedRuleID,
				MatchedRuleName: derefString(rw.MatchedRuleName),
				MatchedSnippet:  derefString(rw.MatchedSnippet),
				CreatedAt:       derefTime(rw.LatestViolationCreatedAt),
				MatchedRules:    rw.MatchedRules,
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

func appendAuditSiteScanFilter(where string, scanStatus string) string {
	switch scanStatus {
	case "never":
		return where + " AND latest_scan.id IS NULL"
	case "scanned":
		return where + " AND latest_scan.id IS NOT NULL"
	case "violation":
		return where + " AND latest_scan.status = 'violation'"
	case "compliant":
		return where + " AND latest_scan.status = 'clean'"
	default:
		return where
	}
}

// AuditSubdomainSibling 同用户其他子域名摘要。
type AuditSubdomainSibling struct {
	ID              uint       `json:"id"`
	FQDN            string     `json:"fqdn"`
	Status          string     `json:"status"`
	SuspendedReason string     `json:"suspended_reason,omitempty"`
	SuspendedAt     *time.Time `json:"suspended_at,omitempty"`
}

// AuditSubdomainDetailBundle 子域名详情。
type AuditSubdomainDetailBundle struct {
	Subdomain         model.Subdomain               `json:"subdomain"`
	UserEmail         string                        `json:"user_email"`
	Scannable         bool                          `json:"scannable"`
	LatestViolation   *AuditWorkbenchViolationBrief `json:"latest_violation"`
	SiblingSubdomains []AuditSubdomainSibling       `json:"sibling_subdomains"`
	DNSRecords        []model.DNSRecord             `json:"dns_records"`
}

func (r *Repository) GetAuditSubdomainDetail(id uint) (*AuditSubdomainDetailBundle, error) {
	sub, err := r.FindSubdomain(id)
	if err != nil {
		return nil, err
	}

	var user model.User
	_ = r.DB.First(&user, sub.UserID).Error
	sub.User = user

	var siblings []AuditSubdomainSibling
	_ = r.DB.Model(&model.Subdomain{}).
		Select("id, fqdn, status, suspended_reason, suspended_at").
		Where("user_id = ? AND id <> ?", sub.UserID, sub.ID).
		Order("fqdn ASC").
		Scan(&siblings).Error
	if siblings == nil {
		siblings = []AuditSubdomainSibling{}
	}

	records, err := r.ListDNSRecordsBySubdomainWithStatus(sub.ID, "")
	if err != nil {
		return nil, err
	}

	var violation AuditWorkbenchViolationBrief
	vErr := r.DB.Raw(`
		SELECT ss.id, ss.matched_rule_id, ss.matched_snippet, ss.created_at, ss.matched_rules,
			ar.name AS matched_rule_name
		FROM subdomain_scans ss
		LEFT JOIN audit_rules ar ON ar.id = ss.matched_rule_id
		JOIN LATERAL (
			SELECT ls.id, ls.status
			FROM subdomain_scans ls
			WHERE ls.subdomain_id = ss.subdomain_id
			ORDER BY ls.created_at DESC
			LIMIT 1
		) latest ON latest.id = ss.id AND latest.status = ?
		WHERE ss.subdomain_id = ?
	`, model.ScanStatusViolation, sub.ID).Scan(&violation).Error

	bundle := &AuditSubdomainDetailBundle{
		Subdomain:         *sub,
		UserEmail:         user.Email,
		Scannable:         audit.SubdomainHasScannableActiveDNS(records),
		SiblingSubdomains: siblings,
		DNSRecords:        records,
	}
	if vErr == nil && violation.ID != 0 {
		bundle.LatestViolation = &violation
	}
	return bundle, nil
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}
