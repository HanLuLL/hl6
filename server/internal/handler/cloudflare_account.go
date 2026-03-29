package handler

import (
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

type CloudflareAccountHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewCloudflareAccountHandler(repo *repository.Repository, cfg *config.Config) *CloudflareAccountHandler {
	return &CloudflareAccountHandler{repo: repo, cfg: cfg}
}

func (h *CloudflareAccountHandler) encryptToken(token string) (string, error) {
	return crypto.EncryptIfKey(token, h.cfg.EncryptionKey)
}

func (h *CloudflareAccountHandler) decryptToken(token string) string {
	return crypto.DecryptOrPlaintext(token, h.cfg.EncryptionKey)
}

func (h *CloudflareAccountHandler) List(c *gin.Context) {
	accounts, err := h.repo.ListCloudflareAccounts()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list accounts", "error.failedToListCloudflareAccounts")
		return
	}
	views := make([]model.CloudflareAccountView, len(accounts))
	for i, a := range accounts {
		a.ApiToken = h.decryptToken(a.ApiToken)
		views[i] = a.ToView()
	}
	response.OK(c, views)
}

func (h *CloudflareAccountHandler) Create(c *gin.Context) {
	var body struct {
		Name     string `json:"name" binding:"required"`
		ApiToken string `json:"api_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	encToken, err := h.encryptToken(body.ApiToken)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt token", "error.encryptionFailed")
		return
	}
	account := &model.CloudflareAccount{
		Name:     body.Name,
		ApiToken: encToken,
	}
	if err := h.repo.CreateCloudflareAccount(account); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create account", "error.failedToCreateCloudflareAccount")
		return
	}
	account.ApiToken = body.ApiToken // use plaintext for ToView hint
	response.Created(c, account.ToView())
}

func (h *CloudflareAccountHandler) Update(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	account, err := h.repo.FindCloudflareAccount(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "account not found", "error.cloudflareAccountNotFound")
		return
	}

	var body struct {
		Name     string `json:"name" binding:"required"`
		ApiToken string `json:"api_token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	account.Name = body.Name
	if body.ApiToken != "" {
		encToken, err := h.encryptToken(body.ApiToken)
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt token", "error.encryptionFailed")
			return
		}
		account.ApiToken = encToken
	}

	if err := h.repo.UpdateCloudflareAccount(account); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update account", "error.failedToUpdateCloudflareAccount")
		return
	}
	if body.ApiToken != "" {
		account.ApiToken = body.ApiToken // use plaintext for ToView hint
	} else {
		account.ApiToken = h.decryptToken(account.ApiToken)
	}
	response.OK(c, account.ToView())
}

func (h *CloudflareAccountHandler) Delete(c *gin.Context) {
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

	if err := h.repo.DeleteCloudflareAccount(id); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete account", "error.failedToDeleteCloudflareAccount")
		return
	}
	response.OK(c, gin.H{"message": "account deleted"})
}

func (h *CloudflareAccountHandler) ListZones(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	account, err := h.repo.FindCloudflareAccount(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "account not found", "error.cloudflareAccountNotFound")
		return
	}

	cf, err := service.NewCloudflareService(h.decryptToken(account.ApiToken))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	zones, err := cf.ListZones(c.Request.Context())
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list cloudflare zones", "error.failedToListCloudflareZones")
		return
	}

	existingDomains, err := h.repo.ListDomains(false)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list domains", "error.failedToListDomains")
		return
	}

	existingZoneIDs := make(map[string]struct{}, len(existingDomains))
	existingNamesLower := make(map[string]struct{}, len(existingDomains))
	for _, d := range existingDomains {
		existingZoneIDs[d.CloudflareZoneID] = struct{}{}
		existingNamesLower[strings.ToLower(strings.TrimSpace(d.Name))] = struct{}{}
	}

	filteredZones := make([]service.ZoneInfo, 0, len(zones))
	for _, zone := range zones {
		if _, found := existingZoneIDs[zone.ID]; found {
			continue
		}
		if _, found := existingNamesLower[strings.ToLower(strings.TrimSpace(zone.Name))]; found {
			continue
		}
		filteredZones = append(filteredZones, zone)
	}

	response.OK(c, filteredZones)
}

// cfForAccount is a helper to get a CloudflareService for a given account ID.
// It decrypts the token if encryption is configured.
func cfForAccount(repo *repository.Repository, cfg *config.Config, accountID uint) (*service.CloudflareService, error) {
	account, err := repo.FindCloudflareAccount(accountID)
	if err != nil {
		return nil, err
	}
	token := crypto.DecryptOrPlaintext(account.ApiToken, cfg.EncryptionKey)
	return service.NewCloudflareService(token)
}
