package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"

	"gorm.io/gorm"
)

const cloudflareTaskResourceDNSRecord = "dns_record"

func enqueueCloudflareTask(repo *repository.Repository, tx *gorm.DB, resourceType string, resourceID uint, action string, payload model.CloudflareTaskPayload, idempotencyKey string) (*model.CloudflareTask, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := &model.CloudflareTask{
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Action:         action,
		Payload:        rawPayload,
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
	}
	if task.IdempotencyKey == "" {
		task.IdempotencyKey = fmt.Sprintf("%s:%d:%s", action, resourceID, payload.RecordID)
	}
	if err := repo.EnqueueCloudflareTask(tx, task); err != nil {
		return nil, err
	}
	return task, nil
}
