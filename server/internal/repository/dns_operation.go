package repository

import (
	"encoding/json"

	"hl6-server/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) TryCreateDNSOperationRequest(scope, key string) (*model.DNSOperationRequest, bool, error) {
	req := &model.DNSOperationRequest{
		Scope:          scope,
		IdempotencyKey: key,
		Status:         model.DNSOperationRequestStatusRunning,
		HTTPStatus:     202,
		Message:        "running",
	}
	if err := r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "scope"}, {Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(req).Error; err != nil {
		return nil, false, err
	}
	if req.ID != 0 {
		return req, true, nil
	}
	existing, err := r.FindDNSOperationRequest(scope, key)
	if err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (r *Repository) FindDNSOperationRequest(scope, key string) (*model.DNSOperationRequest, error) {
	var req model.DNSOperationRequest
	err := r.DB.Where("scope = ? AND idempotency_key = ?", scope, key).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *Repository) CompleteDNSOperationRequest(reqID uint, status string, httpStatus int, message, messageKey string, retryable bool, data interface{}) error {
	var raw json.RawMessage
	if data != nil {
		encoded, err := json.Marshal(data)
		if err != nil {
			return err
		}
		raw = encoded
	}
	updates := map[string]interface{}{
		"status":        status,
		"http_status":   httpStatus,
		"message":       message,
		"message_key":   messageKey,
		"retryable":     retryable,
		"response_data": raw,
	}
	return r.DB.Model(&model.DNSOperationRequest{}).Where("id = ?", reqID).Updates(updates).Error
}

func (r *Repository) CreateDNSOperationEvent(event *model.DNSOperationEvent) error {
	if event == nil {
		return nil
	}
	return r.DB.Create(event).Error
}

func (r *Repository) GetOperationRequestTx(tx *gorm.DB, scope, key string) (*model.DNSOperationRequest, error) {
	db := r.DB
	if tx != nil {
		db = tx
	}
	var req model.DNSOperationRequest
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("scope = ? AND idempotency_key = ?", scope, key).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}
