package model

import (
	"encoding/json"
	"time"
)

const (
	AuthTokenPurposeRegistrationVerify = "registration_verify"
	AuthTokenPurposeAccountActivation  = "account_activation"
	AuthTokenPurposePasswordReset      = "password_reset"
	AuthTokenPurposeRestoreChallenge   = "restore_challenge"
)

const (
	AuthSecurityOutcomeSuccess = "success"
	AuthSecurityOutcomeFailure = "failure"
)

const (
	DatabaseBackupStatusReady   = "ready"
	DatabaseBackupStatusExpired = "expired"
	DatabaseBackupStatusFailed  = "failed"
)

const (
	DatabaseRestoreStatusPending   = "pending"
	DatabaseRestoreStatusRunning   = "running"
	DatabaseRestoreStatusSucceeded = "succeeded"
	DatabaseRestoreStatusFailed    = "failed"
)

// UserCredential owns the local identity and session state while User remains
// the owner of profile and all existing business records.
type UserCredential struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`
	UserID               uint       `json:"user_id" gorm:"uniqueIndex;not null"`
	EmailNormalized      string     `json:"email_normalized" gorm:"uniqueIndex;size:320;not null"`
	PasswordHash         string     `json:"-" gorm:"type:text;not null;default:''"`
	PasswordHashVersion  string     `json:"password_hash_version" gorm:"size:32;not null;default:''"`
	EmailVerifiedAt      *time.Time `json:"email_verified_at"`
	PasswordSetAt        *time.Time `json:"password_set_at"`
	SessionVersion       uint       `json:"-" gorm:"not null;default:1"`
	ActivationRequiredAt *time.Time `json:"activation_required_at"`
	User                 *User      `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// AuthToken stores only a SHA-256 token hash. Raw email-link tokens never
// enter persistent storage or logs.
type AuthToken struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Purpose         string     `json:"purpose" gorm:"size:48;not null;index"`
	UserID          *uint      `json:"user_id" gorm:"index"`
	EmailNormalized string     `json:"email_normalized" gorm:"size:320;not null;index"`
	TokenHash       string     `json:"-" gorm:"uniqueIndex;size:64;not null"`
	Payload         json.RawMessage `json:"-" gorm:"type:jsonb"`
	ExpiresAt       time.Time  `json:"expires_at" gorm:"not null;index"`
	ConsumedAt      *time.Time `json:"consumed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AuthSecurityEvent is an append-only, privacy-preserving authentication audit
// record. IP addresses are represented by a keyed hash rather than raw values.
type AuthSecurityEvent struct {
	ID              uint            `json:"id" gorm:"primaryKey"`
	UserID          *uint           `json:"user_id" gorm:"index"`
	Action          string          `json:"action" gorm:"size:64;not null;index"`
	Outcome         string          `json:"outcome" gorm:"size:16;not null;index"`
	EmailNormalized string          `json:"-" gorm:"size:320;not null;default:''"`
	IPHash          string          `json:"-" gorm:"size:64;not null;default:''"`
	UserAgent       string          `json:"user_agent" gorm:"size:512;not null;default:''"`
	Details         json.RawMessage `json:"details,omitempty" gorm:"type:jsonb"`
	CreatedAt       time.Time       `json:"created_at"`
}

// DatabaseBackup records a verified server-generated PostgreSQL export. It
// intentionally stores metadata only; archive locations are server-controlled.
type DatabaseBackup struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	CreatedByUserID uint      `json:"created_by_user_id" gorm:"not null;index"`
	Filename        string    `json:"filename" gorm:"size:255;not null"`
	ChecksumSHA256  string    `json:"checksum_sha256" gorm:"size:64;not null;index"`
	DatabaseVersion string    `json:"database_version" gorm:"size:64;not null;default:''"`
	SchemaRevision  string    `json:"schema_revision" gorm:"size:64;not null;default:''"`
	StoragePath     string    `json:"-" gorm:"type:text;not null"`
	Status          string    `json:"status" gorm:"size:16;not null;default:ready;index"`
	ExpiresAt       *time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// DatabaseRestoreJob captures the operator, immutable input checksum, and
// outcome of a guarded overwrite restore.
type DatabaseRestoreJob struct {
	ID                 uint       `json:"id" gorm:"primaryKey"`
	CreatedByUserID    uint       `json:"created_by_user_id" gorm:"not null;index"`
	ChallengeHash      string     `json:"-" gorm:"size:64;not null;index"`
	InputChecksumSHA256 string    `json:"input_checksum_sha256" gorm:"size:64;not null;default:''"`
	PreRestoreBackupID *uint      `json:"pre_restore_backup_id" gorm:"index"`
	Status             string     `json:"status" gorm:"size:16;not null;default:pending;index"`
	ValidationResult   string     `json:"validation_result" gorm:"type:text;not null;default:''"`
	FailureDetail      string     `json:"failure_detail" gorm:"type:text;not null;default:''"`
	StartedAt          *time.Time `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// UserSession 用户会话记录，用于多设备登录管理。
// 每个 JWT 会话对应一条记录，通过 jti (JWT ID) 唯一标识。
type UserSession struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UserID       uint       `json:"user_id" gorm:"index;not null"`
	SessionJTI   string     `json:"-" gorm:"uniqueIndex;size:64;not null"` // JWT 的 jti (token ID) SHA-256 哈希
	DeviceName   string     `json:"device_name" gorm:"size:128"`            // "Chrome on Windows", "Android App"
	DeviceType   string     `json:"device_type" gorm:"size:32;not null;default:'browser'"` // browser / native
	UserAgent    string     `json:"user_agent" gorm:"size:512"`
	IPHash       string     `json:"-" gorm:"size:64"`
	LastActiveAt time.Time  `json:"last_active_at" gorm:"not null"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"index;not null"`
	CreatedAt    time.Time  `json:"created_at"`
}

// 设备类型常量
const (
	DeviceTypeBrowser = "browser"
	DeviceTypeNative  = "native"
)
