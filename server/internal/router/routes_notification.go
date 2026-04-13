package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerNotificationRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	authed := api.Group("", auth.Required())
	authed.GET("/notifications", h.Notification.List)
	authed.GET("/notifications/unread", h.Notification.UnreadStatus)
	authed.GET("/notifications/sse", h.Notification.SSE)
	authed.GET("/notifications/images/:id", h.Notification.GetImage)
	authed.GET("/notifications/:id", h.Notification.Get)
	authed.POST("/notifications/:id/read", h.Notification.MarkRead)
}
