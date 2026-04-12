package model

import (
	"encoding/json"
	"time"
)

// Migration task status values
const (
	MigrationTaskStatusPending       = "pending"
	MigrationTaskStatusRunning       = "running"
	MigrationTaskStatusSucceeded     = "succeeded"
	MigrationTaskStatusPartialFailed = "partial_failed"
	MigrationTaskStatusFailed        = "failed"
	MigrationTaskStatusCancelled     = "cancelled"
)

// Migration item status values
const (
	MigrationItemStatusPending   = "pending"
	MigrationItemStatusRunning   = "running"
	MigrationItemStatusSucceeded = "succeeded"
	MigrationItemStatusFailed    = "failed"
	MigrationItemStatusSkipped   = "skipped"
)

// Domain migration state values (stored on Domain)
const (
	DomainMigrationStateIdle          = "idle"
	DomainMigrationStateRunning       = "running"
	DomainMigrationStatePartialFailed = "partial_failed"
	DomainMigrationStateFailed        = "failed"
	DomainMigrationStateQueued        = "queued"
)

// DomainDNSMigrationTask represents a DNS migration task from one provider to another.
type DomainDNSMigrationTask struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	DomainID          uint       `json:"domain_id" gorm:"not null;index:idx_migration_domain_status;index:idx_migration_domain_seq"`
	Status            string     `json:"status" gorm:"type:varchar(24);not null;default:pending;index;index:idx_migration_domain_status"`
	QueueSeq          int64      `json:"queue_seq" gorm:"not null;default:0;uniqueIndex:idx_migration_domain_seq"`
	TriggeredBy       uint       `json:"triggered_by" gorm:"not null;index"`
	SourceProvider    string     `json:"source_provider" gorm:"type:varchar(32);not null"`
	SourceAccountID   uint       `json:"source_account_id" gorm:"not null"`
	SourceZoneID      string     `json:"source_zone_id" gorm:"type:varchar(191);not null"`
	TargetProvider    string     `json:"target_provider" gorm:"type:varchar(32);not null"`
	TargetAccountID   uint       `json:"target_account_id" gorm:"not null"`
	TargetZoneID      string     `json:"target_zone_id" gorm:"type:varchar(191);not null"`
	Reason            string     `json:"reason" gorm:"type:text"`
	TotalItems        int        `json:"total_items" gorm:"not null;default:0"`
	SucceededItems    int        `json:"succeeded_items" gorm:"not null;default:0"`
	FailedItems       int        `json:"failed_items" gorm:"not null;default:0"`
	RetriedItems      int        `json:"retried_items" gorm:"not null;default:0"`
	LastErrorCategory string     `json:"last_error_category" gorm:"type:varchar(32)"`
	LastErrorMessage  string     `json:"last_error_message" gorm:"type:text"`
	StartedAt         *time.Time `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	// Associations (preload only)
	Domain  *Domain                   `json:"domain,omitempty" gorm:"foreignKey:DomainID"`
	Items   []DomainDNSMigrationItem  `json:"items,omitempty" gorm:"foreignKey:TaskID"`
}

func (DomainDNSMigrationTask) TableName() string {
	return "domain_dns_migration_tasks"
}

// IsTerminal returns true if the task is in a terminal state.
func (t *DomainDNSMigrationTask) IsTerminal() bool {
	switch t.Status {
	case MigrationTaskStatusSucceeded, MigrationTaskStatusPartialFailed,
		MigrationTaskStatusFailed, MigrationTaskStatusCancelled:
		return true
	}
	return false
}

// DomainDNSMigrationItem represents a single DNS record migration within a task.
type DomainDNSMigrationItem struct {
	ID                     uint            `json:"id" gorm:"primaryKey"`
	TaskID                 uint            `json:"task_id" gorm:"not null;index"`
	DNSRecordID            uint            `json:"dns_record_id" gorm:"not null;index"`
	RecordType             string          `json:"record_type" gorm:"type:varchar(16);not null"`
	Name                   string          `json:"name" gorm:"type:varchar(255);not null"`
	Content                string          `json:"content" gorm:"type:text;not null"`
	TTL                    int             `json:"ttl" gorm:"not null;default:1"`
	Proxied                bool            `json:"proxied" gorm:"not null;default:false"`
	SourceProviderRecordID string          `json:"source_provider_record_id" gorm:"type:varchar(191)"`
	TargetProviderRecordID string          `json:"target_provider_record_id" gorm:"type:varchar(191)"`
	Status                 string          `json:"status" gorm:"type:varchar(24);not null;default:pending;index"`
	Attempts               int             `json:"attempts" gorm:"not null;default:0"`
	LastErrorCategory      string          `json:"last_error_category" gorm:"type:varchar(32)"`
	LastErrorMessage       string          `json:"last_error_message" gorm:"type:text"`
	ConflictOverwritten    bool            `json:"conflict_overwritten" gorm:"not null;default:false"`
	OverwriteBefore        json.RawMessage `json:"overwrite_before,omitempty" gorm:"type:jsonb"`
	OverwriteAfter         json.RawMessage `json:"overwrite_after,omitempty" gorm:"type:jsonb"`
	FinishedAt             *time.Time      `json:"finished_at"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

func (DomainDNSMigrationItem) TableName() string {
	return "domain_dns_migration_items"
}
