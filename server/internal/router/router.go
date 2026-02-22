package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/handler"
	"hl6-server/internal/middleware"
	"hl6-server/internal/repository"
)

func Setup(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	repo := repository.New(db)
	auth := middleware.NewAuthMiddleware(cfg.SessionSecret)
	rl := middleware.NewRateLimiter(100, time.Minute)

	authH := handler.NewAuthHandler(repo)
	oidcH := handler.NewOIDCHandler(repo, cfg)
	domainH := handler.NewDomainHandler(repo)
	subdomainH := handler.NewSubdomainHandler(repo)
	dnsH := handler.NewDNSHandler(repo)
	creditH := handler.NewCreditHandler(repo)
	adminH := handler.NewAdminHandler(repo)
	cfAccountH := handler.NewCloudflareAccountHandler(repo)

	api := r.Group("/api/v1")
	api.Use(rl.Handler())

	// Public auth routes
	api.GET("/auth/login", oidcH.Login)
	api.GET("/auth/callback", oidcH.Callback)

	// Authenticated routes
	authed := api.Group("", auth.Required())
	authed.GET("/auth/me", authH.Me)
	authed.POST("/auth/logout", oidcH.Logout)

	authed.GET("/domains", domainH.List)

	authed.GET("/subdomains", subdomainH.List)
	authed.POST("/subdomains", subdomainH.Claim)
	authed.GET("/subdomains/:id", subdomainH.Get)
	authed.DELETE("/subdomains/:id", subdomainH.Release)

	authed.GET("/subdomains/:id/records", dnsH.ListRecords)
	authed.POST("/subdomains/:id/records", dnsH.CreateRecord)
	authed.PUT("/subdomains/:id/records/:recordId", dnsH.UpdateRecord)
	authed.DELETE("/subdomains/:id/records/:recordId", dnsH.DeleteRecord)

	authed.GET("/credits", creditH.GetBalance)
	authed.GET("/credits/transactions", creditH.ListTransactions)

	admin := authed.Group("/admin")
	admin.Use(middleware.AdminRequired(db))
	admin.POST("/domains", domainH.AdminCreate)
	admin.PUT("/domains/:id", domainH.AdminUpdate)
	admin.DELETE("/domains/:id", domainH.AdminDelete)
	admin.GET("/domains-full", domainH.AdminListDomainsFull)
	admin.GET("/cloudflare/accounts", cfAccountH.List)
	admin.POST("/cloudflare/accounts", cfAccountH.Create)
	admin.PUT("/cloudflare/accounts/:id", cfAccountH.Update)
	admin.DELETE("/cloudflare/accounts/:id", cfAccountH.Delete)
	admin.GET("/cloudflare/accounts/:id/zones", cfAccountH.ListZones)
	admin.POST("/credits/grant", creditH.AdminGrant)
	admin.GET("/users", adminH.ListUsers)
	admin.PUT("/users/:id/group", adminH.UpdateUserGroup)
	admin.GET("/groups", adminH.ListGroups)
	admin.POST("/groups", adminH.CreateGroup)
	admin.PUT("/groups/:id", adminH.UpdateGroup)
	admin.DELETE("/groups/:id", adminH.DeleteGroup)
	admin.GET("/config", adminH.GetConfig)
	admin.PUT("/config", adminH.UpdateConfig)
	admin.GET("/stats", adminH.Stats)
	admin.GET("/audit-logs", adminH.AuditLogs)

	return r
}
