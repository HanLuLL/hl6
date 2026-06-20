package service

import (
	"errors"
	"strings"
)

// ProviderErrorCategory defines machine-readable error categories for all DNS provider errors.
// These categories are used for frontend display and retry decisions.
type ProviderErrorCategory string

const (
	ErrCategoryAuthFailed              ProviderErrorCategory = "auth_failed"
	ErrCategoryPermissionDenied        ProviderErrorCategory = "permission_denied"
	ErrCategoryRecordConflict          ProviderErrorCategory = "record_conflict"
	ErrCategoryRecordNotFound          ProviderErrorCategory = "record_not_found"
	ErrCategoryProviderUnavailable     ProviderErrorCategory = "provider_unavailable"
	ErrCategoryInvalidRequest          ProviderErrorCategory = "invalid_request"
	ErrCategoryDomainMigrationReadOnly ProviderErrorCategory = "domain_migration_read_only"
	ErrCategoryUnknown                 ProviderErrorCategory = "unknown"
)

// ProviderError wraps a provider error with a standardized category.
type ProviderError struct {
	Category  ProviderErrorCategory
	Retryable bool
	Err       error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Category)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a ProviderError with the given category.
func NewProviderError(category ProviderErrorCategory, retryable bool, err error) *ProviderError {
	return &ProviderError{Category: category, Retryable: retryable, Err: err}
}

// ClassifyProviderError maps a raw provider error to a standardized category.
func ClassifyProviderError(err error) (ProviderErrorCategory, bool) {
	if err == nil {
		return ErrCategoryUnknown, false
	}

	// Check for typed record-not-found errors
	if isProviderRecordNotFoundErr(err) {
		return ErrCategoryRecordNotFound, false
	}

	// Check for migration read-only error
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Category, pe.Retryable
	}

	msg := strings.ToLower(err.Error())

	// Auth errors
	if containsAny(msg, "auth", "unauthorized", "invalid api token", "invalid credentials",
		"authentication", "apitoken", "forbidden", "401", "403") {
		// 403 can be either auth or permission; check more specifically
		if containsAny(msg, "forbidden", "permission", "not allowed", "access denied", "403") {
			return ErrCategoryPermissionDenied, false
		}
		return ErrCategoryAuthFailed, false
	}

	// Permission errors
	if containsAny(msg, "permission denied", "not permitted", "insufficient privileges") {
		return ErrCategoryPermissionDenied, false
	}

	// Provider unavailable / transient
	if containsAny(msg, "timeout", "connection refused", "connection reset", "eof",
		"service unavailable", "502", "503", "504", "temporarily",
		"rate limit", "too many requests", "429", "throttl") {
		return ErrCategoryProviderUnavailable, true
	}

	// Record conflict
	if containsAny(msg, "already exists", "duplicate", "conflict", "record_conflict") {
		return ErrCategoryRecordConflict, false
	}

	// Record not found (string-based fallback)
	if containsAny(msg, "record not found", "does not exist", "no record", "404") {
		return ErrCategoryRecordNotFound, false
	}

	// Invalid request
	if containsAny(msg, "invalid", "malformed", "bad request", "400", "validation") {
		return ErrCategoryInvalidRequest, false
	}

	// Internal provider errors
	if containsAny(msg, "internal server error", "500") {
		return ErrCategoryProviderUnavailable, true
	}

	return ErrCategoryUnknown, false
}

// IsRetryableCategory returns true if the error category is generally retryable.
func IsRetryableCategory(category ProviderErrorCategory) bool {
	switch category {
	case ErrCategoryProviderUnavailable:
		return true
	case ErrCategoryDomainMigrationReadOnly:
		return true // conditional: retry after migration completes
	default:
		return false
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
