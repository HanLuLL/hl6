package handler

import (
	"encoding/json"
	"log/slog"
)

// SSEEventPublisher 将 SSEBroker 适配为 service.EventPublisher。
type SSEEventPublisher struct {
	Broker *SSEBroker
}

func (p *SSEEventPublisher) Publish(userIDs []uint, event string, data any) {
	if p == nil || p.Broker == nil || len(userIDs) == 0 || event == "" {
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		slog.Debug("sse publish marshal failed", "event", event, "err", err)
		return
	}
	p.Broker.SendToUsers(userIDs, SSEEvent{Event: event, Data: string(raw)})
}
