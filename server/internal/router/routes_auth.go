package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerAuthRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	api.GET("/auth/oidc/status", h.OIDC.Status)
	api.POST("/auth/oidc/bootstrap", h.OIDC.Bootstrap)
	api.GET("/auth/login", h.OIDC.Login)
	api.GET("/auth/callback", h.OIDC.Callback)

	authed := api.Group("", auth.Required())
	authed.GET("/auth/me", h.Auth.Me)
	authed.POST("/auth/logout", h.OIDC.Logout)
}
