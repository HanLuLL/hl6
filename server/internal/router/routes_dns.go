package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerDNSRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	authed := api.Group("", auth.Required())

	authed.GET("/domains", h.Domain.List)

	authed.GET("/subdomains", h.Subdomain.List)
	authed.GET("/subdomains/settings", h.Subdomain.Settings)
	authed.POST("/subdomains", h.Subdomain.Claim)
	authed.GET("/subdomains/:id", h.Subdomain.Get)
	authed.DELETE("/subdomains/:id", h.Subdomain.Release)

	authed.GET("/subdomains/:id/records", h.DNS.ListRecords)
	authed.POST("/subdomains/:id/records", h.DNS.CreateRecord)
	authed.PUT("/subdomains/:id/records/:recordId", h.DNS.UpdateRecord)
	authed.DELETE("/subdomains/:id/records/:recordId", h.DNS.DeleteRecord)
}
