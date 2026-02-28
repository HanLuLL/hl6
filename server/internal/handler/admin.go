package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/pkg/response"
)

var allowedConfigKeys = map[string]bool{
	"registration_bonus_credits": true,
	"referral_enabled":           true,
	"referral_inviter_credits":   true,
	"referral_invitee_credits":   true,
}

type AdminHandler struct {
	repo *repository.Repository
}

func NewAdminHandler(repo *repository.Repository) *AdminHandler {
	return &AdminHandler{repo: repo}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	users, total, err := h.repo.ListUsers(page, perPage, search)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list users", "error.failedToListUsers")
		return
	}

	// Batch fetch referral inviters
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	inviterMap, _ := h.repo.GetReferralInvitersForUsers(userIDs)

	type userDTO struct {
		model.User
		InvitedBy *struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"invited_by"`
	}

	result := make([]userDTO, len(users))
	for i, u := range users {
		dto := userDTO{User: u}
		if inviter, ok := inviterMap[u.ID]; ok {
			dto.InvitedBy = &struct {
				ID   uint   `json:"id"`
				Name string `json:"name"`
			}{ID: inviter.ID, Name: inviter.Name}
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

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "15"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}

	logs, total, err := h.repo.ListAuditLogs(page, perPage, search)
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

	count, _ := h.repo.CountUserGroups()
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

	tx := h.repo.GetDB().Begin()

	// Migrate users
	if err := tx.Model(&model.User{}).Where("group_id = ?", group.ID).Update("group_id", targetGroup.ID).Error; err != nil {
		tx.Rollback()
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to migrate users", "error.failedToMigrateUsers")
		return
	}

	// Delete domain group accesses
	if err := h.repo.DeleteDomainGroupAccessByGroup(tx, group.ID); err != nil {
		tx.Rollback()
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete group access records", "error.failedToDeleteGroupAccess")
		return
	}

	// If deleted group was default, make target group the default
	if group.IsDefault {
		if err := tx.Model(&model.UserGroup{}).Where("id = ?", targetGroup.ID).Update("is_default", true).Error; err != nil {
			tx.Rollback()
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update default group", "error.failedToUpdateDefaultGroup")
			return
		}
	}

	// Delete the group
	if err := tx.Delete(&model.UserGroup{}, group.ID).Error; err != nil {
		tx.Rollback()
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete group", "error.failedToDeleteGroup")
		return
	}

	tx.Commit()
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

// System Config

func (h *AdminHandler) GetConfig(c *gin.Context) {
	configs, err := h.repo.GetAllSystemConfigs()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get config", "error.failedToGetConfig")
		return
	}
	response.OK(c, configs)
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

	for key, value := range body {
		if err := h.repo.SetSystemConfig(key, value); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
	}
	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(body)
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_update_config",
			Resource: "system_config",
			Details:  details,
		})
	}
	response.OK(c, gin.H{"message": "config updated"})
}
