package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"hl6-server/internal/repository"
	"hl6-server/pkg/queue"
)

const auditSchedulerBatchSize = 100

// ScheduleDedup 调度周期内去重。
type ScheduleDedup interface {
	TrySchedule(ctx context.Context, dedupKey string, ttl time.Duration) (bool, error)
}

type redisScheduleDedup struct {
	rdb *redis.Client
}

func NewRedisScheduleDedup(rdb *redis.Client) ScheduleDedup {
	return &redisScheduleDedup{rdb: rdb}
}

func (d *redisScheduleDedup) TrySchedule(ctx context.Context, dedupKey string, ttl time.Duration) (bool, error) {
	if d == nil || d.rdb == nil {
		return true, nil
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return d.rdb.SetNX(ctx, "hl6:audit:dedup:"+dedupKey, "1", ttl).Result()
}

type inprocScheduleDedup struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func NewInprocScheduleDedup() ScheduleDedup {
	return &inprocScheduleDedup{items: make(map[string]time.Time)}
}

func (d *inprocScheduleDedup) TrySchedule(ctx context.Context, dedupKey string, ttl time.Duration) (bool, error) {
	_ = ctx
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, exp := range d.items {
		if now.After(exp) {
			delete(d.items, k)
		}
	}
	if exp, ok := d.items[dedupKey]; ok && now.Before(exp) {
		return false, nil
	}
	d.items[dedupKey] = now.Add(ttl)
	return true, nil
}

// AuditScheduler 定期选出待巡检的子域名并投递到队列。
type AuditScheduler struct {
	db       *gorm.DB
	queue    queue.TaskQueue
	dedup    ScheduleDedup
	interval time.Duration
}

func NewAuditScheduler(db *gorm.DB, q queue.TaskQueue, dedup ScheduleDedup, interval time.Duration) *AuditScheduler {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &AuditScheduler{
		db:       db,
		queue:    q,
		dedup:    dedup,
		interval: interval,
	}
}

func (w *AuditScheduler) Run(ctx context.Context) error {
	if w == nil || w.db == nil || w.queue == nil {
		return nil
	}

	if err := w.enqueueAll(ctx); err != nil {
		slog.Warn("audit scheduler initial enqueue failed", "err", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.enqueueAll(ctx); err != nil {
				slog.Warn("audit scheduler enqueue failed", "err", err)
			}
		}
	}
}

func (w *AuditScheduler) enqueueAll(ctx context.Context) error {
	repo := repository.New(w.db)
	total, err := repo.CountActiveScanTargets()
	if err != nil {
		return err
	}
	if total == 0 {
		return nil
	}

	slog.Info("audit scheduler: enqueuing scan targets", "total", total)

	offset := 0
	for {
		targets, listErr := repo.ListActiveScanTargets(offset, auditSchedulerBatchSize)
		if listErr != nil {
			return listErr
		}
		if len(targets) == 0 {
			break
		}

		for _, target := range targets {
			dedupKey := fmt.Sprintf("sub:%d", target.ID)
			ok, dedupErr := w.dedup.TrySchedule(ctx, dedupKey, w.interval)
			if dedupErr != nil {
				slog.Error("audit scheduler: dedup check failed, proceeding without dedup",
					"subdomain_id", target.ID, "err", dedupErr)
				ok = true
			}
			if !ok {
				continue
			}

			taskID := uuid.NewString()
			if _, addErr := w.queue.AddTask(ctx, map[string]interface{}{
				"task_id":      taskID,
				"subdomain_id": target.ID,
				"fqdn":         target.FQDN,
				"source":       "scheduler",
			}); addErr != nil {
				slog.Warn("audit scheduler: add task failed", "subdomain_id", target.ID, "err", addErr)
				continue
			}
		}

		offset += len(targets)
		if len(targets) < auditSchedulerBatchSize {
			break
		}
	}

	return nil
}
