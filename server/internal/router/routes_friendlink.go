package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerFriendLinkRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	// Public endpoint - 公开友链列表
	api.GET("/friend-links", h.FriendLink.GetPublicFriendLinks)

	// Admin endpoints - 后台管理
	admin := api.Group("/admin", auth.Required(), middleware.AdminRequired())
	admin.GET("/friend-links", h.FriendLink.AdminList)
	admin.POST("/friend-links", h.FriendLink.AdminCreate)
	admin.PUT("/friend-links/:id", h.FriendLink.AdminUpdate)
	admin.DELETE("/friend-links/:id", h.FriendLink.AdminDelete)
}
