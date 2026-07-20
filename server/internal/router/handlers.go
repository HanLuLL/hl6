package router

import (
	"hl6-server/internal/config"
	"hl6-server/internal/handler"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

// Handlers 集中管理所有 handler 的初始化，作为依赖注入的单一入口。
type Handlers struct {
	Auth              *handler.AuthHandler
	EmailAuth         *handler.EmailAuthHandler
	Domain            *handler.DomainHandler
	Subdomain         *handler.SubdomainHandler
	DNS               *handler.DNSHandler
	Credit            *handler.CreditHandler
	Admin             *handler.AdminHandler
	Branding          *handler.BrandingHandler
	Referral          *handler.ReferralHandler
	DNSAccount        *handler.DNSProviderAccountHandler
	Migration         *handler.DomainMigrationHandler
	Notification      *handler.NotificationHandler
	NotificationAdmin *handler.NotificationAdminHandler
	Audit             *handler.AuditHandler
	SSEBroker         *handler.SSEBroker
	Payment           *handler.PaymentHandler
	SEO               *handler.SEOHandler
	FriendLink        *handler.FriendLinkHandler
	AIAudit           *handler.AIAuditHandler
	Email             *handler.EmailHandler
	Client            *handler.ClientHandler
	Maintenance       *handler.MaintenanceHandler
	SystemLog         *handler.SystemLogHandler
}

func NewHandlers(cfg *config.Config, repo *repository.Repository, dnsOps *service.DNSOperationService, migSvc *service.DomainMigrationService, maintenanceSvc *service.DatabaseMaintenanceService, sseBroker *handler.SSEBroker, audit auditStack) *Handlers {
	emailSvc := service.NewEmailService(repo, cfg.EncryptionKey)

	return &Handlers{
		Auth:              handler.NewAuthHandler(repo),
		EmailAuth:         handler.NewEmailAuthHandler(repo, emailSvc, cfg),
		Domain:            handler.NewDomainHandler(repo, dnsOps),
		Subdomain:         handler.NewSubdomainHandler(repo, sseBroker, dnsOps, audit.enqueue, audit.notif, audit.subSvc, audit.auditLog),
		DNS:               handler.NewDNSHandler(repo, sseBroker, dnsOps, audit.enqueue, emailSvc),
		Credit:            handler.NewCreditHandler(repo),
		Admin:             handler.NewAdminHandler(repo, cfg, dnsOps, emailSvc),
		Branding:          handler.NewBrandingHandler(repo, cfg),
		Referral:          handler.NewReferralHandler(repo),
		DNSAccount:        handler.NewDNSProviderAccountHandler(repo, cfg, dnsOps),
		Migration:         handler.NewDomainMigrationHandler(migSvc),
		Notification:      handler.NewNotificationHandler(repo, sseBroker),
		NotificationAdmin: handler.NewNotificationAdminHandler(repo, sseBroker, cfg),
		Audit:             handler.NewAuditHandler(repo, audit.auditSvc, audit.subSvc, dnsOps, audit.enqueue, audit.notif, audit.auditLog),
		SSEBroker:         sseBroker,
		Payment:           handler.NewPaymentHandler(repo, cfg),
		SEO:               handler.NewSEOHandler(repo),
		FriendLink:        handler.NewFriendLinkHandler(repo),
		AIAudit:           handler.NewAIAuditHandler(repo, cfg.EncryptionKey),
		Email:             handler.NewEmailHandler(repo, emailSvc),
		Client:            handler.NewClientHandler(repo),
		Maintenance:       handler.NewMaintenanceHandler(repo, cfg, maintenanceSvc),
		SystemLog:         handler.NewSystemLogHandler(repo),
	}
}
