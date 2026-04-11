package model

import "time"

const (
	DNSBulkJobStatusPending   = "pending"
	DNSBulkJobStatusRunning   = "running"
	DNSBulkJobStatusSucceeded = "succeeded"
	DNSBulkJobStatusFailed    = "failed"

	DNSBulkJobItemStatusPending   = "pending"
	DNSBulkJobItemStatusRunning   = "running"
	DNSBulkJobItemStatusSucceeded = "succeeded"
	DNSBulkJobItemStatusFailed    = "failed"
)

type DNSBulkJob struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	Scope          string     `json:"scope" gorm:"type:varchar(191);not null;index"`
	Status         string     `json:"status" gorm:"type:varchar(24);not null;default:pending;index"`
	TotalItems     int        `json:"total_items" gorm:"not null;default:0"`
	SucceededItems int        `json:"succeeded_items" gorm:"not null;default:0"`
	FailedItems    int        `json:"failed_items" gorm:"not null;default:0"`
	MaxAttempts    int        `json:"max_attempts" gorm:"not null;default:3"`
	Message        string     `json:"message" gorm:"type:text"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (DNSBulkJob) TableName() string {
	return "dns_bulk_jobs"
}

type DNSBulkJobItem struct {
	ID                uint       `json:"id" gorm:"primaryKey"`
	JobID             uint       `json:"job_id" gorm:"index;not null"`
	RecordID          uint       `json:"record_id" gorm:"index"`
	SubdomainFQDN     string     `json:"subdomain_fqdn" gorm:"type:varchar(255)"`
	Provider          string     `json:"provider" gorm:"type:varchar(32);index"`
	ProviderAccountID uint       `json:"provider_account_id" gorm:"index"`
	ZoneID            string     `json:"zone_id" gorm:"type:varchar(191);index"`
	ProviderRecordID  string     `json:"provider_record_id" gorm:"type:varchar(191)"`
	RecordType        string     `json:"record_type" gorm:"type:varchar(16);index"`
	Name              string     `json:"name" gorm:"type:varchar(255)"`
	Content           string     `json:"content" gorm:"type:text"`
	TTL               int        `json:"ttl"`
	Proxied           bool       `json:"proxied"`
	Attempts          int        `json:"attempts" gorm:"not null;default:0"`
	Status            string     `json:"status" gorm:"type:varchar(24);not null;default:pending;index"`
	LastError         string     `json:"last_error" gorm:"type:text"`
	FinishedAt        *time.Time `json:"finished_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (DNSBulkJobItem) TableName() string {
	return "dns_bulk_job_items"
}
