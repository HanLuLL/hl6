package queue

import (
	"context"
	"time"
)

// TaskMessage 队列任务消息。
type TaskMessage struct {
	ID     string
	Values map[string]interface{}
}

// TaskQueue 审计扫描任务队列抽象（Redis Stream 或进程内 channel）。
type TaskQueue interface {
	EnsureReady(ctx context.Context) error
	AddTask(ctx context.Context, fields map[string]interface{}) (string, error)
	ReadTasks(ctx context.Context, consumer string, count int64, block time.Duration) ([]TaskMessage, error)
	Ack(ctx context.Context, id string) error
	AutoClaim(ctx context.Context, consumer string, minIdle time.Duration, start string, count int64) ([]TaskMessage, string, error)
}

// redisTaskQueue 将 RedisStreams 适配为 TaskQueue。
type redisTaskQueue struct {
	streams *RedisStreams
}

func NewRedisTaskQueue(streams *RedisStreams) TaskQueue {
	return &redisTaskQueue{streams: streams}
}

func (q *redisTaskQueue) EnsureReady(ctx context.Context) error {
	return q.streams.EnsureConsumerGroup(ctx)
}

func (q *redisTaskQueue) AddTask(ctx context.Context, fields map[string]interface{}) (string, error) {
	return q.streams.AddTask(ctx, fields)
}

func (q *redisTaskQueue) ReadTasks(ctx context.Context, consumer string, count int64, block time.Duration) ([]TaskMessage, error) {
	xstreams, err := q.streams.ReadGroup(ctx, consumer, count, block)
	if err != nil || len(xstreams) == 0 {
		return nil, err
	}
	var out []TaskMessage
	for _, xs := range xstreams {
		for _, msg := range xs.Messages {
			out = append(out, TaskMessage{ID: msg.ID, Values: msg.Values})
		}
	}
	return out, nil
}

func (q *redisTaskQueue) Ack(ctx context.Context, id string) error {
	return q.streams.Ack(ctx, id)
}

func (q *redisTaskQueue) AutoClaim(ctx context.Context, consumer string, minIdle time.Duration, start string, count int64) ([]TaskMessage, string, error) {
	msgs, next, err := q.streams.AutoClaim(ctx, consumer, minIdle, start, count)
	if err != nil {
		return nil, "", err
	}
	out := make([]TaskMessage, 0, len(msgs))
	for _, msg := range msgs {
		out = append(out, TaskMessage{ID: msg.ID, Values: msg.Values})
	}
	return out, next, nil
}
