package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/middleware"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

func Setup(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	repo := repository.New(db)
	dnsOps := service.NewDNSOperationService(repo, cfg)
	migSvc := service.NewDomainMigrationService(repo, cfg)
	h := NewHandlers(cfg, repo, dnsOps, migSvc)

	auth := middleware.NewAuthMiddleware(cfg.SessionSecret, repo)
	rl := middleware.NewRateLimiter(100, time.Minute)

	api := r.Group("/api/v1")
	api.Use(rl.Handler())

	// Public branding routes
	api.GET("/branding", h.Branding.GetBranding)
	api.GET("/branding/logo.webp", h.Branding.GetLogo)
	api.GET("/branding/favicon.ico", h.Branding.GetFavicon)

	registerAuthRoutes(api, auth, h)
	registerDNSRoutes(api, auth, h)
	registerCreditRoutes(api, auth, h)
	registerNotificationRoutes(api, auth, h)
	registerAdminRoutes(api, auth, h)

	setupFrontendRoutes(r)

	return r
}
