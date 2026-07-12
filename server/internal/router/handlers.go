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
	OIDC              *handler.OIDCHandler
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
}

func NewHandlers(cfg *config.Config, repo *repository.Repository, dnsOps *service.DNSOperationService, migSvc *service.DomainMigrationService, sseBroker *handler.SSEBroker, audit auditStack) *Handlers {
	// Initialize payment services
	var epaySvc *service.EpayService
	var codepaySvc *service.CodePayService

	backendURL := cfg.BackendURL
	if backendURL == "" {
		backendURL = "http://localhost:8081"
	}

	if cfg.EpayURL != "" && cfg.EpayPID != "" && cfg.EpayKey != "" {
		epaySvc = service.NewEpayService(cfg.EpayURL, cfg.EpayPID, cfg.EpayKey, backendURL+"/api/v1/payment/epay/notify", backendURL+"/api/v1/payment/return")
	}
	if cfg.CodePayURL != "" && cfg.CodePayID != "" && cfg.CodePayKey != "" {
		codepaySvc = service.NewCodePayService(cfg.CodePayURL, cfg.CodePayID, cfg.CodePayKey, backendURL+"/api/v1/payment/codepay/notify", backendURL+"/api/v1/payment/return")
	}

	return &Handlers{
		Auth:              handler.NewAuthHandler(repo),
		OIDC:              handler.NewOIDCHandler(repo, cfg),
		Domain:            handler.NewDomainHandler(repo, dnsOps),
		Subdomain:         handler.NewSubdomainHandler(repo, sseBroker, dnsOps, audit.enqueue, audit.notif, audit.subSvc, audit.auditLog),
		DNS:               handler.NewDNSHandler(repo, sseBroker, dnsOps, audit.enqueue),
		Credit:            handler.NewCreditHandler(repo),
		Admin:             handler.NewAdminHandler(repo, cfg, dnsOps),
		Branding:          handler.NewBrandingHandler(repo, cfg),
		Referral:          handler.NewReferralHandler(repo),
		DNSAccount:        handler.NewDNSProviderAccountHandler(repo, cfg, dnsOps),
		Migration:         handler.NewDomainMigrationHandler(migSvc),
		Notification:      handler.NewNotificationHandler(repo, sseBroker),
		NotificationAdmin: handler.NewNotificationAdminHandler(repo, sseBroker, cfg),
		Audit:             handler.NewAuditHandler(repo, audit.auditSvc, audit.subSvc, dnsOps, audit.enqueue, audit.notif, audit.auditLog),
		SSEBroker:         sseBroker,
		Payment:           handler.NewPaymentHandler(repo, epaySvc, codepaySvc),
		SEO:               handler.NewSEOHandler(repo),
	}
}
