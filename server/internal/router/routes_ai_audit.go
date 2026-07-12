package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerAIAuditRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	// AI 审查统计（公开，需认证）
	api.GET("/ai-audit/stats", auth.Required(), h.AIAudit.GetAIStats)

	// AI 审查记录（管理员）
	adminAI := api.Group("/admin/ai-audit", auth.Required(), middleware.AdminRequired())
	adminAI.GET("/reviews", h.AIAudit.ListAIReviews)
	adminAI.GET("/reviews/:id", h.AIAudit.GetAIReview)
	adminAI.PUT("/reviews/:id", h.AIAudit.ReviewAIReview)

	// AI 模型配置（管理员）
	adminAI.GET("/models", h.AIAudit.ListModelConfigs)
	adminAI.POST("/models", h.AIAudit.CreateModelConfig)
	adminAI.PUT("/models/:id", h.AIAudit.UpdateModelConfig)
	adminAI.DELETE("/models/:id", h.AIAudit.DeleteModelConfig)

	// 提示词模板（管理员）
	adminAI.GET("/prompt-templates", h.AIAudit.ListPromptTemplates)
	adminAI.POST("/prompt-templates", h.AIAudit.CreatePromptTemplate)
	adminAI.PUT("/prompt-templates/:id", h.AIAudit.UpdatePromptTemplate)
	adminAI.DELETE("/prompt-templates/:id", h.AIAudit.DeletePromptTemplate)

	// 申诉管理（管理员）
	adminAI.GET("/appeals", h.AIAudit.AdminListAppeals)
	adminAI.PUT("/appeals/:id", h.AIAudit.AdminReviewAppeal)

	// 用户端接口（需认证）
	userAPI := api.Group("", auth.Required())
	userAPI.GET("/ban-info", h.AIAudit.GetBanInfo)
	userAPI.POST("/appeals", h.AIAudit.CreateUserAppeal)
	userAPI.GET("/appeals", h.AIAudit.ListMyAppeals)
}
