package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/config"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/crypto"
	"hl6-server/pkg/response"
)

type DNSProviderAccountHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewDNSProviderAccountHandler(repo *repository.Repository, cfg *config.Config) *DNSProviderAccountHandler {
	return &DNSProviderAccountHandler{repo: repo, cfg: cfg}
}

func (h *DNSProviderAccountHandler) encryptRawCredentials(raw string) (string, error) {
	return crypto.EncryptIfKey(raw, h.cfg.EncryptionKey)
}

func (h *DNSProviderAccountHandler) decryptRawCredentials(raw string) string {
	return crypto.DecryptOrPlaintext(raw, h.cfg.EncryptionKey)
}

func (h *DNSProviderAccountHandler) List(c *gin.Context) {
	accounts, err := h.repo.ListDNSProviderAccounts()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list accounts", "error.failedToListDNSProviderAccounts")
		return
	}

	views := make([]model.DNSProviderAccountView, 0, len(accounts))
	for i := range accounts {
		plain := h.decryptRawCredentials(accountCredentialRaw(&accounts[i]))
		viewAccount := accounts[i]
		viewAccount.Credentials = plain
		viewAccount.LegacyAPIToken = plain
		if viewAccount.Provider == "" {
			viewAccount.Provider = model.DNSProviderCloudflare
		}
		views = append(views, viewAccount.ToView())
	}
	response.OK(c, views)
}

func (h *DNSProviderAccountHandler) Create(c *gin.Context) {
	var body struct {
		Provider    string            `json:"provider" binding:"required"`
		Name        string            `json:"name" binding:"required"`
		Credentials map[string]string `json:"credentials" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	provider := model.NormalizeProvider(body.Provider)
	if !model.IsSupportedProvider(provider) {
		response.Error(c, http.StatusBadRequest, "unsupported provider")
		return
	}
	if len(body.Credentials) == 0 {
		response.Error(c, http.StatusBadRequest, "credentials are required")
		return
	}
	if _, err := service.BuildProviderClient(provider, body.Credentials); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrProviderNotImplemented) {
			status = http.StatusUnprocessableEntity
		}
		response.Error(c, status, err.Error())
		return
	}

	rawJSON, err := json.Marshal(body.Credentials)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid credentials", "error.invalidRequestBody")
		return
	}
	encCredentials, err := h.encryptRawCredentials(string(rawJSON))
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt credentials", "error.encryptionFailed")
		return
	}

	account := &model.DNSProviderAccount{
		Provider:       provider,
		Name:           strings.TrimSpace(body.Name),
		Credentials:    encCredentials,
		LegacyAPIToken: encCredentials,
	}

	if err := h.repo.CreateDNSProviderAccount(account); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create account", "error.failedToCreateDNSProviderAccount")
		return
	}

	account.Credentials = string(rawJSON)
	account.LegacyAPIToken = string(rawJSON)
	response.Created(c, account.ToView())
}

func (h *DNSProviderAccountHandler) Update(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	account, err := h.repo.FindDNSProviderAccount(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "account not found", "error.cloudflareAccountNotFound")
		return
	}

	var body struct {
		Provider    string             `json:"provider"`
		Name        string             `json:"name" binding:"required"`
		Credentials *map[string]string `json:"credentials"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	originalProvider := model.NormalizeProvider(account.Provider)
	if originalProvider == "" {
		originalProvider = model.DNSProviderCloudflare
	}
	providerChanged := false
	if body.Provider != "" {
		provider := model.NormalizeProvider(body.Provider)
		if !model.IsSupportedProvider(provider) {
			response.Error(c, http.StatusBadRequest, "unsupported provider")
			return
		}
		providerChanged = provider != model.NormalizeProvider(account.Provider)
		account.Provider = provider
	}
	account.Name = strings.TrimSpace(body.Name)
	if providerChanged && body.Credentials == nil {
		response.Error(c, http.StatusBadRequest, "credentials are required when changing provider")
		return
	}

	parseExistingCredentials := func(provider string) (map[string]string, error) {
		plain := h.decryptRawCredentials(accountCredentialRaw(account))
		return service.ParseProviderCredentials(provider, plain)
	}

	incomingCredentials := map[string]string{}
	if body.Credentials != nil {
		for k, v := range *body.Credentials {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			incomingCredentials[key] = trimmed
		}
	}

	var credentialsForValidation map[string]string
	switch {
	case body.Credentials == nil:
		parsed, parseErr := parseExistingCredentials(originalProvider)
		if parseErr != nil {
			response.Error(c, http.StatusBadRequest, parseErr.Error())
			return
		}
		credentialsForValidation = parsed
	case providerChanged:
		if len(incomingCredentials) == 0 {
			response.Error(c, http.StatusBadRequest, "credentials are required")
			return
		}
		credentialsForValidation = incomingCredentials
	default:
		parsed, parseErr := parseExistingCredentials(originalProvider)
		if parseErr != nil {
			response.Error(c, http.StatusBadRequest, parseErr.Error())
			return
		}
		credentialsForValidation = parsed
		for k, v := range incomingCredentials {
			credentialsForValidation[k] = v
		}
	}
	if len(credentialsForValidation) == 0 {
		response.Error(c, http.StatusBadRequest, "credentials are required")
		return
	}
	if _, err := service.BuildProviderClient(account.Provider, credentialsForValidation); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrProviderNotImplemented) {
			status = http.StatusUnprocessableEntity
		}
		response.Error(c, status, err.Error())
		return
	}

	if body.Credentials != nil {
		rawJSON, err := json.Marshal(credentialsForValidation)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid credentials", "error.invalidRequestBody")
			return
		}
		encCredentials, err := h.encryptRawCredentials(string(rawJSON))
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt credentials", "error.encryptionFailed")
			return
		}
		account.Credentials = encCredentials
		account.LegacyAPIToken = encCredentials
	}

	if err := h.repo.UpdateDNSProviderAccount(account); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update account", "error.failedToUpdateDNSProviderAccount")
		return
	}

	plain := h.decryptRawCredentials(accountCredentialRaw(account))
	account.Credentials = plain
	account.LegacyAPIToken = plain
	response.OK(c, account.ToView())
}

func (h *DNSProviderAccountHandler) Delete(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	count, err := h.repo.CountDomainsByAccount(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	if count > 0 {
		response.ErrorWithKey(c, http.StatusConflict, "account has associated domains", "error.cloudflareAccountHasDomains")
		return
	}

	if err := h.repo.DeleteDNSProviderAccount(id); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete account", "error.failedToDeleteDNSProviderAccount")
		return
	}
	response.OK(c, gin.H{"message": "account deleted"})
}

func (h *DNSProviderAccountHandler) ListZones(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	account, err := h.repo.FindDNSProviderAccount(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "account not found", "error.cloudflareAccountNotFound")
		return
	}
	provider := model.NormalizeProvider(account.Provider)
	if provider == "" {
		provider = model.DNSProviderCloudflare
	}

	plain := h.decryptRawCredentials(accountCredentialRaw(account))
	credentials, err := service.ParseProviderCredentials(provider, plain)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	client, err := service.BuildProviderClient(provider, credentials)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrProviderNotImplemented) {
			status = http.StatusUnprocessableEntity
		}
		response.Error(c, status, err.Error())
		return
	}
	zones, err := client.ListZones(c.Request.Context())
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list zones", "error.failedToListCloudflareZones")
		return
	}

	existingDomains, err := h.repo.ListDomains(false)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list domains", "error.failedToListDomains")
		return
	}

	existingZoneKeys := make(map[string]struct{}, len(existingDomains))
	existingNamesLower := make(map[string]struct{}, len(existingDomains))
	for _, d := range existingDomains {
		existingZoneKeys[d.Provider+":"+d.ProviderZoneID] = struct{}{}
		existingNamesLower[strings.ToLower(strings.TrimSpace(d.Name))] = struct{}{}
	}

	filteredZones := make([]service.ZoneInfo, 0, len(zones))
	for _, zone := range zones {
		if _, found := existingZoneKeys[provider+":"+zone.ID]; found {
			continue
		}
		if _, found := existingNamesLower[strings.ToLower(strings.TrimSpace(zone.Name))]; found {
			continue
		}
		filteredZones = append(filteredZones, zone)
	}

	response.OK(c, filteredZones)
}

func accountCredentialRaw(account *model.DNSProviderAccount) string {
	if account == nil {
		return ""
	}
	if strings.TrimSpace(account.Credentials) != "" {
		return account.Credentials
	}
	return account.LegacyAPIToken
}
