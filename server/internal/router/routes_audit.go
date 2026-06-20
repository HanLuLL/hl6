package router

import "github.com/gin-gonic/gin"

func registerAdminAuditRoutes(admin *gin.RouterGroup, h *Handlers) {
	audit := admin.Group("/audit")
	audit.GET("/summary", h.Audit.GetSummary)
	audit.GET("/cases", h.Audit.ListCases)
	audit.GET("/subdomains/:id", h.Audit.GetSubdomainDetail)
	audit.GET("/subdomains/:id/scans", h.Audit.ListSubdomainScans)
	audit.POST("/subdomains/:id/restore", h.Audit.RestoreSubdomain)
	audit.DELETE("/subdomains/:id/release", h.Audit.ReleaseSubdomain)
	audit.POST("/subdomains/:id/rescan", h.Audit.RescanSubdomain)
	audit.POST("/subdomains/bulk-rescan", h.Audit.BulkRescan)

	rules := audit.Group("/rules")
	rules.GET("", h.Audit.ListRules)
	rules.POST("", h.Audit.CreateRule)
	rules.PUT("/:id", h.Audit.UpdateRule)
	rules.DELETE("/:id", h.Audit.DeleteRule)
	rules.PUT("/:id/toggle", h.Audit.ToggleRule)
	rules.GET("/scenarios", h.Audit.ListScenarios)
	rules.POST("/test", h.Audit.TestRule)

	audit.GET("/scans", h.Audit.ListScans)
	audit.GET("/scans/:id", h.Audit.GetScan)
}
