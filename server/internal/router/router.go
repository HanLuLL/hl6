package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/handler"
	"hl6-server/internal/middleware"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

func Setup(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	repo := repository.New(db)
	cfTaskWorker := service.NewCloudflareTaskWorker(repo, cfg, 3*time.Second, 25)
	cfTaskWorker.Start(context.Background())

	auth := middleware.NewAuthMiddleware(cfg.SessionSecret, repo)
	rl := middleware.NewRateLimiter(100, time.Minute)

	authH := handler.NewAuthHandler(repo)
	oidcH := handler.NewOIDCHandler(repo, cfg)
	domainH := handler.NewDomainHandler(repo)
	sseBroker := handler.NewSSEBroker()
	subdomainH := handler.NewSubdomainHandler(repo, sseBroker)
	creditH := handler.NewCreditHandler(repo)
	adminH := handler.NewAdminHandler(repo, cfg)
	brandingH := handler.NewBrandingHandler(repo, cfg)
	referralH := handler.NewReferralHandler(repo)
	cfAccountH := handler.NewCloudflareAccountHandler(repo, cfg)

	dnsH := handler.NewDNSHandler(repo, sseBroker)
	notifH := handler.NewNotificationHandler(repo, sseBroker)
	notifAdminH := handler.NewNotificationAdminHandler(repo, sseBroker, cfg)

	api := r.Group("/api/v1")
	api.Use(rl.Handler())

	// Public auth routes
	api.GET("/auth/oidc/status", oidcH.Status)
	api.POST("/auth/oidc/bootstrap", oidcH.Bootstrap)
	api.GET("/auth/login", oidcH.Login)
	api.GET("/auth/callback", oidcH.Callback)
	api.GET("/branding", brandingH.GetBranding)
	api.GET("/branding/logo.webp", brandingH.GetLogo)
	api.GET("/branding/favicon.ico", brandingH.GetFavicon)

	// Authenticated routes
	authed := api.Group("", auth.Required())
	authed.GET("/auth/me", authH.Me)
	authed.POST("/auth/logout", oidcH.Logout)

	authed.GET("/domains", domainH.List)

	authed.GET("/subdomains", subdomainH.List)
	authed.GET("/subdomains/settings", subdomainH.Settings)
	authed.POST("/subdomains", subdomainH.Claim)
	authed.GET("/subdomains/:id", subdomainH.Get)
	authed.DELETE("/subdomains/:id", subdomainH.Release)

	authed.GET("/subdomains/:id/records", dnsH.ListRecords)
	authed.POST("/subdomains/:id/records", dnsH.CreateRecord)
	authed.PUT("/subdomains/:id/records/:recordId", dnsH.UpdateRecord)
	authed.DELETE("/subdomains/:id/records/:recordId", dnsH.DeleteRecord)

	authed.GET("/credits", creditH.GetBalance)
	authed.GET("/credits/transactions", creditH.ListTransactions)

	authed.GET("/referrals", referralH.GetReferralInfo)

	authed.GET("/notifications", notifH.List)
	authed.GET("/notifications/unread", notifH.UnreadStatus)
	authed.GET("/notifications/sse", notifH.SSE)
	authed.GET("/notifications/images/:id", notifH.GetImage)
	authed.GET("/notifications/:id", notifH.Get)
	authed.POST("/notifications/:id/read", notifH.MarkRead)

	admin := authed.Group("/admin")
	admin.Use(middleware.AdminRequired())
	admin.POST("/domains", domainH.AdminCreate)
	admin.GET("/domains/reserved-prefixes", domainH.AdminGetReservedPrefixes)
	admin.PUT("/domains/reserved-prefixes", domainH.AdminUpdateReservedPrefixes)
	admin.PUT("/domains/:id", domainH.AdminUpdate)
	admin.DELETE("/domains/:id", domainH.AdminDelete)
	admin.GET("/domains-full", domainH.AdminListDomainsFull)
	admin.GET("/cloudflare/accounts", cfAccountH.List)
	admin.POST("/cloudflare/accounts", cfAccountH.Create)
	admin.PUT("/cloudflare/accounts/:id", cfAccountH.Update)
	admin.DELETE("/cloudflare/accounts/:id", cfAccountH.Delete)
	admin.GET("/cloudflare/accounts/:id/zones", cfAccountH.ListZones)
	admin.POST("/credits/grant", creditH.AdminGrant)
	admin.GET("/dns-records", dnsH.AdminListRecords)
	admin.DELETE("/dns-records/:id", dnsH.AdminDeleteRecord)
	admin.GET("/claimed-subdomains", subdomainH.AdminListClaimed)
	admin.DELETE("/claimed-subdomains/:id", subdomainH.AdminRelease)
	admin.GET("/users", adminH.ListUsers)
	admin.PUT("/users/:id/group", adminH.UpdateUserGroup)
	admin.PUT("/users/:id/ban", adminH.BanUser)
	admin.PUT("/users/:id/unban", adminH.UnbanUser)
	admin.GET("/groups", adminH.ListGroups)
	admin.POST("/groups", adminH.CreateGroup)
	admin.PUT("/groups/:id", adminH.UpdateGroup)
	admin.DELETE("/groups/:id", adminH.DeleteGroup)
	admin.GET("/config", adminH.GetConfig)
	admin.PUT("/config", adminH.UpdateConfig)
	admin.POST("/config/url-confirm", adminH.ConfirmURLConfig)
	admin.GET("/stats", adminH.Stats)
	admin.GET("/cloudflare/tasks/dead", adminH.ListDeadCloudflareTasks)
	admin.GET("/audit-logs", adminH.AuditLogs)
	admin.GET("/notifications", notifAdminH.List)
	admin.POST("/notifications", notifAdminH.Create)
	admin.PUT("/notifications/:id", notifAdminH.Update)
	admin.DELETE("/notifications/:id", notifAdminH.Delete)
	admin.POST("/notifications/images", notifAdminH.UploadImage)
	admin.PUT("/branding", brandingH.AdminUpdateBranding)
	admin.POST("/branding/logo", brandingH.AdminUploadLogo)
	admin.DELETE("/branding/logo", brandingH.AdminDeleteLogo)

	setupFrontendRoutes(r)

	return r
}
