package model

import (
	"encoding/json"
	"time"
)

const (
	CloudflareTaskActionCreateDNSRecord = "create_dns_record"
	CloudflareTaskActionUpdateDNSRecord = "update_dns_record"
	CloudflareTaskActionDeleteDNSRecord = "delete_dns_record"

	CloudflareTaskStatusPending   = "pending"
	CloudflareTaskStatusRunning   = "running"
	CloudflareTaskStatusRetry     = "retry"
	CloudflareTaskStatusSucceeded = "succeeded"
	CloudflareTaskStatusDead      = "dead"
)

type CloudflareTaskPayload struct {
	CloudflareAccountID uint   `json:"cloudflare_account_id"`
	ZoneID              string `json:"zone_id"`
	RecordID            string `json:"record_id,omitempty"`
	RecordType          string `json:"record_type,omitempty"`
	Name                string `json:"name,omitempty"`
	Content             string `json:"content,omitempty"`
	TTL                 int    `json:"ttl,omitempty"`
	Proxied             bool   `json:"proxied,omitempty"`
}

type CloudflareTask struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	ResourceType   string          `json:"resource_type" gorm:"type:varchar(32);not null;index"`
	ResourceID     uint            `json:"resource_id" gorm:"not null;index"`
	Action         string          `json:"action" gorm:"type:varchar(64);not null;index"`
	Payload        json.RawMessage `json:"payload" gorm:"type:jsonb;not null"`
	Status         string          `json:"status" gorm:"type:varchar(24);not null;default:pending;index"`
	Attempts       int             `json:"attempts" gorm:"not null;default:0"`
	MaxAttempts    int             `json:"max_attempts" gorm:"not null;default:8"`
	NextRetryAt    time.Time       `json:"next_retry_at" gorm:"not null;index"`
	LastError      string          `json:"last_error,omitempty" gorm:"type:text"`
	IdempotencyKey string          `json:"idempotency_key" gorm:"type:varchar(191);not null;uniqueIndex"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
