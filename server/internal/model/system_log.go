package model

import (
	"encoding/json"
	"time"
)

// SystemLog represents a structured system log entry stored in the database.
type SystemLog struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	Level     string          `json:"level" gorm:"type:varchar(16);not null;index"` // DEBUG, INFO, WARN, ERROR
	Module    string          `json:"module" gorm:"type:varchar(64);not null;index"` // auth, dns, subdomain, etc.
	Message   string          `json:"message" gorm:"type:text;not null"`
	Fields    json.RawMessage `json:"fields" gorm:"type:jsonb"`
	CreatedAt time.Time       `json:"created_at" gorm:"index"`
}

// TableName returns the table name for SystemLog.
func (SystemLog) TableName() string {
	return "system_logs"
}

// SystemLogLevel constants.
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// ValidLogLevels contains all valid log level values.
var ValidLogLevels = []string{
	LogLevelDebug,
	LogLevelInfo,
	LogLevelWarn,
	LogLevelError,
}

// IsValidLogLevel checks if the given level is valid.
func IsValidLogLevel(level string) bool {
	for _, l := range ValidLogLevels {
		if l == level {
			return true
		}
	}
	return false
}

// ToLogEntry converts a SystemLog to a logger.SystemLogEntry.
func (sl *SystemLog) ToLogEntry() map[string]interface{} {
	result := map[string]interface{}{
		"id":        sl.ID,
		"level":     sl.Level,
		"module":    sl.Module,
		"message":   sl.Message,
		"timestamp": sl.CreatedAt,
	}
	if sl.Fields != nil {
		var fields map[string]interface{}
		if err := json.Unmarshal(sl.Fields, &fields); err == nil {
			result["fields"] = fields
		}
	}
	return result
}