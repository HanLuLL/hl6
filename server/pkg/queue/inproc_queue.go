package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultInProcQueueCapacity = 4096

// InProcQueue 进程内 buffered channel 队列（无 Redis 时降级）。
type InProcQueue struct {
	ch       chan TaskMessage
	capacity int
	once     sync.Once
}

func NewInProcQueue(capacity int) *InProcQueue {
	if capacity <= 0 {
		capacity = defaultInProcQueueCapacity
	}
	return &InProcQueue{
		ch:       make(chan TaskMessage, capacity),
		capacity: capacity,
	}
}

func (q *InProcQueue) EnsureReady(ctx context.Context) error {
	return nil
}

func (q *InProcQueue) AddTask(ctx context.Context, fields map[string]interface{}) (string, error) {
	if q == nil {
		return "", fmt.Errorf("inproc queue: nil")
	}
	id := uuid.NewString()
	msg := TaskMessage{ID: id, Values: fields}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case q.ch <- msg:
		return id, nil
	default:
		return "", fmt.Errorf("inproc queue: full (capacity %d)", q.capacity)
	}
}

func (q *InProcQueue) ReadTasks(ctx context.Context, consumer string, count int64, block time.Duration) ([]TaskMessage, error) {
	if q == nil {
		return nil, fmt.Errorf("inproc queue: nil")
	}
	_ = consumer
	if count <= 0 {
		count = 1
	}
	if block <= 0 {
		block = 10 * time.Second
	}

	var out []TaskMessage
	deadline := time.Now().Add(block)
	for int64(len(out)) < count {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case msg := <-q.ch:
			out = append(out, msg)
		case <-time.After(remaining):
			return out, nil
		}
	}
	return out, nil
}

func (q *InProcQueue) Ack(ctx context.Context, id string) error {
	_ = ctx
	_ = id
	return nil
}

func (q *InProcQueue) AutoClaim(ctx context.Context, consumer string, minIdle time.Duration, start string, count int64) ([]TaskMessage, string, error) {
	_ = ctx
	_ = consumer
	_ = minIdle
	_ = start
	_ = count
	return nil, "0-0", nil
}
