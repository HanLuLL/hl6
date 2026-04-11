package model

import (
	"encoding/json"
	"time"
)

const (
	DNSOperationRequestStatusRunning   = "running"
	DNSOperationRequestStatusSucceeded = "succeeded"
	DNSOperationRequestStatusFailed    = "failed"
)

type DNSOperationRequest struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	Scope          string          `json:"scope" gorm:"type:varchar(191);not null;uniqueIndex:idx_dns_op_scope_key"`
	IdempotencyKey string          `json:"idempotency_key" gorm:"type:varchar(191);not null;uniqueIndex:idx_dns_op_scope_key"`
	Status         string          `json:"status" gorm:"type:varchar(24);not null;default:running;index"`
	HTTPStatus     int             `json:"http_status" gorm:"not null;default:200"`
	Message        string          `json:"message" gorm:"type:text"`
	MessageKey     string          `json:"message_key" gorm:"type:varchar(191)"`
	Retryable      bool            `json:"retryable" gorm:"not null;default:false"`
	ResponseData   json.RawMessage `json:"response_data" gorm:"type:jsonb"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (DNSOperationRequest) TableName() string {
	return "dns_operation_requests"
}

type DNSOperationEvent struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	RequestID     *uint           `json:"request_id,omitempty" gorm:"index"`
	Scope         string          `json:"scope" gorm:"type:varchar(191);not null;index"`
	Step          string          `json:"step" gorm:"type:varchar(64);not null;index"`
	Success       bool            `json:"success" gorm:"not null;default:false"`
	Message       string          `json:"message" gorm:"type:text"`
	Detail        json.RawMessage `json:"detail,omitempty" gorm:"type:jsonb"`
	Provider      string          `json:"provider,omitempty" gorm:"type:varchar(32);index"`
	ProviderZone  string          `json:"provider_zone,omitempty" gorm:"type:varchar(191);index"`
	RecordID      uint            `json:"record_id,omitempty" gorm:"index"`
	ProviderRecID string          `json:"provider_record_id,omitempty" gorm:"type:varchar(191)"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (DNSOperationEvent) TableName() string {
	return "dns_operation_events"
}
