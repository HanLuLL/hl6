package worker

import (
	"context"
	"log/slog"
	"time"

	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

const auditExemptionPollInterval = time.Minute
const auditExemptionBatchSize = 50

// AuditExemptionWorker 轮询到期的豁免记录并触发延迟重扫。
type AuditExemptionWorker struct {
	repo    *repository.Repository
	enqueue *service.AuditEnqueueService
}

func NewAuditExemptionWorker(repo *repository.Repository, enqueue *service.AuditEnqueueService) *AuditExemptionWorker {
	return &AuditExemptionWorker{repo: repo, enqueue: enqueue}
}

func (w *AuditExemptionWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(auditExemptionPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.processDue(ctx)
		}
	}
}

func (w *AuditExemptionWorker) processDue(ctx context.Context) {
	if w == nil || w.repo == nil || w.enqueue == nil {
		return
	}
	claimed, err := w.repo.ClaimDueExemptions(auditExemptionBatchSize)
	if err != nil {
		slog.Error("audit exemption worker: claim due failed", "err", err)
		return
	}
	for _, item := range claimed {
		fqdn, fqdnErr := w.repo.FindSubdomainFQDNByID(item.SubdomainID)
		if fqdnErr != nil || fqdn == "" {
			slog.Warn("audit exemption worker: subdomain not found", "subdomain_id", item.SubdomainID, "err", fqdnErr)
			continue
		}
		if err := w.enqueue.EnqueueScan(ctx, item.SubdomainID, fqdn, "exemption_recheck", service.EnqueueOpts{BypassDedup: true}); err != nil {
			slog.Error("audit exemption worker: enqueue failed", "fqdn", fqdn, "err", err)
		}
	}
}
