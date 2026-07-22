package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerRealnameRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	// 用户接口（认证）
	authed := api.Group("", auth.Required())
	authed.POST("/realname/apply", h.Realname.SubmitApplication)
	authed.GET("/realname/status", h.Realname.GetStatus)
	authed.GET("/realname/history", h.Realname.GetHistory)
	authed.POST("/realname/retry", h.Realname.RetryVerification)

	// 管理员接口
	admin := api.Group("/admin", auth.Required(), middleware.AdminRequired())
	admin.GET("/realname/applications", h.Realname.AdminListApplications)
	admin.GET("/realname/applications/:id", h.Realname.AdminGetApplication)
	// 管理员按需查看明文实名（POST：需请求体携带 reason，强制审计留痕）
	admin.POST("/realname/applications/:id/full", h.Realname.AdminGetApplicationFull)
	admin.PUT("/realname/applications/:id/review", h.Realname.AdminReview)
	admin.POST("/realname/applications/:id/retry", h.Realname.AdminRetryVerification)
	admin.GET("/realname/stats", h.Realname.AdminGetStats)
	// 管理员直接修改用户的实名状态（不依赖申请单）
	admin.PUT("/users/:id/realname", h.Realname.AdminUpdateUserRealname)
}
