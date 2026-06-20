package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StreamAuditScanTasks  = "audit:scan:tasks"
	GroupAuditScanWorkers = "audit-scan-workers"
)

// RedisStreams 封装 Redis Streams 操作，支持配置 stream 与 consumer group 名。
type RedisStreams struct {
	rdb           *redis.Client
	streamName    string
	consumerGroup string
}

// StreamOption 配置 RedisStreams 的 stream 与 consumer group。
type StreamOption func(*RedisStreams)

// WithStreamName 设置 stream 名称。
func WithStreamName(name string) StreamOption {
	return func(s *RedisStreams) { s.streamName = name }
}

// WithConsumerGroup 设置 consumer group 名称。
func WithConsumerGroup(group string) StreamOption {
	return func(s *RedisStreams) { s.consumerGroup = group }
}

// NewRedisStreams 构造 Redis Streams 封装；调用方应通过 StreamOption 指定 stream/group（当前用于审计扫描队列）。
func NewRedisStreams(rdb *redis.Client, opts ...StreamOption) *RedisStreams {
	s := &RedisStreams{
		rdb:           rdb,
		streamName:    StreamAuditScanTasks,
		consumerGroup: GroupAuditScanWorkers,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// EnsureConsumerGroup 创建流（若需要）与消费者组。
func (s *RedisStreams) EnsureConsumerGroup(ctx context.Context) error {
	if s == nil || s.rdb == nil {
		return errors.New("redis streams: nil client")
	}
	err := s.rdb.XGroupCreateMkStream(ctx, s.streamName, s.consumerGroup, "0").Err()
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "busygroup") {
		return fmt.Errorf("xgroup create: %w", err)
	}
	return nil
}

// AddTask 入队任务（近似修剪流长度）。
func (s *RedisStreams) AddTask(ctx context.Context, fields map[string]interface{}) (string, error) {
	if s == nil || s.rdb == nil {
		return "", errors.New("redis streams: nil client")
	}
	id, err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.streamName,
		MaxLen: 10000,
		Approx: true,
		Values: fields,
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd: %w", err)
	}
	return id, nil
}

// ReadArgs 配置 XREADGROUP。
type ReadArgs struct {
	Consumer   string
	Count      int64
	Block      time.Duration
	StreamsIDs []string
}

// ReadGroup 阻塞等待分配给本消费者的新消息。
func (s *RedisStreams) ReadGroup(ctx context.Context, consumer string, count int64, block time.Duration) ([]redis.XStream, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("redis streams: nil client")
	}
	if consumer == "" {
		return nil, errors.New("redis streams: empty consumer")
	}
	res, err := s.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    s.consumerGroup,
		Consumer: consumer,
		Streams:  []string{s.streamName, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

// Ack 确认消息已处理。
func (s *RedisStreams) Ack(ctx context.Context, streamID string) error {
	if s == nil || s.rdb == nil {
		return errors.New("redis streams: nil client")
	}
	return s.rdb.XAck(ctx, s.streamName, s.consumerGroup, streamID).Err()
}

// AutoClaim 回收空闲超过 minIdle 的 pending 条目。
func (s *RedisStreams) AutoClaim(ctx context.Context, consumer string, minIdle time.Duration, start string, count int64) ([]redis.XMessage, string, error) {
	if s == nil || s.rdb == nil {
		return nil, "", errors.New("redis streams: nil client")
	}
	msgs, next, err := s.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   s.streamName,
		Group:    s.consumerGroup,
		Consumer: consumer,
		MinIdle:  minIdle,
		Start:    start,
		Count:    count,
	}).Result()
	if err != nil {
		return nil, "", err
	}
	return msgs, next, nil
}

// StreamInfo 返回 XINFO STREAM 数据供积压/年龄估算。
func (s *RedisStreams) StreamInfo(ctx context.Context) (*redis.XInfoStream, error) {
	if s == nil || s.rdb == nil {
		return nil, errors.New("redis streams: nil client")
	}
	info, err := s.rdb.XInfoStream(ctx, s.streamName).Result()
	if err != nil {
		if strings.Contains(err.Error(), "no such key") || strings.Contains(err.Error(), "ERR") {
			return &redis.XInfoStream{}, nil
		}
		return nil, err
	}
	return info, nil
}

// PendingCount 返回消费组的 pending 条目数（聚合）。
func (s *RedisStreams) PendingCount(ctx context.Context) (int64, error) {
	if s == nil || s.rdb == nil {
		return 0, errors.New("redis streams: nil client")
	}
	pending, err := s.rdb.XPending(ctx, s.streamName, s.consumerGroup).Result()
	if err != nil {
		if strings.Contains(err.Error(), "no such key") || strings.Contains(strings.ToLower(err.Error()), "nogroup") {
			return 0, nil
		}
		return 0, err
	}
	return pending.Count, nil
}
