package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

// SystemLogHandler handles system log API endpoints.
type SystemLogHandler struct {
	repo *repository.Repository
}

// NewSystemLogHandler creates a new SystemLogHandler.
func NewSystemLogHandler(repo *repository.Repository) *SystemLogHandler {
	return &SystemLogHandler{repo: repo}
}

// ListSystemLogs lists system logs with pagination and filtering.
// GET /api/admin/logs
func (h *SystemLogHandler) ListSystemLogs(c *gin.Context) {
	page, perPage := helpers.ParsePageParams(c, 20, 100)

	filter := repository.SystemLogFilter{
		Levels:  queryStringSlice(c, "level"),
		Modules: queryStringSlice(c, "module"),
		Search:  strings.TrimSpace(c.Query("search")),
	}

	// Parse time range
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	logs, total, err := h.repo.ListSystemLogs(page, perPage, filter)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list system logs", "error.failedToListLogs")
		return
	}

	response.Paginated(c, logs, total, page, perPage)
}

// GetSystemLog returns a single system log by ID.
// GET /api/admin/logs/:id
func (h *SystemLogHandler) GetSystemLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid log id", "error.invalidLogId")
		return
	}

	log, err := h.repo.GetSystemLog(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "log not found", "error.logNotFound")
		return
	}

	response.OK(c, log)
}

// ExportSystemLogs exports system logs as JSON or text.
// GET /api/admin/logs/export
func (h *SystemLogHandler) ExportSystemLogs(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "txt" {
		format = "json"
	}

	// Parse filters (same as list)
	filter := repository.SystemLogFilter{
		Levels:  queryStringSlice(c, "level"),
		Modules: queryStringSlice(c, "module"),
		Search:  strings.TrimSpace(c.Query("search")),
	}

	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	// Get all logs (up to 10000 for export)
	logs, _, err := h.repo.ListSystemLogs(1, 10000, filter)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to export logs", "error.failedToExportLogs")
		return
	}

	// Generate filename
	filename := fmt.Sprintf("system_logs_%s", time.Now().Format("20060102_150405"))

	if format == "txt" {
		h.exportAsText(c, logs, filename)
		return
	}

	h.exportAsJSON(c, logs, filename)
}

func (h *SystemLogHandler) exportAsJSON(c *gin.Context, logs []model.SystemLog, filename string) {
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to marshal logs", "error.failedToExportLogs")
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))
	c.Data(http.StatusOK, "application/json", data)
}

func (h *SystemLogHandler) exportAsText(c *gin.Context, logs []model.SystemLog, filename string) {
	var sb strings.Builder
	sb.WriteString("System Logs Export\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total Entries: %d\n", len(logs)))
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, log := range logs {
		sb.WriteString(fmt.Sprintf("[%s] [%s] %s\n", log.CreatedAt.Format(time.RFC3339), log.Level, log.Module))
		sb.WriteString(fmt.Sprintf("Message: %s\n", log.Message))
		if log.Fields != nil && len(log.Fields) > 2 {
			var fields map[string]interface{}
			if err := json.Unmarshal(log.Fields, &fields); err == nil && len(fields) > 0 {
				sb.WriteString("Fields:\n")
				for k, v := range fields {
					sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
				}
			}
		}
		sb.WriteString(strings.Repeat("-", 40) + "\n")
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", filename))
	c.String(http.StatusOK, sb.String())
}

// GetSystemLogModules returns distinct module names.
// GET /api/admin/logs/modules
func (h *SystemLogHandler) GetSystemLogModules(c *gin.Context) {
	modules, err := h.repo.GetSystemLogModules()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get modules", "error.internal")
		return
	}
	response.OK(c, modules)
}

// GetSystemLogStats returns log statistics.
// GET /api/admin/logs/stats
func (h *SystemLogHandler) GetSystemLogStats(c *gin.Context) {
	stats, err := h.repo.GetSystemLogStats()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get stats", "error.internal")
		return
	}
	response.OK(c, stats)
}