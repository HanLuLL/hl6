package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/audit"
)

var auditScannableRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true,
}

// AuditService 封装内容合规巡检与处置（Suspend/Restore/Scan）。
type AuditService struct {
	repo     *repository.Repository
	ops      *DNSOperationService
	notif    *NotificationService
	fetcher  *audit.SafeFetcher
	engine   *AuditRuleEngine
	auditLog *AuditLogService
}

func NewAuditService(repo *repository.Repository, ops *DNSOperationService, notif *NotificationService, fetchTimeout time.Duration, auditLog *AuditLogService) *AuditService {
	opts := make([]audit.SafeFetcherOption, 0)
	if fetchTimeout > 0 {
		opts = append(opts, audit.WithTimeout(fetchTimeout))
	}
	return &AuditService{
		repo:     repo,
		ops:      ops,
		notif:    notif,
		fetcher:  audit.NewSafeFetcher(opts...),
		engine:   NewAuditRuleEngine(repo),
		auditLog: auditLog,
	}
}

func (s *AuditService) Fetcher() *audit.SafeFetcher { return s.fetcher }
func (s *AuditService) Engine() *AuditRuleEngine    { return s.engine }

func (s *AuditService) ScanSubdomain(ctx context.Context, target model.AuditScanTarget) {
	fqdn := target.FQDN

	sub, err := s.repo.FindSubdomain(target.ID)
	if err != nil {
		slog.Warn("audit scan: subdomain not found", "id", target.ID, "fqdn", fqdn, "err", err)
		return
	}
	if sub.Status != model.SubdomainStatusActive {
		return
	}
	if !hasScannableActiveDNS(sub) {
		return
	}

	fetchResult := s.fetcher.Fetch(ctx, fqdn)

	scan := &model.SubdomainScan{
		SubdomainID:    target.ID,
		FQDN:           fqdn,
		URL:            "https://" + fqdn,
		HTTPStatusCode: fetchResult.StatusCode,
		FinalURL:       fetchResult.FinalURL,
		ContentHash:    fetchResult.ContentHash,
		MatchedRules:   model.MatchedRulesSlice{},
	}
	scan.Status = s.scanStatusFromFetch(fetchResult)

	skipRuleMatch := false
	if fetchResult.Status == audit.FetchStatusClean && fetchResult.ContentHash != "" {
		prevHash, prevAt, hashErr := s.repo.FindLatestCleanScanHash(target.ID)
		if hashErr == nil && prevHash == fetchResult.ContentHash {
			updated, ruleErr := s.repo.HasEnabledAuditRuleUpdatedSince(prevAt)
			if ruleErr == nil && !updated {
				skipRuleMatch = true
			}
		}
	}

	if fetchResult.Status == audit.FetchStatusClean && !skipRuleMatch {
		matches, matchErr := s.engine.MatchAll(ctx, sub.DomainID, fetchResult)
		if matchErr != nil {
			slog.Warn("audit scan: rule matching error", "fqdn", fqdn, "err", matchErr)
		}
		if len(matches) > 0 {
			primary := PickPrimaryMatch(matches)
			scan.Status = model.ScanStatusViolation
			scan.MatchedRules = ToMatchedRuleHits(matches)
			if primary != nil {
				scan.MatchedRuleID = &primary.Rule.ID
				scan.MatchedSnippet = primary.Snippet

				if err := s.repo.CreateSubdomainScan(scan); err != nil {
					slog.Error("audit scan: persist scan record failed", "fqdn", fqdn, "err", err)
					return
				}

				switch primary.Rule.Action {
				case model.AuditActionUser:
					s.suspendUserSubdomains(ctx, sub.UserID, primary.Rule, primary.Snippet)
				case model.AuditActionSite:
					s.suspendSubdomain(ctx, sub, primary.Rule, primary.Snippet)
				case model.AuditActionObserve:
					_ = s.auditLog.RecordUser(sub.UserID, "audit_observe_violation", "subdomain", sub.ID, map[string]interface{}{
						"fqdn":            sub.FQDN,
						"rule":            primary.Rule.Name,
						"rule_id":         primary.Rule.ID,
						"matched_snippet": helpers.TruncateRunes(primary.Snippet, 100),
					})
				}
				return
			}
		}
	}

	if err := s.repo.CreateSubdomainScan(scan); err != nil {
		slog.Error("audit scan: persist scan record failed", "fqdn", fqdn, "err", err)
	}
}

func hasScannableActiveDNS(sub *model.Subdomain) bool {
	for _, rec := range sub.DNSRecords {
		if rec.Status == model.DNSRecordStatusActive && auditScannableRecordTypes[rec.Type] {
			return true
		}
	}
	return false
}

func (s *AuditService) suspendSubdomain(ctx context.Context, sub *model.Subdomain, rule *model.AuditRule, snippet string) bool {
	if sub.Status == model.SubdomainStatusSuspended {
		return true
	}

	slog.Warn("audit: suspending subdomain", "fqdn", sub.FQDN, "rule", rule.Name, "action", rule.Action)

	var activeRecords []model.DNSRecord
	for _, rec := range sub.DNSRecords {
		if rec.Status == model.DNSRecordStatusActive {
			activeRecords = append(activeRecords, rec)
		}
	}

	var deletedRecords []model.DNSRecord
	for i := range activeRecords {
		rec := activeRecords[i]
		if err := s.deleteProviderRecord(ctx, sub, &rec); err != nil {
			slog.Error("audit: delete provider record failed, aborting suspend",
				"fqdn", sub.FQDN, "record_type", rec.Type, "record_name", rec.Name, "err", err,
			)
			if compErr := s.compensateProviderRecords(ctx, sub, deletedRecords); compErr != nil {
				slog.Error("audit: compensate provider records after partial delete failed",
					"fqdn", sub.FQDN, "err", compErr,
				)
			}
			return false
		}
		deletedRecords = append(deletedRecords, rec)
	}

	now := time.Now()
	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		for _, rec := range activeRecords {
			if err := s.repo.UpdateDNSRecordStatus(tx, rec.ID, model.DNSRecordStatusSuspended, ""); err != nil {
				return fmt.Errorf("update dns record status: %w", err)
			}
		}
		if err := s.repo.UpdateSubdomainStatusFunc(tx, sub.ID, model.SubdomainStatusSuspended, rule.Name, &now); err != nil {
			return fmt.Errorf("update subdomain status: %w", err)
		}
		return s.auditLog.RecordUserTx(tx, sub.UserID, "audit_suspend_subdomain", "subdomain", sub.ID, map[string]interface{}{
			"fqdn":            sub.FQDN,
			"rule":            rule.Name,
			"rule_id":         rule.ID,
			"action":          rule.Action,
			"matched_snippet": helpers.TruncateRunes(snippet, 100),
		})
	}); err != nil {
		slog.Error("audit: suspend transaction failed", "fqdn", sub.FQDN, "err", err)
		if compErr := s.compensateProviderRecords(ctx, sub, deletedRecords); compErr != nil {
			slog.Error("audit: compensate provider records after transaction failed",
				"fqdn", sub.FQDN, "err", compErr,
			)
		}
		return false
	}

	s.notifySuspension(sub.UserID, sub.FQDN, rule.Name)
	return true
}

func (s *AuditService) suspendUserSubdomains(ctx context.Context, userID uint, rule *model.AuditRule, snippet string) {
	slog.Warn("audit: suspending all user subdomains", "user_id", userID, "rule", rule.Name)

	subs, err := s.repo.ListActiveSubdomainsByUser(userID)
	if err != nil {
		slog.Error("audit: list user subdomains failed", "user_id", userID, "err", err)
		return
	}

	var failed []string
	success := 0
	for i := range subs {
		if s.suspendSubdomain(ctx, &subs[i], rule, snippet) {
			success++
		} else {
			failed = append(failed, subs[i].FQDN)
		}
	}
	if len(failed) > 0 {
		slog.Warn("audit: user-level suspend incomplete",
			"user_id", userID, "success", success, "failed_fqdns", failed,
		)
	}
}

func (s *AuditService) deleteProviderRecord(ctx context.Context, sub *model.Subdomain, rec *model.DNSRecord) error {
	client, _, err := s.ops.providerClientForAccount(sub.Domain.ProviderAccountID, sub.Domain.Provider)
	if err != nil {
		return fmt.Errorf("get provider client: %w", err)
	}

	recordID := rec.ProviderRecordID
	if recordID == "" {
		foundID, findErr := client.FindRecord(ctx, sub.Domain.ProviderZoneID, rec.Type, rec.Name, rec.Content)
		if findErr != nil {
			return fmt.Errorf("find record: %w", findErr)
		}
		recordID = foundID
	}

	if recordID != "" {
		if err := client.DeleteRecord(ctx, sub.Domain.ProviderZoneID, recordID); err != nil {
			return fmt.Errorf("delete provider record %s: %w", recordID, err)
		}
	}

	return nil
}

func (s *AuditService) recreateProviderRecord(ctx context.Context, sub *model.Subdomain, rec *model.DNSRecord) (string, error) {
	client, _, err := s.ops.providerClientForAccount(sub.Domain.ProviderAccountID, sub.Domain.Provider)
	if err != nil {
		return "", fmt.Errorf("get provider client: %w", err)
	}
	providerRecordID, err := client.CreateRecord(ctx,
		sub.Domain.ProviderZoneID,
		rec.Type, rec.Name, rec.Content,
		rec.TTL, rec.Proxied,
	)
	if err != nil {
		return "", fmt.Errorf("recreate provider record %s %s: %w", rec.Type, rec.Name, err)
	}
	return providerRecordID, nil
}

func (s *AuditService) compensateProviderRecords(ctx context.Context, sub *model.Subdomain, records []model.DNSRecord) error {
	if len(records) == 0 {
		return nil
	}
	var failed int
	for i := range records {
		rec := records[i]
		if _, err := s.recreateProviderRecord(ctx, sub, &rec); err != nil {
			failed++
			slog.Error("audit: compensate recreate provider record failed",
				"fqdn", sub.FQDN, "record_type", rec.Type, "record_name", rec.Name, "err", err,
			)
		}
	}
	if failed > 0 {
		return fmt.Errorf("compensate failed for %d of %d records", failed, len(records))
	}
	return nil
}

func (s *AuditService) RestoreSubdomain(ctx context.Context, sub *model.Subdomain) error {
	if sub.Status != model.SubdomainStatusSuspended {
		return fmt.Errorf("subdomain %s is not suspended", sub.FQDN)
	}

	slog.Info("audit: restoring subdomain", "fqdn", sub.FQDN)

	records, err := s.repo.ListDNSRecordsBySubdomainWithStatus(sub.ID, model.DNSRecordStatusSuspended)
	if err != nil {
		return fmt.Errorf("list suspended records: %w", err)
	}

	type restored struct {
		recordID         uint
		providerRecordID string
	}

	restoredRecords := make([]restored, 0, len(records))
	for _, rec := range records {
		newProviderRecordID, createErr := s.recreateProviderRecord(ctx, sub, &rec)
		if createErr != nil {
			return createErr
		}
		restoredRecords = append(restoredRecords, restored{recordID: rec.ID, providerRecordID: newProviderRecordID})
	}

	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		for _, r := range restoredRecords {
			if err := s.repo.UpdateDNSRecordStatus(tx, r.recordID, model.DNSRecordStatusActive, r.providerRecordID); err != nil {
				return fmt.Errorf("restore dns record status: %w", err)
			}
		}
		return s.repo.RestoreSubdomainStatus(tx, sub.ID)
	}); err != nil {
		return fmt.Errorf("restore transaction: %w", err)
	}

	slog.Info("audit: subdomain restored", "fqdn", sub.FQDN)
	return nil
}

func (s *AuditService) scanStatusFromFetch(fr audit.FetchResult) string {
	switch fr.Status {
	case audit.FetchStatusClean:
		return model.ScanStatusClean
	case audit.FetchStatusUnreachable:
		return model.ScanStatusUnreachable
	default:
		return model.ScanStatusError
	}
}

func (s *AuditService) notifySuspension(userID uint, fqdn, ruleName string) {
	args, _ := json.Marshal(map[string]any{"fqdn": fqdn, "rule": ruleName})
	_, err := s.notif.NotifyUsers(
		[]uint{userID},
		0,
		"urgent",
		fqdn,
		" ",
		"notification.subdomainSuspended",
		args,
	)
	if err != nil {
		slog.Error("audit: notification failed", "fqdn", fqdn, "err", err)
	}
}

func (s *AuditService) TestRuleMatch(ctx context.Context, fqdn string, domainID uint, rules []model.AuditRule) (audit.FetchResult, []MatchedRule, *MatchedRule) {
	fr := s.fetcher.Fetch(ctx, fqdn)
	matches := s.engine.MatchAllWithRules(domainID, fr, rules)
	return fr, matches, PickPrimaryMatch(matches)
}
