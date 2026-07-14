package router

import (
	"hl6-server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerEmailRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	admin := api.Group("", auth.Required(), middleware.AdminRequired())

	// 管理员邮件日志
	admin.GET("/admin/emails", h.Email.ListEmailLogs)
	admin.POST("/admin/emails/:id/retry", h.Email.RetryEmail)
	admin.POST("/admin/emails/test", h.Email.TestSMTPConfig)
}
