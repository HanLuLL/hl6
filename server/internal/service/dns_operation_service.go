package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"

	"gorm.io/gorm"
)

const (
	singleOperationTimeout   = 3 * time.Second
	singleOperationAttempts  = 2
	defaultBatchMaxAttempts  = 3
	retryBackoffMilliseconds = 180
)

type OperationResult struct {
	HTTPStatus int         `json:"http_status"`
	Message    string      `json:"message"`
	MessageKey string      `json:"message_key,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Retryable  bool        `json:"retryable"`
}

type CreateRecordInput struct {
	Subdomain *model.Subdomain
	Record    *model.DNSRecord
}

type UpdateRecordInput struct {
	Subdomain    *model.Subdomain
	Record       *model.DNSRecord
	NewContent   string
	NewTTL       int
	NewProxied   bool
	BeforeDBSave func(tx *gorm.DB, record *model.DNSRecord) error
}

type DeleteRecordInput struct {
	Subdomain *model.Subdomain
	Record    *model.DNSRecord
}

type BatchDeleteItem struct {
	RecordID          uint
	SubdomainFQDN     string
	Provider          string
	ProviderAccountID uint
	ZoneID            string
	ProviderRecordID  string
	RecordType        string
	Name              string
	Content           string
}

type BatchDeleteFailure struct {
	RecordID         uint   `json:"record_id"`
	SubdomainFQDN    string `json:"subdomain_fqdn"`
	RecordType       string `json:"record_type"`
	RecordContent    string `json:"record_content"`
	ProviderRecordID string `json:"provider_record_id"`
	Error            string `json:"error"`
}

type BatchDeleteResult struct {
	Total     int                  `json:"total"`
	Succeeded int                  `json:"succeeded"`
	Failed    int                  `json:"failed"`
	Failures  []BatchDeleteFailure `json:"failures"`
}

type DNSOperationService struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewDNSOperationService(repo *repository.Repository, cfg *config.Config) *DNSOperationService {
	return &DNSOperationService{repo: repo, cfg: cfg}
}

func (s *DNSOperationService) ExecuteIdempotent(ctx context.Context, scope, idempotencyKey string, exec func(context.Context) (OperationResult, error)) OperationResult {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return OperationResult{
			HTTPStatus: 400,
			Message:    "missing idempotency key",
			MessageKey: "error.invalidRequestBody",
			Retryable:  false,
		}
	}
	req, created, err := s.repo.TryCreateDNSOperationRequest(scope, key)
	if err != nil {
		return OperationResult{
			HTTPStatus: 500,
			Message:    "failed to persist idempotency request",
			MessageKey: "error.databaseError",
			Retryable:  true,
		}
	}

	if !created {
		if req.Status == model.DNSOperationRequestStatusRunning {
			return OperationResult{
				HTTPStatus: 409,
				Message:    "operation is in progress",
				MessageKey: "error.cloudflareOperationInProgress",
				Retryable:  true,
			}
		}
		return decodeOperationResult(req)
	}

	result, execErr := exec(ctx)
	if execErr != nil {
		result = OperationResult{
			HTTPStatus: 500,
			Message:    execErr.Error(),
			MessageKey: "error.databaseError",
			Retryable:  true,
		}
	}
	if result.HTTPStatus == 0 {
		result.HTTPStatus = 500
		if result.Message == "" {
			result.Message = "empty operation result"
		}
		if result.MessageKey == "" {
			result.MessageKey = "error.databaseError"
		}
		result.Retryable = true
	}

	status := model.DNSOperationRequestStatusSucceeded
	if result.HTTPStatus >= 400 {
		status = model.DNSOperationRequestStatusFailed
	}
	if completeErr := s.repo.CompleteDNSOperationRequest(req.ID, status, result.HTTPStatus, result.Message, result.MessageKey, result.Retryable, result.Data); completeErr != nil {
		return OperationResult{
			HTTPStatus: 500,
			Message:    "failed to persist operation result",
			MessageKey: "error.databaseError",
			Retryable:  true,
		}
	}
	return result
}

func decodeOperationResult(req *model.DNSOperationRequest) OperationResult {
	res := OperationResult{
		HTTPStatus: req.HTTPStatus,
		Message:    req.Message,
		MessageKey: req.MessageKey,
		Retryable:  req.Retryable,
	}
	if res.HTTPStatus == 0 {
		if req.Status == model.DNSOperationRequestStatusSucceeded {
			res.HTTPStatus = 200
		} else {
			res.HTTPStatus = 500
		}
	}
	if len(req.ResponseData) > 0 {
		var data interface{}
		if err := json.Unmarshal(req.ResponseData, &data); err == nil {
			res.Data = data
		}
	}
	return res
}

func (s *DNSOperationService) CreateRecordAtomic(ctx context.Context, input CreateRecordInput, beforePersist func(tx *gorm.DB) error) error {
	if input.Subdomain == nil || input.Record == nil {
		return errors.New("create record input is incomplete")
	}
	client, provider, err := s.providerClientForAccount(input.Subdomain.Domain.ProviderAccountID, input.Subdomain.Domain.Provider)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, singleOperationTimeout)
	defer cancel()

	providerRecordID, err := retryValue(ctx, singleOperationAttempts, func(callCtx context.Context) (string, error) {
		return client.CreateRecord(callCtx,
			input.Subdomain.Domain.ProviderZoneID,
			input.Record.Type,
			input.Record.Name,
			input.Record.Content,
			input.Record.TTL,
			input.Record.Proxied,
		)
	})
	if err != nil {
		s.logEvent(nil, "create_provider", false, err.Error(), map[string]interface{}{"record": input.Record.Name}, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, "")
		return err
	}

	input.Record.ProviderRecordID = strings.TrimSpace(providerRecordID)
	if dbErr := s.repo.Transaction(func(tx *gorm.DB) error {
		if beforePersist != nil {
			if err := beforePersist(tx); err != nil {
				return err
			}
		}
		return tx.Create(input.Record).Error
	}); dbErr != nil {
		rollbackErr := retryError(ctx, singleOperationAttempts, func(callCtx context.Context) error {
			return client.DeleteRecord(callCtx, input.Subdomain.Domain.ProviderZoneID, providerRecordID)
		})
		s.logEvent(nil, "create_db", false, dbErr.Error(), map[string]interface{}{"provider_record_id": providerRecordID}, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
		if rollbackErr != nil {
			s.logEvent(nil, "create_compensate_delete", false, rollbackErr.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
			return fmt.Errorf("database create failed (%v), provider rollback failed (%w)", dbErr, rollbackErr)
		}
		s.logEvent(nil, "create_compensate_delete", true, "rollback succeeded", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
		return fmt.Errorf("database create failed after provider create: %w", dbErr)
	}

	s.logEvent(nil, "create_done", true, "ok", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, providerRecordID)
	return nil
}

func (s *DNSOperationService) UpdateRecordAtomic(ctx context.Context, input UpdateRecordInput) error {
	if input.Subdomain == nil || input.Record == nil {
		return errors.New("update record input is incomplete")
	}
	client, provider, err := s.providerClientForAccount(input.Subdomain.Domain.ProviderAccountID, input.Subdomain.Domain.Provider)
	if err != nil {
		return err
	}

	oldRecord := *input.Record
	recordID := strings.TrimSpace(input.Record.ProviderRecordID)
	if recordID == "" {
		return errors.New("provider record id is empty")
	}

	ctx, cancel := context.WithTimeout(ctx, singleOperationTimeout)
	defer cancel()

	if err := retryError(ctx, singleOperationAttempts, func(callCtx context.Context) error {
		return client.UpdateRecord(callCtx,
			input.Subdomain.Domain.ProviderZoneID,
			recordID,
			input.Record.Type,
			input.Record.Name,
			input.NewContent,
			input.NewTTL,
			input.NewProxied,
		)
	}); err != nil {
		s.logEvent(nil, "update_provider", false, err.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
		return err
	}

	if dbErr := s.repo.Transaction(func(tx *gorm.DB) error {
		input.Record.Content = input.NewContent
		input.Record.TTL = input.NewTTL
		input.Record.Proxied = input.NewProxied
		if input.BeforeDBSave != nil {
			if err := input.BeforeDBSave(tx, input.Record); err != nil {
				return err
			}
		}
		return tx.Save(input.Record).Error
	}); dbErr != nil {
		rollbackErr := retryError(ctx, singleOperationAttempts, func(callCtx context.Context) error {
			return client.UpdateRecord(callCtx,
				input.Subdomain.Domain.ProviderZoneID,
				recordID,
				oldRecord.Type,
				oldRecord.Name,
				oldRecord.Content,
				oldRecord.TTL,
				oldRecord.Proxied,
			)
		})
		s.logEvent(nil, "update_db", false, dbErr.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
		if rollbackErr != nil {
			s.logEvent(nil, "update_compensate", false, rollbackErr.Error(), nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
			return fmt.Errorf("database update failed (%v), provider rollback failed (%w)", dbErr, rollbackErr)
		}
		s.logEvent(nil, "update_compensate", true, "rollback succeeded", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
		return fmt.Errorf("database update failed after provider update: %w", dbErr)
	}

	s.logEvent(nil, "update_done", true, "ok", nil, provider, input.Subdomain.Domain.ProviderZoneID, input.Record.ID, recordID)
	return nil
}

func (s *DNSOperationService) DeleteRecordAtomic(ctx context.Context, input DeleteRecordInput) error {
	if input.Subdomain == nil || input.Record == nil {
		return errors.New("delete record input is incomplete")
	}
	item := BatchDeleteItem{
		RecordID:          input.Record.ID,
		SubdomainFQDN:     input.Subdomain.FQDN,
		Provider:          input.Subdomain.Domain.Provider,
		ProviderAccountID: input.Subdomain.Domain.ProviderAccountID,
		ZoneID:            input.Subdomain.Domain.ProviderZoneID,
		ProviderRecordID:  input.Record.ProviderRecordID,
		RecordType:        input.Record.Type,
		Name:              input.Record.Name,
		Content:           input.Record.Content,
	}
	_, err := s.deleteSingleWithRetry(ctx, item, defaultBatchMaxAttempts)
	return err
}

func (s *DNSOperationService) DeleteRecordsBatch(ctx context.Context, items []BatchDeleteItem, maxAttempts int) BatchDeleteResult {
	if maxAttempts <= 0 {
		maxAttempts = defaultBatchMaxAttempts
	}
	result := BatchDeleteResult{Total: len(items), Failures: make([]BatchDeleteFailure, 0)}
	for _, item := range items {
		success, err := s.deleteSingleWithRetry(ctx, item, maxAttempts)
		if success {
			result.Succeeded++
			continue
		}
		result.Failed++
		failure := BatchDeleteFailure{
			RecordID:         item.RecordID,
			SubdomainFQDN:    item.SubdomainFQDN,
			RecordType:       item.RecordType,
			RecordContent:    item.Content,
			ProviderRecordID: item.ProviderRecordID,
		}
		if err != nil {
			failure.Error = err.Error()
		}
		result.Failures = append(result.Failures, failure)
	}
	return result
}

func (s *DNSOperationService) deleteSingleWithRetry(ctx context.Context, item BatchDeleteItem, maxAttempts int) (bool, error) {
	if maxAttempts <= 0 {
		maxAttempts = defaultBatchMaxAttempts
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := s.deleteSingle(ctx, item); err == nil {
			return true, nil
		} else {
			lastErr = err
		}
		if attempt == maxAttempts || !shouldRetryProviderError(lastErr) {
			break
		}
		timer := time.NewTimer(time.Duration(attempt*retryBackoffMilliseconds) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, ctx.Err()
		case <-timer.C:
		}
	}
	return false, lastErr
}

func (s *DNSOperationService) deleteSingle(ctx context.Context, item BatchDeleteItem) error {
	client, provider, err := s.providerClientForAccount(item.ProviderAccountID, item.Provider)
	if err != nil {
		return err
	}

	callCtx, cancel := context.WithTimeout(ctx, singleOperationTimeout)
	defer cancel()

	recordID := strings.TrimSpace(item.ProviderRecordID)
	if recordID == "" {
		if strings.TrimSpace(item.RecordType) != "" && strings.TrimSpace(item.Name) != "" && strings.TrimSpace(item.Content) != "" {
			recordID, err = retryValue(callCtx, singleOperationAttempts, func(c context.Context) (string, error) {
				return client.FindRecord(c, item.ZoneID, item.RecordType, item.Name, item.Content)
			})
			if err != nil {
				if isProviderRecordNotFoundErr(err) {
					recordID = ""
				} else {
					s.logEvent(nil, "delete_find_provider", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, "")
					return err
				}
			}
		}
	}

	if recordID != "" {
		if err := retryError(callCtx, singleOperationAttempts, func(c context.Context) error {
			return client.DeleteRecord(c, item.ZoneID, recordID)
		}); err != nil {
			s.logEvent(nil, "delete_provider", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
			return err
		}
	}

	if item.RecordID > 0 {
		if err := s.repo.DeleteDNSRecord(item.RecordID); err != nil {
			s.logEvent(nil, "delete_db", false, err.Error(), nil, provider, item.ZoneID, item.RecordID, recordID)
			return fmt.Errorf("delete local record failed: %w", err)
		}
	}

	s.logEvent(nil, "delete_done", true, "ok", nil, provider, item.ZoneID, item.RecordID, recordID)
	return nil
}

func (s *DNSOperationService) providerClientForAccount(accountID uint, providerHint string) (DNSProviderClient, string, error) {
	if accountID == 0 {
		return nil, "", errors.New("provider account id is required")
	}
	account, err := s.repo.FindDNSProviderAccount(accountID)
	if err != nil {
		return nil, "", err
	}
	rawCredentials := strings.TrimSpace(account.Credentials)
	if rawCredentials == "" {
		rawCredentials = strings.TrimSpace(account.LegacyAPIToken)
	}
	credentials, err := ParseProviderCredentials(account.Provider, crypto.DecryptOrPlaintext(rawCredentials, s.cfg.EncryptionKey))
	if err != nil {
		return nil, "", err
	}

	provider := model.NormalizeProvider(account.Provider)
	if provider == "" {
		provider = model.DNSProviderCloudflare
	}
	if hint := model.NormalizeProvider(providerHint); hint != "" {
		provider = hint
	}
	client, err := BuildProviderClient(provider, credentials)
	if err != nil {
		return nil, "", err
	}
	return client, provider, nil
}

func retryError(ctx context.Context, attempts int, fn func(context.Context) error) error {
	_, err := retryValue(ctx, attempts, func(c context.Context) (struct{}, error) {
		return struct{}{}, fn(c)
	})
	return err
}

func retryValue[T any](ctx context.Context, attempts int, fn func(context.Context) (T, error)) (T, error) {
	if attempts < 1 {
		attempts = 1
	}
	var zero T
	var lastErr error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		value, err := fn(ctx)
		if err == nil {
			return value, nil
		}
		lastErr = err
		if i == attempts-1 || !shouldRetryProviderError(err) {
			break
		}
		timer := time.NewTimer(time.Duration((i+1)*retryBackoffMilliseconds) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, lastErr
}

func shouldRetryProviderError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := strings.ToLower(err.Error())
	transientTokens := []string{
		"timeout",
		"tempor",
		"connection reset",
		"connection refused",
		"eof",
		"429",
		"500",
		"502",
		"503",
		"504",
		"rate limit",
	}
	for _, token := range transientTokens {
		if strings.Contains(msg, token) {
			return true
		}
	}
	return false
}

func (s *DNSOperationService) logEvent(requestID *uint, step string, success bool, message string, detail interface{}, provider, zone string, recordID uint, providerRecordID string) {
	var raw json.RawMessage
	if detail != nil {
		if encoded, err := json.Marshal(detail); err == nil {
			raw = encoded
		}
	}
	_ = s.repo.CreateDNSOperationEvent(&model.DNSOperationEvent{
		RequestID:     requestID,
		Scope:         "dns_operation",
		Step:          step,
		Success:       success,
		Message:       message,
		Detail:        raw,
		Provider:      provider,
		ProviderZone:  zone,
		RecordID:      recordID,
		ProviderRecID: providerRecordID,
	})
}
