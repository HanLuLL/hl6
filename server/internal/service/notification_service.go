package service

import (
	"encoding/json"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"
)

// EventPublisher 抽象面向用户的推送投递（SSE 等）。
type EventPublisher interface {
	Publish(userIDs []uint, event string, data any)
}

// NotificationService 发送应用内通知并可选 SSE 扇出。
type NotificationService struct {
	repo *repository.Repository
	pub  EventPublisher
}

func NewNotificationService(repo *repository.Repository, pub EventPublisher) *NotificationService {
	return &NotificationService{repo: repo, pub: pub}
}

func (s *NotificationService) NotifyUsers(
	userIDs []uint,
	createdBy uint,
	nType, title, content string,
	messageKey string,
	messageArgs json.RawMessage,
) (*model.Notification, error) {
	targetIDs, err := json.Marshal(userIDs)
	if err != nil {
		return nil, err
	}
	n := &model.Notification{
		Title:       title,
		Content:     content,
		MessageKey:  messageKey,
		MessageArgs: messageArgs,
		Type:        nType,
		TargetType:  "users",
		TargetIDs:   targetIDs,
		CreatedBy:   createdBy,
	}
	if err := s.repo.CreateNotificationWithImages(n); err != nil {
		return nil, err
	}
	if s.pub != nil && len(userIDs) > 0 {
		s.pub.Publish(userIDs, "new_notification", map[string]any{"id": n.ID})
	}
	return n, nil
}
