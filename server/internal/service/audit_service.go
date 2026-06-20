package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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

	probe := s.fetcher.FetchDualConfirmed(ctx, fqdn)
	dual := probe.Primary
	primaryCh := dual.PrimaryChannel()
	combinedHash := audit.CombinedContentHash(dual)

	scan := &model.SubdomainScan{
		SubdomainID:    target.ID,
		FQDN:           fqdn,
		URL:            primaryCh.RequestURL,
		HTTPStatusCode: primaryCh.StatusCode,
		FinalURL:       primaryCh.FinalURL,
		ContentHash:    combinedHash,
		FetchDetails:   buildFetchDetails(dual),
		MatchedRules:   model.MatchedRulesSlice{},
	}
	scan.Status = s.scanStatusFromProbe(probe)

	skipRuleMatch := probe.SkipRuleMatch
	if combinedHash != "" {
		prevHash, prevAt, hashErr := s.repo.FindLatestCleanScanHash(target.ID)
		if hashErr == nil && prevHash == combinedHash {
			updated, ruleErr := s.repo.HasEnabledAuditRuleUpdatedSince(prevAt)
			if ruleErr == nil && !updated {
				skipRuleMatch = true
			}
		}
	}

	if !skipRuleMatch && !audit.HasPrivateIPError(dual) {
		matches, matchErr := s.engine.MatchAll(ctx, sub.DomainID, probe)
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

				if s.tryHandleViolationExemption(ctx, sub, primary.Rule, primary.Snippet) {
					return
				}

				switch primary.Rule.Action {
				case model.AuditActionUser:
					s.suspendUserSubdomains(ctx, sub.UserID, primary.Rule, primary.Snippet)
				case model.AuditActionSite:
					s.suspendSubdomain(ctx, sub, primary.Rule, primary.Snippet)
				case model.AuditActionDeleteDNS:
					s.deleteScannableRecords(ctx, sub, primary.Rule, primary.Snippet)
				case model.AuditActionObserve:
					s.notifyBanIfConfigured(sub.UserID, sub.FQDN, primary.Rule, primary.Snippet)
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

func (s *AuditService) deleteScannableRecords(ctx context.Context, sub *model.Subdomain, rule *model.AuditRule, snippet string) bool {
	var records []model.DNSRecord
	for _, rec := range sub.DNSRecords {
		if rec.Status == model.DNSRecordStatusActive && auditScannableRecordTypes[rec.Type] {
			records = append(records, rec)
		}
	}
	if len(records) == 0 {
		return true
	}

	slog.Warn("audit: deleting scannable DNS records", "fqdn", sub.FQDN, "rule", rule.Name, "count", len(records))

	for i := range records {
		rec := records[i]
		if err := s.ops.DeleteRecordAtomic(ctx, DeleteRecordInput{
			Subdomain: sub,
			Record:    &rec,
		}); err != nil {
			slog.Error("audit: delete scannable record failed, aborting",
				"fqdn", sub.FQDN, "record_type", rec.Type, "record_name", rec.Name, "err", err,
			)
			return false
		}
	}

	_ = s.auditLog.RecordUser(sub.UserID, "audit_delete_dns", "subdomain", sub.ID, map[string]interface{}{
		"fqdn":            sub.FQDN,
		"rule":            rule.Name,
		"rule_id":         rule.ID,
		"matched_snippet": helpers.TruncateRunes(snippet, 100),
		"deleted_count":   len(records),
	})
	s.notifyBanIfConfigured(sub.UserID, sub.FQDN, rule, snippet)
	return true
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

	s.notifyBanIfConfigured(sub.UserID, sub.FQDN, rule, snippet)
	return true
}

func (s *AuditService) tryHandleViolationExemption(ctx context.Context, sub *model.Subdomain, rule *model.AuditRule, snippet string) bool {
	if !rule.ExemptEnabled {
		return false
	}
	_ = ctx

	pending, err := s.repo.FindActiveExemptionPending(sub.ID, rule.ID)
	if err != nil {
		slog.Error("audit: find exemption pending failed", "fqdn", sub.FQDN, "rule_id", rule.ID, "err", err)
		return false
	}
	if pending != nil {
		slog.Info("audit: exemption pending, skipping enforcement", "fqdn", sub.FQDN, "rule", rule.Name)
		return true
	}

	recheckAt := time.Now().Add(time.Duration(rule.ExemptRecheckMinutes) * time.Minute)
	if err := s.repo.CreateExemptionPending(sub.ID, rule.ID, recheckAt); err != nil {
		slog.Error("audit: create exemption pending failed", "fqdn", sub.FQDN, "rule_id", rule.ID, "err", err)
		return false
	}
	slog.Info("audit: exemption created", "fqdn", sub.FQDN, "rule", rule.Name, "recheck_at", recheckAt)

	if content := strings.TrimSpace(rule.ExemptNotifyContent); content != "" {
		s.sendAuditCustomNotification(sub.UserID, sub.FQDN, rule, snippet, content, recheckAt)
	}
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

func (s *AuditService) scanStatusFromProbe(probe audit.DualFetchProbeResult) string {
	if audit.HasPrivateIPError(probe.Primary) {
		return model.ScanStatusError
	}
	if probe.ConfirmedUnreachable() {
		return model.ScanStatusUnreachable
	}
	return model.ScanStatusClean
}

func buildFetchDetails(dual audit.DualFetchResult) *model.DualFetchDetailsJSON {
	details := model.DualFetchDetails{
		HTTPS: channelDetail(dual.HTTPS),
		HTTP:  channelDetail(dual.HTTP),
	}
	out := model.DualFetchDetailsJSON(details)
	return &out
}

func channelDetail(ch audit.ChannelResult) model.FetchChannelDetail {
	title := ch.Title
	if len(title) > 120 {
		title = title[:120]
	}
	return model.FetchChannelDetail{
		Scheme:         ch.Scheme,
		RequestURL:     ch.RequestURL,
		Status:         ch.Status,
		HTTPStatusCode: ch.StatusCode,
		FinalURL:       ch.FinalURL,
		ErrorMessage:   ch.ErrorMessage,
		TitlePreview:   title,
	}
}

func (s *AuditService) notifyBanIfConfigured(userID uint, fqdn string, rule *model.AuditRule, snippet string) {
	content := strings.TrimSpace(rule.BanNotifyContent)
	if content == "" {
		return
	}
	s.sendAuditCustomNotification(userID, fqdn, rule, snippet, content, time.Time{})
}

func (s *AuditService) sendAuditCustomNotification(userID uint, fqdn string, rule *model.AuditRule, snippet, tmpl string, recheckAt time.Time) {
	rendered := audit.RenderNotifyTemplate(tmpl, audit.NotifyTemplateVars{
		FQDN:           fqdn,
		RuleName:       rule.Name,
		MatchedSnippet: helpers.TruncateRunes(snippet, 200),
		Action:         rule.Action,
		RecheckMinutes: rule.ExemptRecheckMinutes,
		RecheckAt:      recheckAt,
	})
	if !audit.ValidateNotifyTemplateContent(rendered) {
		slog.Warn("audit: notification content too long after render", "fqdn", fqdn, "rule", rule.Name)
		return
	}
	_, err := s.notif.NotifyUsers(
		[]uint{userID},
		0,
		"urgent",
		fqdn,
		rendered,
		"",
		nil,
	)
	if err != nil {
		slog.Error("audit: custom notification failed", "fqdn", fqdn, "err", err)
	}
}

func (s *AuditService) TestRuleMatch(ctx context.Context, fqdn string, domainID uint, rules []model.AuditRule) (audit.DualFetchResult, []MatchedRule, *MatchedRule) {
	probe := s.fetcher.FetchDualConfirmed(ctx, fqdn)
	matches := s.engine.MatchAllProbeWithRules(domainID, probe, rules)
	return probe.Primary, matches, PickPrimaryMatch(matches)
}
