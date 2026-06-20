package model

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID           uint            `json:"id" gorm:"primaryKey"`
	Title        string          `json:"title" gorm:"type:varchar(50);not null"`
	Content      string          `json:"content" gorm:"type:text;not null"`
	MessageKey   string          `json:"message_key,omitempty" gorm:"type:varchar(64)"`
	MessageArgs  json.RawMessage `json:"message_args,omitempty" gorm:"type:jsonb"`
	Type         string          `json:"type" gorm:"type:varchar(10);not null"`
	TargetType   string          `json:"target_type" gorm:"type:varchar(10);not null;index"`
	TargetIDs    json.RawMessage `json:"target_ids,omitempty" gorm:"type:jsonb"`
	VisibleToNew bool            `json:"visible_to_new" gorm:"default:false"`
	CreatedBy    uint            `json:"created_by"`
	Creator      *User           `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt    time.Time       `json:"created_at" gorm:"index"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type NotificationRead struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	NotificationID uint      `json:"notification_id" gorm:"uniqueIndex:idx_notification_user;not null"`
	UserID         uint      `json:"user_id" gorm:"uniqueIndex:idx_notification_user;not null"`
	ReadAt         time.Time `json:"read_at" gorm:"autoCreateTime"`
}

type NotificationImage struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Data           []byte    `json:"-" gorm:"type:bytea;not null"`
	Size           int       `json:"size" gorm:"not null"`
	NotificationID *uint     `json:"notification_id" gorm:"index"`
	CreatedBy      uint      `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}
