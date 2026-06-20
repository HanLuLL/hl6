package service

import (
	"context"
	"errors"
	"net/http"

	"gorm.io/gorm"
	"hl6-server/internal/apperr"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
)

// ReleaseOpts 捕获用户发起与管理端发起释放的差异。
type ReleaseOpts struct {
	ActorID     uint
	AuditAction string
	AuditExtra  map[string]interface{}
}

// SubdomainService 拥有子域名的写侧业务逻辑。
type SubdomainService struct {
	repo     *repository.Repository
	ops      *DNSOperationService
	auditLog *AuditLogService
}

func NewSubdomainService(repo *repository.Repository, ops *DNSOperationService, auditLog *AuditLogService) *SubdomainService {
	return &SubdomainService{repo: repo, ops: ops, auditLog: auditLog}
}

func BuildBatchDeleteItemsForSubdomain(sub *model.Subdomain) []BatchDeleteItem {
	if sub == nil {
		return nil
	}
	items := make([]BatchDeleteItem, 0, len(sub.DNSRecords))
	for _, rec := range sub.DNSRecords {
		items = append(items, BatchDeleteItem{
			RecordID:          rec.ID,
			SubdomainFQDN:     sub.FQDN,
			Provider:          sub.Domain.Provider,
			ProviderAccountID: sub.Domain.ProviderAccountID,
			ZoneID:            sub.Domain.ProviderZoneID,
			ProviderRecordID:  rec.ProviderRecordID,
			RecordType:        rec.Type,
			Name:              rec.Name,
			Content:           rec.Content,
			TTL:               rec.TTL,
			Proxied:           rec.Proxied,
		})
	}
	return items
}

func ToCFFailureRecords(failures []BatchDeleteFailure) []map[string]string {
	out := make([]map[string]string, 0, len(failures))
	for _, f := range failures {
		out = append(out, map[string]string{
			"subdomain_fqdn":     f.SubdomainFQDN,
			"record_type":        f.RecordType,
			"record_content":     f.RecordContent,
			"provider_record_id": f.ProviderRecordID,
			"error":              f.Error,
		})
	}
	return out
}

func (s *SubdomainService) ReleaseSubdomain(ctx context.Context, sub *model.Subdomain, opts ReleaseOpts) OperationResult {
	if sub == nil {
		return OperationResult{HTTPStatus: http.StatusNotFound, Message: "subdomain not found", MessageKey: apperr.KeySubdomainNotFound}
	}

	items := BuildBatchDeleteItemsForSubdomain(sub)
	deleteResult := s.ops.DeleteRecordsBatch(ctx, items, 3)
	if deleteResult.Async {
		return OperationResult{
			HTTPStatus: http.StatusConflict,
			Message:    "dns bulk delete queued, retry release after job succeeds",
			MessageKey: apperr.KeyCloudflareOperationInProgress,
			Data:       map[string]interface{}{"bulk_job_id": deleteResult.JobID, "bulk_async": true},
		}
	}
	if deleteResult.Failed > 0 {
		return OperationResult{
			HTTPStatus: http.StatusConflict,
			Message:    "some cloudflare dns records failed to delete",
			MessageKey: apperr.KeyCloudflareDeleteFailed,
			Data:       map[string]interface{}{"failed_records": ToCFFailureRecords(deleteResult.Failures)},
		}
	}

	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.DeleteDNSRecordsBySubdomainID(tx, sub.ID); err != nil {
			return err
		}
		if err := s.repo.DeleteSubdomainByID(tx, sub.ID); err != nil {
			return err
		}
		details := map[string]interface{}{
			"fqdn":              sub.FQDN,
			"deleted_dns_count": deleteResult.Succeeded,
		}
		for k, v := range opts.AuditExtra {
			details[k] = v
		}
		return s.auditLog.RecordUserTx(tx, opts.ActorID, opts.AuditAction, "subdomain", sub.ID, details)
	}); err != nil {
		return OperationResult{
			HTTPStatus: http.StatusInternalServerError,
			Message:    "failed to release subdomain",
			MessageKey: apperr.KeyDatabaseError,
			Retryable:  true,
		}
	}

	return OperationResult{
		HTTPStatus: http.StatusOK,
		Message:    "ok",
		Data:       map[string]interface{}{"message": "subdomain released", "deleted_dns_count": deleteResult.Succeeded},
	}
}

var ErrSubdomainNotFound = errors.New("subdomain not found")
