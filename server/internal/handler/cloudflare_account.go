package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

type CloudflareAccountHandler struct {
	repo *repository.Repository
}

func NewCloudflareAccountHandler(repo *repository.Repository) *CloudflareAccountHandler {
	return &CloudflareAccountHandler{repo: repo}
}

func (h *CloudflareAccountHandler) List(c *gin.Context) {
	accounts, err := h.repo.ListCloudflareAccounts()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list accounts", "error.failedToListCloudflareAccounts")
		return
	}
	views := make([]model.CloudflareAccountView, len(accounts))
	for i, a := range accounts {
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

	account := &model.CloudflareAccount{
		Name:     body.Name,
		ApiToken: body.ApiToken,
	}
	if err := h.repo.CreateCloudflareAccount(account); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create account", "error.failedToCreateCloudflareAccount")
		return
	}
	response.Created(c, account.ToView())
}

func (h *CloudflareAccountHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid ID", "error.invalidID")
		return
	}

	account, err := h.repo.FindCloudflareAccount(uint(id))
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
		account.ApiToken = body.ApiToken
	}

	if err := h.repo.UpdateCloudflareAccount(account); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update account", "error.failedToUpdateCloudflareAccount")
		return
	}
	response.OK(c, account.ToView())
}

func (h *CloudflareAccountHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid ID", "error.invalidID")
		return
	}

	count, err := h.repo.CountDomainsByAccount(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	if count > 0 {
		response.ErrorWithKey(c, http.StatusConflict, "account has associated domains", "error.cloudflareAccountHasDomains")
		return
	}

	if err := h.repo.DeleteCloudflareAccount(uint(id)); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete account", "error.failedToDeleteCloudflareAccount")
		return
	}
	response.OK(c, gin.H{"message": "account deleted"})
}

func (h *CloudflareAccountHandler) ListZones(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid ID", "error.invalidID")
		return
	}

	account, err := h.repo.FindCloudflareAccount(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "account not found", "error.cloudflareAccountNotFound")
		return
	}

	cf := service.NewCloudflareService(account.ApiToken)
	zones, err := cf.ListZones(c.Request.Context())
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list cloudflare zones", "error.failedToListCloudflareZones")
		return
	}
	response.OK(c, zones)
}

// cfForAccount is a helper to get a CloudflareService for a given account ID.
func cfForAccount(repo *repository.Repository, accountID uint) (*service.CloudflareService, error) {
	account, err := repo.FindCloudflareAccount(accountID)
	if err != nil {
		return nil, err
	}
	return service.NewCloudflareService(account.ApiToken), nil
}
