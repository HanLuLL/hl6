package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/oidc"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/crypto"
	"hl6-server/pkg/response"
)

const adminBanGuardLockKey int64 = 19490332

var errCannotBanLastActiveAdmin = errors.New("cannot ban last active admin")

var allowedConfigKeys = map[string]bool{
	"registration_bonus_credits": true,
	"referral_enabled":           true,
	"referral_inviter_credits":   true,
	"referral_invitee_credits":   true,
	"daily_checkin_enabled":      true,
	"daily_checkin_credits":      true,
	"daily_checkin_group_ids":    true,
	"frontend_urls":              true,
	"frontend_url":               true,
	"backend_urls":               true,
	"backend_url":                true,
	"oidc_issuer":                true,
	"oidc_client_id":             true,
	"oidc_client_secret":         true,
}

type AdminHandler struct {
	repo         *repository.Repository
	cfg          *config.Config
	ops          *service.DNSOperationService
	urlResolver  *URLResolver
	oidcResolver *OIDCRuntimeResolver
}

func NewAdminHandler(repo *repository.Repository, cfg *config.Config, ops *service.DNSOperationService) *AdminHandler {
	return &AdminHandler{
		repo:         repo,
		cfg:          cfg,
		ops:          ops,
		urlResolver:  NewURLResolver(repo, cfg),
		oidcResolver: NewOIDCRuntimeResolver(repo, cfg),
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, perPage := helpers.ParsePageParams(c, 50, 100)
	search := c.Query("search")
	inviter := strings.TrimSpace(c.Query("inviter"))
	banStatus := strings.ToLower(strings.TrimSpace(c.DefaultQuery("ban_status", "all")))
	role := strings.ToLower(strings.TrimSpace(c.DefaultQuery("role", "all")))
	groupIDStr := strings.TrimSpace(c.Query("group_id"))
	var groupID *uint
	switch banStatus {
	case "all", "active", "banned":
	default:
		banStatus = "all"
	}
	switch role {
	case "all", "user", "admin":
	default:
		role = "all"
	}
	if groupIDStr != "" {
		parsed, err := strconv.ParseUint(groupIDStr, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid group_id")
			return
		}
		parsedID := uint(parsed)
		groupID = &parsedID
	}
	users, total, err := h.repo.ListUsers(page, perPage, search, banStatus, role, groupID, inviter)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list users", "error.failedToListUsers")
		return
	}

	// Batch fetch referral inviters
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	inviterMap, invErr := h.repo.GetReferralInvitersForUsers(userIDs)
	if invErr != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list users", "error.failedToListUsers")
		return
	}

	type userDTO struct {
		model.User
		Credits   model.Credit `json:"credits"`
		InvitedBy *struct {
			ID    uint   `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"invited_by"`
	}

	result := make([]userDTO, len(users))
	for i, u := range users {
		dto := userDTO{User: u.User, Credits: u.Credits}
		if inviter, ok := inviterMap[u.ID]; ok {
			dto.InvitedBy = &struct {
				ID    uint   `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			}{ID: inviter.ID, Name: inviter.Name, Email: inviter.Email}
		}
		result[i] = dto
	}

	response.Paginated(c, result, total, page, perPage)
}

func (h *AdminHandler) Stats(c *gin.Context) {
	stats, err := h.repo.GetStats()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get stats", "error.failedToGetStats")
		return
	}
	response.OK(c, stats)
}

// GetDNSProviderStatus returns aggregated health status for all DNS providers.
// GET /api/v1/admin/dns-providers/status
func (h *AdminHandler) GetDNSProviderStatus(c *gin.Context) {
	entries, err := h.repo.GetDNSProviderStatus()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get provider status", "error.databaseError")
		return
	}
	response.OK(c, entries)
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	page, perPage := helpers.ParsePageParams(c, 15, 100)
	operator := strings.TrimSpace(c.Query("operator"))
	action := strings.TrimSpace(c.Query("action"))

	logs, total, err := h.repo.ListAuditLogs(page, perPage, operator, action)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list audit logs", "error.failedToListAuditLogs")
		return
	}
	response.Paginated(c, logs, total, page, perPage)
}

// User Group CRUD

func (h *AdminHandler) ListGroups(c *gin.Context) {
	groups, err := h.repo.ListUserGroups()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list groups", "error.failedToListGroups")
		return
	}
	response.OK(c, groups)
}

func (h *AdminHandler) CreateGroup(c *gin.Context) {
	var body struct {
		Name      string `json:"name" binding:"required"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	group := &model.UserGroup{
		Name: body.Name,
	}

	if err := h.repo.CreateUserGroup(group); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create group", "error.failedToCreateGroup")
		return
	}

	if body.IsDefault {
		if err := h.repo.SetDefaultUserGroup(group.ID); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update default group", "error.failedToUpdateDefaultGroup")
			return
		}
		group.IsDefault = true
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{"group_name": body.Name})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_create_group",
			Resource:   "user_group",
			ResourceID: group.ID,
			Details:    details,
		})
	}
	response.Created(c, group)
}

func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	group, err := h.repo.FindUserGroup(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "group not found", "error.groupNotFound")
		return
	}

	var body struct {
		Name      *string `json:"name"`
		IsDefault *bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	if body.Name != nil {
		group.Name = *body.Name
	}
	if body.IsDefault != nil && *body.IsDefault {
		if err := h.repo.SetDefaultUserGroup(group.ID); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to set default group", "error.failedToSetDefaultGroup")
			return
		}
		group.IsDefault = true
	}

	if err := h.repo.UpdateUserGroup(group); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update group", "error.failedToUpdateGroup")
		return
	}
	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{"group_name": group.Name})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_update_group",
			Resource:   "user_group",
			ResourceID: group.ID,
			Details:    details,
		})
	}
	response.OK(c, group)
}

func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	migrateToStr := c.Query("migrate_to")
	if migrateToStr == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "migrate_to parameter is required", "error.migrateToRequired")
		return
	}
	migrateTo, err := strconv.ParseUint(migrateToStr, 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid migrate_to parameter", "error.invalidMigrateTo")
		return
	}
	if id == uint(migrateTo) {
		response.ErrorWithKey(c, http.StatusBadRequest, "cannot migrate to the same group being deleted", "error.cannotMigrateToSameGroup")
		return
	}

	count, err := h.repo.CountUserGroups()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	if count <= 1 {
		response.ErrorWithKey(c, http.StatusBadRequest, "cannot delete the last group", "error.cannotDeleteLastGroup")
		return
	}

	group, err := h.repo.FindUserGroup(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "group not found", "error.groupNotFound")
		return
	}

	targetGroup, err := h.repo.FindUserGroup(uint(migrateTo))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "target group not found", "error.targetGroupNotFound")
		return
	}

	if err := h.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("group_id = ?", group.ID).Update("group_id", targetGroup.ID).Error; err != nil {
			return err
		}
		if err := h.repo.DeleteDomainGroupAccessByGroup(tx, group.ID); err != nil {
			return err
		}
		if group.IsDefault {
			if err := tx.Model(&model.UserGroup{}).Where("id = ?", targetGroup.ID).Update("is_default", true).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&model.UserGroup{}, group.ID).Error
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete group", "error.failedToDeleteGroup")
		return
	}
	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{"group_name": group.Name, "migrated_to": targetGroup.Name})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_delete_group",
			Resource:   "user_group",
			ResourceID: group.ID,
			Details:    details,
		})
	}
	response.OK(c, gin.H{"message": "group deleted and users migrated"})
}

// Update user's group
func (h *AdminHandler) UpdateUserGroup(c *gin.Context) {
	userID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var body struct {
		GroupID uint `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	// Verify group exists
	if _, err := h.repo.FindUserGroup(body.GroupID); err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "group not found", "error.groupNotFound")
		return
	}

	if err := h.repo.UpdateUserGroupID(userID, body.GroupID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update user group", "error.failedToUpdateUserGroup")
		return
	}
	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{"target_user_id": userID, "new_group_id": body.GroupID})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_change_user_group",
			Resource:   "user",
			ResourceID: userID,
			Details:    details,
		})
	}
	response.OK(c, gin.H{"message": "user group updated"})
}

// Ban a user and always delete all owned subdomains and DNS records.
func (h *AdminHandler) BanUser(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	userID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	if admin.ID == userID {
		response.ErrorWithKey(c, http.StatusBadRequest, "cannot ban yourself", "error.cannotBanSelf")
		return
	}

	target, err := h.repo.FindUserByID(userID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "user not found", "error.userNotFound")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	reason := strings.TrimSpace(body.Reason)
	key, ok := idempotencyKeyFromHeader(c)
	if !ok {
		return
	}
	scope := fmt.Sprintf("admin:ban:user:%d", target.ID)
	opResult := h.ops.ExecuteIdempotent(c.Request.Context(), scope, key, func(ctx context.Context) (service.OperationResult, error) {
		result, failures, asyncJobID, err := executeAdminBanUserWithCleanup(ctx, h.repo, h.ops, admin.ID, target, reason)
		if asyncJobID != nil {
			return service.OperationResult{
				HTTPStatus: http.StatusConflict,
				Message:    "dns bulk delete queued, retry ban after job succeeds",
				MessageKey: "error.cloudflareOperationInProgress",
				Data:       gin.H{"bulk_job_id": *asyncJobID, "bulk_async": true},
			}, nil
		}
		if len(failures) > 0 {
			return service.OperationResult{
				HTTPStatus: http.StatusConflict,
				Message:    "some cloudflare dns records failed to delete",
				MessageKey: "error.cloudflareDeleteFailed",
				Data:       gin.H{"failed_records": failures},
			}, nil
		}
		if err != nil {
			if errors.Is(err, errCannotBanLastActiveAdmin) {
				return service.OperationResult{HTTPStatus: http.StatusBadRequest, Message: "cannot ban last active admin", MessageKey: "error.cannotBanLastAdmin"}, nil
			}
			return service.OperationResult{HTTPStatus: http.StatusInternalServerError, Message: "failed to ban user", MessageKey: "error.failedToBanUser", Retryable: true}, nil
		}

		details, _ := json.Marshal(map[string]interface{}{
			"target_user_id":     target.ID,
			"target_user_role":   result.TargetRole,
			"reason":             reason,
			"delete_resources":   true,
			"subdomains_deleted": result.SubdomainsDeleted,
			"deleted_dns_count":  result.DeletedDNSCount,
		})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_ban_user",
			Resource:   "user",
			ResourceID: target.ID,
			Details:    details,
		})
		return service.OperationResult{
			HTTPStatus: http.StatusOK,
			Message:    "ok",
			Data:       gin.H{"message": "user banned", "deleted_dns_count": result.DeletedDNSCount},
		}, nil
	})
	writeOperationResult(c, opResult)
}

func (h *AdminHandler) UnbanUser(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	userID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	target, err := h.repo.FindUserByID(userID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "user not found", "error.userNotFound")
		return
	}

	if err := h.repo.GetDB().Model(&model.User{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
		"is_banned":     false,
		"banned_reason": "",
		"banned_at":     nil,
		"banned_by":     nil,
	}).Error; err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to unban user", "error.failedToUnbanUser")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{
		"target_user_id": target.ID,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_unban_user",
		Resource:   "user",
		ResourceID: target.ID,
		Details:    details,
	})

	response.OK(c, gin.H{"message": "user unbanned"})
}

// System Config

func (h *AdminHandler) GetConfig(c *gin.Context) {
	keys := make([]string, 0, len(allowedConfigKeys))
	for key := range allowedConfigKeys {
		keys = append(keys, key)
	}
	configs, err := h.repo.GetSystemConfigsByKeys(keys)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get config", "error.failedToGetConfig")
		return
	}
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime url config", "error.failedToGetConfig")
		return
	}
	oidcState, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime oidc config", "error.failedToGetConfig")
		return
	}

	// Never return client secret value to frontend.
	delete(configs, configKeyOIDCClientSecret)

	response.OK(c, gin.H{
		"values": configs,
		"url_runtime": gin.H{
			"frontend_urls":       urlState.FrontendURLs,
			"backend_urls":        urlState.BackendURLs,
			"frontend_url":        urlState.FrontendURL,
			"backend_url":         urlState.BackendURL,
			"frontend_source":     urlState.FrontendSource,
			"backend_source":      urlState.BackendSource,
			"frontend_env_locked": urlState.FrontendEnvLocked,
			"backend_env_locked":  urlState.BackendEnvLocked,
			"confirmed":           urlState.Confirmed,
		},
		"oidc_runtime": gin.H{
			"configured":               oidcState.Configured,
			"missing_fields":           oidcState.MissingFields,
			"issuer":                   oidcState.Issuer,
			"client_id":                oidcState.ClientID,
			"issuer_source":            oidcState.IssuerSource,
			"client_id_source":         oidcState.ClientIDSource,
			"client_secret_source":     oidcState.ClientSecretSource,
			"issuer_env_locked":        oidcState.IssuerEnvLocked,
			"client_id_env_locked":     oidcState.ClientIDEnvLocked,
			"client_secret_env_locked": oidcState.ClientSecretEnvLocked,
			"client_secret_configured": oidcState.ClientSecretConfigured,
		},
	})
}

func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	for key := range body {
		if !allowedConfigKeys[key] {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("config key %q is not allowed", key))
			return
		}
	}

	details := make(map[string]string, len(body)+4)
	normalizedConfigValues := make(map[string]string, 3)

	if raw, ok := pickConfigInput(body, configKeyFrontendURLs, configKeyFrontendURL); ok {
		if h.urlResolver.cfg.FrontendURLEnvSet {
			response.Error(c, http.StatusBadRequest, "frontend_urls is controlled by env and cannot be changed")
			return
		}
		parsed, err := config.ParsePublicURLList(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid frontend_urls")
			return
		}
		if err := persistURLList(h.repo, configKeyFrontendURLs, configKeyFrontendURL, parsed); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
		details[configKeyFrontendURLs] = strings.Join(parsed, ",")
		details[configKeyFrontendURL] = firstOrEmptyString(parsed)
	}

	if raw, ok := pickConfigInput(body, configKeyBackendURLs, configKeyBackendURL); ok {
		if h.urlResolver.cfg.BackendURLEnvSet {
			response.Error(c, http.StatusBadRequest, "backend_urls is controlled by env and cannot be changed")
			return
		}
		parsed, err := config.ParsePublicURLList(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid backend_urls")
			return
		}
		if err := persistURLList(h.repo, configKeyBackendURLs, configKeyBackendURL, parsed); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
		details[configKeyBackendURLs] = strings.Join(parsed, ",")
		details[configKeyBackendURL] = firstOrEmptyString(parsed)
	}

	oidcState, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve oidc config", "error.failedToUpdateConfig")
		return
	}

	oidcChanged := false
	candidate := *oidcState

	if raw, ok := body[configKeyOIDCIssuer]; ok {
		trimmed := strings.TrimSpace(raw)
		if oidcState.IssuerEnvLocked {
			if trimmed != "" && trimmed != oidcState.Issuer {
				response.ErrorWithKey(c, http.StatusBadRequest, "oidc_issuer is controlled by env and cannot be changed", "error.oidcEnvLocked")
				return
			}
		} else {
			candidate.Issuer = trimmed
			oidcChanged = true
			details[configKeyOIDCIssuer] = trimmed
		}
	}

	if raw, ok := body[configKeyOIDCClientID]; ok {
		trimmed := strings.TrimSpace(raw)
		if oidcState.ClientIDEnvLocked {
			if trimmed != "" && trimmed != oidcState.ClientID {
				response.ErrorWithKey(c, http.StatusBadRequest, "oidc_client_id is controlled by env and cannot be changed", "error.oidcEnvLocked")
				return
			}
		} else {
			candidate.ClientID = trimmed
			oidcChanged = true
			details[configKeyOIDCClientID] = trimmed
		}
	}

	secretUpdated := false
	if raw, ok := body[configKeyOIDCClientSecret]; ok {
		trimmed := strings.TrimSpace(raw)
		if oidcState.ClientSecretEnvLocked {
			if trimmed != "" && trimmed != oidcState.ClientSecret {
				response.ErrorWithKey(c, http.StatusBadRequest, "oidc_client_secret is controlled by env and cannot be changed", "error.oidcEnvLocked")
				return
			}
		} else if trimmed != "" {
			candidate.ClientSecret = trimmed
			oidcChanged = true
			secretUpdated = true
			details[configKeyOIDCClientSecret] = "***"
		}
	}

	if oidcChanged {
		missing := collectOIDCMissingFields(&candidate)
		if len(missing) > 0 {
			response.ErrorWithKey(c, http.StatusBadRequest, "oidc config is incomplete", "error.oidcMissingFields")
			return
		}

		issuer, err := validateOIDCIssuer(candidate.Issuer)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid oidc issuer", "error.invalidOIDCIssuer")
			return
		}
		if _, err := oidc.Discover(c.Request.Context(), issuer); err != nil {
			response.ErrorWithKey(c, http.StatusBadGateway, "oidc discovery failed", "error.oidcDiscoveryFailed")
			return
		}

		if !oidcState.IssuerEnvLocked {
			if err := h.repo.SetSystemConfig(configKeyOIDCIssuer, issuer); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
				return
			}
			details[configKeyOIDCIssuer] = issuer
		}
		if !oidcState.ClientIDEnvLocked {
			if err := h.repo.SetSystemConfig(configKeyOIDCClientID, candidate.ClientID); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
				return
			}
		}
		if !oidcState.ClientSecretEnvLocked && secretUpdated {
			encSecret, err := crypto.EncryptIfKey(candidate.ClientSecret, h.cfg.EncryptionKey)
			if err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt oidc client secret", "error.encryptionFailed")
				return
			}
			if err := h.repo.SetSystemConfig(configKeyOIDCClientSecret, encSecret); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
				return
			}
		}
	}

	if raw, ok := body[configKeyDailyCheckinEnabled]; ok {
		normalized, err := normalizeBooleanConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_enabled", "error.invalidDailyCheckinEnabled")
			return
		}
		normalizedConfigValues[configKeyDailyCheckinEnabled] = normalized
		details[configKeyDailyCheckinEnabled] = normalized
	}

	if raw, ok := body["registration_bonus_credits"]; ok {
		normalized, err := normalizeNonNegativeCreditConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid registration_bonus_credits", "error.invalidCreditAmount")
			return
		}
		normalizedConfigValues["registration_bonus_credits"] = normalized
		details["registration_bonus_credits"] = normalized
	}

	if raw, ok := body["referral_inviter_credits"]; ok {
		normalized, err := normalizeNonNegativeCreditConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid referral_inviter_credits", "error.invalidCreditAmount")
			return
		}
		normalizedConfigValues["referral_inviter_credits"] = normalized
		details["referral_inviter_credits"] = normalized
	}

	if raw, ok := body["referral_invitee_credits"]; ok {
		normalized, err := normalizeNonNegativeCreditConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid referral_invitee_credits", "error.invalidCreditAmount")
			return
		}
		normalizedConfigValues["referral_invitee_credits"] = normalized
		details["referral_invitee_credits"] = normalized
	}

	if raw, ok := body[configKeyDailyCheckinCredits]; ok {
		normalized, err := normalizeDailyCheckinCredits(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_credits", "error.invalidDailyCheckinCredits")
			return
		}
		normalizedConfigValues[configKeyDailyCheckinCredits] = normalized
		details[configKeyDailyCheckinCredits] = normalized
	}

	if raw, ok := body[configKeyDailyCheckinGroupIDs]; ok {
		groupIDs, err := parseDailyCheckinGroupIDs(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_group_ids", "error.invalidDailyCheckinGroupIDs")
			return
		}
		if err := validateDailyCheckinGroupsExist(h.repo, groupIDs); err != nil {
			if errors.Is(err, errDailyCheckinGroupsNotFound) {
				response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_group_ids", "error.invalidDailyCheckinGroupIDs")
				return
			}
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to validate daily_checkin_group_ids", "error.failedToUpdateConfig")
			return
		}
		normalized := formatDailyCheckinGroupIDs(groupIDs)
		normalizedConfigValues[configKeyDailyCheckinGroupIDs] = normalized
		details[configKeyDailyCheckinGroupIDs] = normalized
	}

	for key, value := range body {
		if isURLConfigKey(key) || isOIDCConfigKey(key) {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if normalized, ok := normalizedConfigValues[key]; ok {
			trimmed = normalized
		}
		if err := h.repo.SetSystemConfig(key, trimmed); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
		details[key] = trimmed
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		detailBytes, _ := json.Marshal(details)
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_update_config",
			Resource: "system_config",
			Details:  detailBytes,
		})
	}
	response.OK(c, gin.H{"message": "config updated"})
}

func (h *AdminHandler) ConfirmURLConfig(c *gin.Context) {
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime url config", "error.failedToUpdateConfig")
		return
	}

	if err := h.repo.SetSystemConfig(configKeyURLConfirmedSignature, urlState.Signature); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save config confirmation", "error.failedToUpdateConfig")
		return
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{
			"frontend_urls":   urlState.FrontendURLs,
			"backend_urls":    urlState.BackendURLs,
			"frontend_url":    urlState.FrontendURL,
			"backend_url":     urlState.BackendURL,
			"frontend_source": urlState.FrontendSource,
			"backend_source":  urlState.BackendSource,
			"signature":       urlState.Signature,
		})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_confirm_runtime_url_config",
			Resource: "system_config",
			Details:  details,
		})
	}

	response.OK(c, gin.H{"message": "url config confirmed"})
}

func pickConfigInput(body map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			return value, true
		}
	}
	return "", false
}

func isURLConfigKey(key string) bool {
	switch key {
	case configKeyFrontendURL, configKeyFrontendURLs, configKeyBackendURL, configKeyBackendURLs:
		return true
	default:
		return false
	}
}

func isOIDCConfigKey(key string) bool {
	switch key {
	case configKeyOIDCIssuer, configKeyOIDCClientID, configKeyOIDCClientSecret:
		return true
	default:
		return false
	}
}

func firstOrEmptyString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
