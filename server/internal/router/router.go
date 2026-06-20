package router

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/handler"
	"hl6-server/internal/middleware"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/internal/worker"
	"hl6-server/pkg/queue"
)

type auditStack struct {
	enqueue  *service.AuditEnqueueService
	auditSvc *service.AuditService
	auditLog *service.AuditLogService
	notif    *service.NotificationService
	subSvc   *service.SubdomainService
}

func Setup(cfg *config.Config, db *gorm.DB, ctx context.Context) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	repo := repository.New(db)
	dnsOps := service.NewDNSOperationService(repo, cfg)
	migSvc := service.NewDomainMigrationService(repo, cfg)

	sseBroker := handler.NewSSEBroker()
	audit := bootstrapAudit(ctx, cfg, db, repo, dnsOps, sseBroker)

	h := NewHandlers(cfg, repo, dnsOps, migSvc, sseBroker, audit)

	auth := middleware.NewAuthMiddleware(cfg.SessionSecret, repo)

	api := r.Group("/api/v1")

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

func bootstrapAudit(ctx context.Context, cfg *config.Config, db *gorm.DB, repo *repository.Repository, dnsOps *service.DNSOperationService, sseBroker *handler.SSEBroker) auditStack {
	taskQueue, enqueueDedup, schedDedup := initAuditQueue(ctx, cfg)

	ssePub := &handler.SSEEventPublisher{Broker: sseBroker}
	auditLogSvc := service.NewAuditLogService(repo)
	notifSvc := service.NewNotificationService(repo, ssePub)
	subSvc := service.NewSubdomainService(repo, dnsOps, auditLogSvc)
	auditSvc := service.NewAuditService(repo, dnsOps, notifSvc, cfg.AuditScanTimeout, auditLogSvc)
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
