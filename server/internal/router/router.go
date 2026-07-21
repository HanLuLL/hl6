package router

import (
	"context"
	"log"
	"net/http"

	"hl6-server/internal/config"
	"hl6-server/internal/handler"
	"hl6-server/internal/logger"
	"hl6-server/internal/middleware"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/internal/worker"
	"hl6-server/pkg/queue"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type auditStack struct {
	enqueue  *service.AuditEnqueueService
	auditSvc *service.AuditService
	auditLog *service.AuditLogService
	notif    *service.NotificationService
	subSvc   *service.SubdomainService
}

func Setup(cfg *config.Config, db *gorm.DB, ctx context.Context, logSink *logger.DBSink) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins, cfg.FrontendURLs...))
	// access log 异步入库到 system_logs 表（module="http"），
	// gin.Default() 的默认 Logger 仍输出到 stdout，两者互不干扰。
	if logSink != nil {
		r.Use(logger.GinSinkMiddleware(logSink))
	}
	r.GET("/health", func(c *gin.Context) {
		status := "ok"
		dbOK := true
		if sqlDB, err := db.DB(); err == nil {
			if err := sqlDB.Ping(); err != nil {
				dbOK = false
				status = "degraded"
			}
		} else {
			dbOK = false
			status = "degraded"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   status,
			"database": dbOK,
		})
	})

	repo := repository.New(db)
	dnsOps := service.NewDNSOperationService(repo, cfg)
	migSvc := service.NewDomainMigrationService(repo, cfg)
	maintenanceGate := service.NewDatabaseMaintenanceGate()
	maintenanceSvc := service.NewDatabaseMaintenanceService(db, repo, cfg, maintenanceGate)

	sseBroker := handler.NewSSEBroker()
	audit := bootstrapAudit(ctx, cfg, db, repo, dnsOps, sseBroker)

	h := NewHandlers(cfg, repo, dnsOps, migSvc, maintenanceSvc, sseBroker, audit)

	trustedOrigins := append([]string{}, cfg.AllowedOrigins...)
	trustedOrigins = append(trustedOrigins, cfg.FrontendURLs...)
	auth := middleware.NewAuthMiddleware(cfg.SessionSecret, repo, trustedOrigins)

	api := r.Group("/api/v1")
	api.Use(h.Client.ValidatePresentedKey())
	api.Use(middleware.MaintenanceMode(maintenanceGate))

	api.GET("/branding", h.Branding.GetBranding)
	api.GET("/branding/logo.webp", h.Branding.GetLogo)
	api.GET("/branding/favicon.ico", h.Branding.GetFavicon)

	// SEO public endpoints
	r.GET("/robots.txt", h.SEO.RobotsTXT)
	r.GET("/sitemap.xml", h.SEO.SitemapXML)
	api.GET("/seo/meta", h.SEO.GetSEOMeta)
	api.GET("/client/version", h.Client.GetVersion)

	registerAuthRoutes(api, auth, h)
	registerDNSRoutes(api, auth, h)
	registerCreditRoutes(api, auth, h)
	registerNotificationRoutes(api, auth, h)
	registerAdminRoutes(api, auth, h)
	registerPaymentRoutes(api, auth, h)
	registerFriendLinkRoutes(api, auth, h)
	registerAIAuditRoutes(api, auth, h)
	registerEmailRoutes(api, auth, h)

	setupFrontendRoutes(r)

	return r
}

func bootstrapAudit(ctx context.Context, cfg *config.Config, db *gorm.DB, repo *repository.Repository, dnsOps *service.DNSOperationService, sseBroker *handler.SSEBroker) auditStack {
	taskQueue, enqueueDedup, schedDedup := initAuditQueue(ctx, cfg)

	ssePub := &handler.SSEEventPublisher{Broker: sseBroker}
	auditLogSvc := service.NewAuditLogService(repo)
	notifSvc := service.NewNotificationService(repo, ssePub)
	subSvc := service.NewSubdomainService(repo, dnsOps, auditLogSvc)
	auditSvc := service.NewAuditService(repo, dnsOps, subSvc, notifSvc, cfg.AuditScanTimeout, auditLogSvc)
	auditEnqueue := service.NewAuditEnqueueService(taskQueue, enqueueDedup)

	startAuditWorkers(ctx, cfg, db, repo, taskQueue, schedDedup, auditSvc, auditEnqueue)

	return auditStack{
		enqueue:  auditEnqueue,
		auditSvc: auditSvc,
		auditLog: auditLogSvc,
		notif:    notifSvc,
		subSvc:   subSvc,
	}
}

func initAuditQueue(ctx context.Context, cfg *config.Config) (queue.TaskQueue, service.AuditDedup, worker.ScheduleDedup) {
	if cfg.RedisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("WARN: REDIS_ADDR configured but ping failed (%v), falling back to in-process queue", err)
		} else {
			log.Printf("Audit queue: Redis at %s", cfg.RedisAddr)
			streams := queue.NewRedisStreams(rdb,
				queue.WithStreamName(queue.StreamAuditScanTasks),
				queue.WithConsumerGroup(queue.GroupAuditScanWorkers),
			)
			if err := streams.EnsureConsumerGroup(ctx); err != nil {
				log.Printf("WARN: audit redis consumer group: %v", err)
			}
			return queue.NewRedisTaskQueue(streams), service.NewRedisAuditDedup(rdb), worker.NewRedisScheduleDedup(rdb)
		}
	}

	log.Println("Audit queue: in-process channel (set REDIS_ADDR for multi-instance)")
	return queue.NewInProcQueue(0), service.NewInprocAuditDedup(), worker.NewInprocScheduleDedup()
}

func startAuditWorkers(ctx context.Context, cfg *config.Config, db *gorm.DB, repo *repository.Repository, taskQueue queue.TaskQueue, schedDedup worker.ScheduleDedup, auditSvc *service.AuditService, auditEnqueue *service.AuditEnqueueService) {
	if err := taskQueue.EnsureReady(ctx); err != nil {
		log.Printf("WARN: audit queue ensure ready: %v", err)
	}

	workerCount := cfg.AuditScanWorkerCount
	if workerCount <= 0 {
		workerCount = 2
	}
	for i := 0; i < workerCount; i++ {
		w := worker.NewAuditScanWorker(taskQueue, auditSvc, worker.AuditConsumerName(i))
		go w.Run(ctx)
	}

	sched := worker.NewAuditScheduler(db, taskQueue, schedDedup, cfg.AuditScanInterval)
	go func() {
		if err := sched.Run(ctx); err != nil {
			log.Printf("audit scheduler stopped: %v", err)
		}
	}()

	exemptW := worker.NewAuditExemptionWorker(repo, auditEnqueue)
	go func() {
		if err := exemptW.Run(ctx); err != nil {
			log.Printf("audit exemption worker stopped: %v", err)
		}
	}()
}
