package service

import (
	"encoding/json"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
)

// AuditLogService 审核相关审计日志写入入口。
type AuditLogService struct {
	repo *repository.Repository
}

func NewAuditLogService(repo *repository.Repository) *AuditLogService {
	return &AuditLogService{repo: repo}
}

func marshalAuditDetails(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return raw
}

func (s *AuditLogService) RecordUser(userID uint, action, resource string, resourceID uint, details any) error {
	log := &model.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    marshalAuditDetails(details),
	}
	if err := s.repo.CreateAuditLog(log); err != nil {
		slog.Warn("audit log write failed", "action", action, "err", err)
		return err
	}
	return nil
}

func (s *AuditLogService) RecordUserTx(tx *gorm.DB, userID uint, action, resource string, resourceID uint, details any) error {
	log := &model.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    marshalAuditDetails(details),
	}
	return s.repo.CreateAuditLogTx(tx, log)
}

func (s *AuditLogService) RecordFromHTTP(c *gin.Context, userID uint, action, resource string, resourceID uint, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	if c != nil && c.ClientIP() != "" {
		details["client_ip"] = c.ClientIP()
	}
	return s.RecordUser(userID, action, resource, resourceID, details)
}
