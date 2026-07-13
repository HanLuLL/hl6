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
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

var errCannotBanLastActiveAdmin = errors.New("cannot ban last active admin")

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
	admin := mustGetUser(c)
	if admin == nil {
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

		// 异步发送封禁通知邮件
		go func() {
			if h.emailSvc != nil {
				_ = h.emailSvc.SendBanNotification(target, reason)
			}
		}()

		return service.OperationResult{
			HTTPStatus: http.StatusOK,
			Message:    "ok",
			Data:       gin.H{"message": "user banned", "deleted_dns_count": result.DeletedDNSCount},
		}, nil
	})
	writeOperationResult(c, opResult)
}

func (h *AdminHandler) UnbanUser(c *gin.Context) {
	admin := mustGetUser(c)
	if admin == nil {
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
		IsAdmin   bool   `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	group := &model.UserGroup{
		Name:    body.Name,
		IsAdmin: body.IsAdmin,
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
		details, _ := json.Marshal(map[string]interface{}{"group_name": body.Name, "is_admin": body.IsAdmin})
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
		IsAdmin   *bool   `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	if body.Name != nil {
		group.Name = *body.Name
	}
	if body.IsAdmin != nil {
		group.IsAdmin = *body.IsAdmin
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
		details, _ := json.Marshal(map[string]interface{}{"group_name": group.Name, "is_admin": group.IsAdmin})
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
