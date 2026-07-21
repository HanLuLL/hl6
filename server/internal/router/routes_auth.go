package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerAuthRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	api.POST("/auth/registration/request", h.EmailAuth.RegistrationRequest)
	api.POST("/auth/activation/request", h.EmailAuth.ActivationRequest)
	api.POST("/auth/password/forgot", h.EmailAuth.ForgotPassword)
	api.GET("/auth/password/verify-token", h.EmailAuth.VerifyAuthToken)
	api.POST("/auth/password/complete", h.EmailAuth.CompletePassword)
	api.POST("/auth/login", h.EmailAuth.Login)

	authed := api.Group("", auth.Required())
	authed.GET("/auth/me", h.Auth.Me)
	authed.PUT("/auth/profile", h.Auth.UpdateProfile)
	authed.POST("/auth/logout", h.EmailAuth.Logout)
	// 设备管理
	authed.GET("/auth/sessions", h.Session.ListSessions)
	authed.DELETE("/auth/sessions/:id", h.Session.DeleteSession)
	authed.POST("/auth/logout-all", h.Session.LogoutAll)
}
