package handler

import (
	"net/http"
	"strconv"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// EmailHandler 邮件管理 handler。
type EmailHandler struct {
	repo     *repository.Repository
	emailSvc *service.EmailService
}

// NewEmailHandler 创建邮件管理 handler。
func NewEmailHandler(repo *repository.Repository, emailSvc *service.EmailService) *EmailHandler {
	return &EmailHandler{repo: repo, emailSvc: emailSvc}
}

// ListEmailLogs 管理员查看邮件发送记录。
func (h *EmailHandler) ListEmailLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	emailType := c.Query("email_type")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	logs, total, err := h.repo.ListEmailLogs(page, perPage, emailType, status)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list email logs", "error.internal")
		return
	}

	response.Paginated(c, logs, total, page, perPage)
}

// RetryEmail 管理员手动重试发送失败的邮件。
func (h *EmailHandler) RetryEmail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "error.invalidRequestBody")
		return
	}

	emailLog, err := h.repo.FindEmailLog(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "email log not found", "error.notFound")
		return
	}

	if emailLog.Status != model.EmailStatusFailed {
		response.ErrorWithKey(c, http.StatusBadRequest, "only failed emails can be retried", "error.invalidRequestBody")
		return
	}

	if err := h.emailSvc.RetrySingleEmail(emailLog); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to retry email", "error.internal")
		return
	}

	auditLogIfAdmin(nil, c, "admin_retry_email", "email_log", uint(id), nil)
	response.OK(c, gin.H{"retried": true})
}

// TestSMTPConfig 测试 SMTP 配置是否可用。
func (h *EmailHandler) TestSMTPConfig(c *gin.Context) {
	admin, ok := requireUser(c)
	if !ok {
		return
	}

	if admin.Email == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "admin user has no email address", "error.noEmail")
		return
	}

	siteName := "SubDomain"
	if name, err := h.repo.GetSystemConfig("brand_name"); err == nil && name != "" {
		siteName = name
	}

	if err := h.emailSvc.SendTestEmail(admin.Email, siteName); err != nil {
		auditLogIfAdmin(nil, c, "admin_test_smtp_failed", "system_config", 0, map[string]any{"error": err.Error()})
		response.ErrorWithKey(c, http.StatusInternalServerError, "SMTP test failed: "+err.Error(), "error.smtpTestFailed")
		return
	}

	auditLogIfAdmin(nil, c, "admin_test_smtp", "system_config", 0, nil)
	response.OK(c, gin.H{"sent": true, "recipient": admin.Email})
}
