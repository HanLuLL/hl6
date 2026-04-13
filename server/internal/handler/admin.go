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
	urlResolver  *URLResolver
	oidcResolver *OIDCRuntimeResolver
}

func NewAdminHandler(repo *repository.Repository, cfg *config.Config, ops *service.DNSOperationService) *AdminHandler {
	return &AdminHandler{
		repo:         repo,
		cfg:          cfg,
		ops:          ops,
		urlResolver:  NewURLResolver(repo, cfg),
		oidcResolver: NewOIDCRuntimeResolver(repo, cfg),
	}
}
