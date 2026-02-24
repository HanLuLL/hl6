package handler

import (
	"sync"
)

type SSEEvent struct {
	Event string
	Data  string
}

type SSEBroker struct {
	mu      sync.RWMutex
	clients map[uint]map[chan SSEEvent]struct{}
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[uint]map[chan SSEEvent]struct{}),
	}
}

func (b *SSEBroker) Subscribe(userID uint) chan SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan SSEEvent, 16)
	if b.clients[userID] == nil {
		b.clients[userID] = make(map[chan SSEEvent]struct{})
	}
	b.clients[userID][ch] = struct{}{}
	return ch
}

func (b *SSEBroker) Unsubscribe(userID uint, ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if channels, ok := b.clients[userID]; ok {
		delete(channels, ch)
		if len(channels) == 0 {
			delete(b.clients, userID)
		}
	}
	close(ch)
}

func (b *SSEBroker) SendToUsers(userIDs []uint, event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, uid := range userIDs {
		if channels, ok := b.clients[uid]; ok {
			for ch := range channels {
				select {
				case ch <- event:
				default:
				}
			}
		}
	}
}

func (b *SSEBroker) SendToAll(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, channels := range b.clients {
		for ch := range channels {
			select {
			case ch <- event:
			default:
			}
		}
	}
}
