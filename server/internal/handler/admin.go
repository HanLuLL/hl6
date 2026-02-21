package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type AdminHandler struct {
	repo *repository.Repository
}

func NewAdminHandler(repo *repository.Repository) *AdminHandler {
	return &AdminHandler{repo: repo}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	users, total, err := h.repo.ListUsers(page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list users")
		return
	}
	response.Paginated(c, users, total, page, perPage)
}

func (h *AdminHandler) Stats(c *gin.Context) {
	stats, err := h.repo.GetStats()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get stats")
		return
	}
	response.OK(c, stats)
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	logs, total, err := h.repo.ListAuditLogs(page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list audit logs")
		return
	}
	response.Paginated(c, logs, total, page, perPage)
}

// User Group CRUD

func (h *AdminHandler) ListGroups(c *gin.Context) {
	groups, err := h.repo.ListUserGroups()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list groups")
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
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	group := &model.UserGroup{
		Name: body.Name,
	}

	if body.IsDefault {
		if err := h.repo.SetDefaultUserGroup(0); err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to update default group")
			return
		}
		group.IsDefault = true
	}

	if err := h.repo.CreateUserGroup(group); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create group")
		return
	}
	response.Created(c, group)
}

func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	group, err := h.repo.FindUserGroup(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "group not found")
		return
	}

	var body struct {
		Name      *string `json:"name"`
		IsDefault *bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name != nil {
		group.Name = *body.Name
	}
	if body.IsDefault != nil && *body.IsDefault {
		if err := h.repo.SetDefaultUserGroup(group.ID); err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to set default group")
			return
		}
		group.IsDefault = true
	}

	if err := h.repo.UpdateUserGroup(group); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update group")
		return
	}
	response.OK(c, group)
}

func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	migrateToStr := c.Query("migrate_to")
	if migrateToStr == "" {
		response.Error(c, http.StatusBadRequest, "migrate_to parameter is required")
		return
	}
	migrateTo, err := strconv.ParseUint(migrateToStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid migrate_to parameter")
		return
	}
	if uint(id) == uint(migrateTo) {
		response.Error(c, http.StatusBadRequest, "cannot migrate to the same group being deleted")
		return
	}

	count, _ := h.repo.CountUserGroups()
	if count <= 1 {
		response.Error(c, http.StatusBadRequest, "cannot delete the last group")
		return
	}

	group, err := h.repo.FindUserGroup(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "group not found")
		return
	}

	targetGroup, err := h.repo.FindUserGroup(uint(migrateTo))
	if err != nil {
		response.Error(c, http.StatusNotFound, "target group not found")
		return
	}

	tx := h.repo.DB.Begin()

	// Migrate users
	if err := tx.Model(&model.User{}).Where("group_id = ?", group.ID).Update("group_id", targetGroup.ID).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to migrate users")
		return
	}

	// Delete domain group accesses
	if err := h.repo.DeleteDomainGroupAccessByGroup(tx, group.ID); err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to delete group access records")
		return
	}

	// If deleted group was default, make target group the default
	if group.IsDefault {
		if err := tx.Model(&model.UserGroup{}).Where("id = ?", targetGroup.ID).Update("is_default", true).Error; err != nil {
			tx.Rollback()
			response.Error(c, http.StatusInternalServerError, "failed to update default group")
			return
		}
	}

	// Delete the group
	if err := tx.Delete(&model.UserGroup{}, group.ID).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to delete group")
		return
	}

	tx.Commit()
	response.OK(c, gin.H{"message": "group deleted and users migrated"})
}

// Update user's group
func (h *AdminHandler) UpdateUserGroup(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var body struct {
		GroupID uint `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Verify group exists
	if _, err := h.repo.FindUserGroup(body.GroupID); err != nil {
		response.Error(c, http.StatusNotFound, "group not found")
		return
	}

	if err := h.repo.UpdateUserGroupID(uint(userID), body.GroupID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update user group")
		return
	}
	response.OK(c, gin.H{"message": "user group updated"})
}

// System Config

func (h *AdminHandler) GetConfig(c *gin.Context) {
	configs, err := h.repo.GetAllSystemConfigs()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get config")
		return
	}
	response.OK(c, configs)
}

func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	for key, value := range body {
		if err := h.repo.SetSystemConfig(key, value); err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to update config")
			return
		}
	}
	response.OK(c, gin.H{"message": "config updated"})
}
