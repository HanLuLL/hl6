package worker

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"hl6-server/internal/model"
	"hl6-server/internal/service"
	"hl6-server/pkg/queue"
)

const (
	auditReadBlock         = 10 * time.Second
	auditAutoClaimInterval = 30 * time.Second
	auditAutoClaimMinIdle  = 2 * time.Minute
	auditAutoClaimStart    = "0-0"
)

// AuditScanWorker 消费审计扫描任务队列。
type AuditScanWorker struct {
	queue queue.TaskQueue
	svc   *service.AuditService
	name  string
}

func NewAuditScanWorker(q queue.TaskQueue, svc *service.AuditService, consumerName string) *AuditScanWorker {
	return &AuditScanWorker{
		queue: q,
		svc:   svc,
		name:  consumerName,
	}
}

func (w *AuditScanWorker) Run(ctx context.Context) {
	if w == nil || w.queue == nil || w.svc == nil || w.name == "" {
		return
	}

	go w.autoClaimLoop(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgs, err := w.queue.ReadTasks(ctx, w.name, 4, auditReadBlock)
		if err != nil {
			slog.Warn("audit scan worker read", "err", err)
			time.Sleep(time.Second)
			continue
		}
		for _, msg := range msgs {
			w.handleMessage(ctx, msg.ID, msg.Values)
		}
	}
}

func (w *AuditScanWorker) autoClaimLoop(ctx context.Context) {
	t := time.NewTicker(auditAutoClaimInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			start := auditAutoClaimStart
			for {
				msgs, next, err := w.queue.AutoClaim(ctx, w.name, auditAutoClaimMinIdle, start, 16)
				if err != nil {
					slog.Warn("audit scan worker autoclaim", "err", err)
					break
				}
				for _, msg := range msgs {
					w.handleMessage(ctx, msg.ID, msg.Values)
				}
				if next == "0-0" || len(msgs) == 0 {
					break
				}
				start = next
			}
		}
	}
}

func (w *AuditScanWorker) handleMessage(ctx context.Context, id string, values map[string]interface{}) {
	fqdn := stringField(values["fqdn"])
	subIDStr := stringField(values["subdomain_id"])
	subID64, _ := strconv.ParseUint(subIDStr, 10, 64)
	subID := uint(subID64)

	if subID == 0 || fqdn == "" {
		_ = w.queue.Ack(ctx, id)
		return
	}

	w.svc.ScanSubdomain(ctx, model.AuditScanTarget{ID: subID, FQDN: fqdn})

	if err := w.queue.Ack(ctx, id); err != nil {
		slog.Warn("audit scan ack failed", "id", id, "err", err)
	}
}

func stringField(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}

func AuditConsumerName(workerIndex int) string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return host + "-audit-" + strconv.Itoa(os.Getpid()) + "-" + strconv.Itoa(workerIndex)
}
