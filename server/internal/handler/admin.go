package handler

import (
	"hl6-server/internal/config"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

const adminBanGuardLockKey int64 = 19490332

type AdminHandler struct {
	repo         *repository.Repository
	cfg          *config.Config
	ops          *service.DNSOperationService
	emailSvc     *service.EmailService
	urlResolver  *URLResolver
}

func NewAdminHandler(repo *repository.Repository, cfg *config.Config, ops *service.DNSOperationService, emailSvc *service.EmailService) *AdminHandler {
	return &AdminHandler{
		repo:         repo,
		cfg:          cfg,
		ops:          ops,
		emailSvc:     emailSvc,
		urlResolver:  NewURLResolver(repo, cfg),
	}
}
