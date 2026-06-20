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
	subID := uintField(values["subdomain_id"])

	if subID == 0 || fqdn == "" {
		slog.Warn("audit scan worker: drop invalid task", "id", id, "subdomain_id", values["subdomain_id"], "fqdn", fqdn)
		_ = w.queue.Ack(ctx, id)
		return
	}

	w.svc.ScanSubdomain(ctx, model.AuditScanTarget{
		ID:     subID,
		FQDN:   fqdn,
		Source: stringField(values["source"]),
		RuleID: uintField(values["rule_id"]),
	})

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

// uintField 解析队列字段中的无符号整数。进程内队列保留 Go 原生数值类型；Redis Stream 返回字符串。
func uintField(v interface{}) uint {
	switch t := v.(type) {
	case uint:
		return t
	case uint8:
		return uint(t)
	case uint16:
		return uint(t)
	case uint32:
		return uint(t)
	case uint64:
		return uint(t)
	case int:
		if t > 0 {
			return uint(t)
		}
	case int64:
		if t > 0 {
			return uint(t)
		}
	case string:
		n, err := strconv.ParseUint(t, 10, 64)
		if err == nil {
			return uint(n)
		}
	case []byte:
		n, err := strconv.ParseUint(string(t), 10, 64)
		if err == nil {
			return uint(n)
		}
	}
	return 0
}

func AuditConsumerName(workerIndex int) string {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return host + "-audit-" + strconv.Itoa(os.Getpid()) + "-" + strconv.Itoa(workerIndex)
}
