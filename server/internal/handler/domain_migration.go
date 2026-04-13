package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

// DomainMigrationHandler handles DNS migration task endpoints.
type DomainMigrationHandler struct {
	migSvc *service.DomainMigrationService
}

func NewDomainMigrationHandler(migSvc *service.DomainMigrationService) *DomainMigrationHandler {
	return &DomainMigrationHandler{migSvc: migSvc}
}

// Create creates a new migration task for a domain.
// POST /api/v1/admin/domains/:id/migrations
func (h *DomainMigrationHandler) Create(c *gin.Context) {
	domainID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}

	var body struct {
		TargetProviderAccountID uint   `json:"target_provider_account_id" binding:"required"`
		TargetProviderZoneID    string `json:"target_provider_zone_id" binding:"required"`
		Reason                  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	result, err := h.migSvc.CreateMigration(c.Request.Context(), service.CreateMigrationInput{
		DomainID:                domainID,
		TriggeredBy:             user.ID,
		TargetProviderAccountID: body.TargetProviderAccountID,
		TargetProviderZoneID:    body.TargetProviderZoneID,
		Reason:                  strings.TrimSpace(body.Reason),
	})
	if err != nil {
		var blocked *service.MigrationTaskBlockedError
		var pe *service.ProviderError
		switch {
		case errors.As(err, &blocked):
			response.ErrorWithKeyData(c, http.StatusConflict, err.Error(), "error.migrationTaskBlocked", gin.H{
				"blocking_task_id": blocked.TaskID,
				"blocking_status":  blocked.TaskStatus,
			})
		case errors.As(err, &pe):
			response.Error(c, http.StatusBadRequest, err.Error())
		case isClientError(err):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	c.JSON(http.StatusCreated, response.Response{Code: 0, Message: "created", Data: result})
}

// List lists migration tasks for a domain.
// GET /api/v1/admin/domains/:id/migrations
func (h *DomainMigrationHandler) List(c *gin.Context) {
	domainID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	tasks, total, err := h.migSvc.Repo().ListMigrationTasks(domainID, status, page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list migrations", "error.databaseError")
		return
	}

	response.OK(c, gin.H{
		"tasks":    tasks,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// Get returns detail of a migration task including its items.
// GET /api/v1/admin/domains/:id/migrations/:taskId
func (h *DomainMigrationHandler) Get(c *gin.Context) {
	taskID, ok := helpers.ParseUintParam(c, "taskId")
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))

	task, err := h.migSvc.Repo().FindMigrationTask(taskID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "migration task not found", "error.notFound")
		return
	}

	items, itemTotal, err := h.migSvc.Repo().ListMigrationItems(taskID, page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list migration items", "error.databaseError")
		return
	}

	// Clear preloaded items; return via separate paginated field
	task.Items = nil
	response.OK(c, gin.H{
		"task":       task,
		"items":      items,
		"item_total": itemTotal,
		"page":       page,
		"per_page":   perPage,
	})
}

// RetryFailures retries all failed items of a migration task.
// POST /api/v1/admin/domains/:id/migrations/:taskId/retry-failures
func (h *DomainMigrationHandler) RetryFailures(c *gin.Context) {
	taskID, ok := helpers.ParseUintParam(c, "taskId")
	if !ok {
		return
	}

	result, err := h.migSvc.RetryFailures(c.Request.Context(), taskID)
	if err != nil {
		var pe *service.ProviderError
		if errors.As(err, &pe) && pe.Category == service.ErrCategoryInvalidRequest {
			response.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}

// CleanupSource deletes all platform-managed source DNS records after migration.
// POST /api/v1/admin/domains/:id/migrations/:taskId/cleanup-source
func (h *DomainMigrationHandler) CleanupSource(c *gin.Context) {
	taskID, ok := helpers.ParseUintParam(c, "taskId")
	if !ok {
		return
	}

	var body struct {
		ConfirmDomainName string `json:"confirm_domain_name" binding:"required"`
		ConfirmPhrase     string `json:"confirm_phrase" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	result, err := h.migSvc.CleanupSource(c.Request.Context(), taskID, body.ConfirmDomainName, body.ConfirmPhrase)
	if err != nil {
		var pe *service.ProviderError
		if errors.As(err, &pe) && pe.Category == service.ErrCategoryInvalidRequest {
			response.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.OK(c, result)
}
