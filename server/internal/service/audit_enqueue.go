package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"hl6-server/pkg/queue"
)

// EnqueueOpts 扫描入队选项。
type EnqueueOpts struct {
	BypassDedup bool
	RuleID      uint
}

// AuditDedup 入队去重接口。
type AuditDedup interface {
	TryEnqueue(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

type redisAuditDedup struct {
	rdb *redis.Client
}

func NewRedisAuditDedup(rdb *redis.Client) AuditDedup {
	return &redisAuditDedup{rdb: rdb}
}

func (d *redisAuditDedup) TryEnqueue(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if d == nil || d.rdb == nil {
		return true, nil
	}
	return d.rdb.SetNX(ctx, key, "1", ttl).Result()
}

type inprocAuditDedup struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func NewInprocAuditDedup() AuditDedup {
	return &inprocAuditDedup{items: make(map[string]time.Time)}
}

func (d *inprocAuditDedup) TryEnqueue(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	_ = ctx
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, exp := range d.items {
		if now.After(exp) {
			delete(d.items, k)
		}
	}
	if exp, ok := d.items[key]; ok && now.Before(exp) {
		return false, nil
	}
	d.items[key] = now.Add(ttl)
	return true, nil
}

// AuditEnqueueService 将子域名扫描任务投递到队列。
type AuditEnqueueService struct {
	queue queue.TaskQueue
	dedup AuditDedup
}

func NewAuditEnqueueService(q queue.TaskQueue, dedup AuditDedup) *AuditEnqueueService {
	return &AuditEnqueueService{queue: q, dedup: dedup}
}

func (s *AuditEnqueueService) EnqueueScan(ctx context.Context, subdomainID uint, fqdn, source string, opts EnqueueOpts) error {
	if s == nil || s.queue == nil {
		return fmt.Errorf("audit enqueue: not configured")
	}
	if subdomainID == 0 || fqdn == "" {
		return fmt.Errorf("audit enqueue: invalid target")
	}
	if source == "" {
		source = "manual"
	}

	if !opts.BypassDedup && s.dedup != nil {
		dedupKey := fmt.Sprintf("hl6:audit:dedup:sub:%d", subdomainID)
		ok, err := s.dedup.TryEnqueue(ctx, dedupKey, 5*time.Minute)
		if err != nil {
			slog.Warn("audit enqueue: dedup check failed, proceeding", "subdomain_id", subdomainID, "err", err)
		} else if !ok {
			return nil
		}
	}

	taskID := uuid.NewString()
	payload := map[string]interface{}{
		"task_id":      taskID,
		"subdomain_id": subdomainID,
		"fqdn":         fqdn,
		"source":       source,
	}
	if opts.RuleID > 0 {
		payload["rule_id"] = opts.RuleID
	}
	_, err := s.queue.AddTask(ctx, payload)
	return err
}

func (s *AuditEnqueueService) EnqueueBulk(ctx context.Context, targets []struct {
	ID   uint
	FQDN string
}, source string) (int, error) {
	queued := 0
	for _, t := range targets {
		if err := s.EnqueueScan(ctx, t.ID, t.FQDN, source, EnqueueOpts{BypassDedup: true}); err != nil {
			return queued, err
		}
		queued++
	}
	return queued, nil
}
