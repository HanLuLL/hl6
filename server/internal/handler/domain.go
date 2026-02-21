package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type DomainHandler struct {
	repo *repository.Repository
}

func NewDomainHandler(repo *repository.Repository) *DomainHandler {
	return &DomainHandler{repo: repo}
}

func (h *DomainHandler) List(c *gin.Context) {
	domains, err := h.repo.ListDomains(true)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list domains")
		return
	}
	response.OK(c, domains)
}

func (h *DomainHandler) AdminCreate(c *gin.Context) {
	var body struct {
		Name             string `json:"name" binding:"required"`
		CloudflareZoneID string `json:"cloudflare_zone_id" binding:"required"`
		CreditCost       int    `json:"credit_cost"`
		Description      string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.CreditCost <= 0 {
		body.CreditCost = 1
	}
	domain := &model.Domain{
		Name:             body.Name,
		CloudflareZoneID: body.CloudflareZoneID,
		CreditCost:       body.CreditCost,
		IsActive:         true,
		Description:      body.Description,
	}
	if err := h.repo.CreateDomain(domain); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create domain")
		return
	}
	response.Created(c, domain)
}

func (h *DomainHandler) AdminUpdate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.repo.FindDomain(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "domain not found")
		return
	}
	var body struct {
		CloudflareZoneID *string `json:"cloudflare_zone_id"`
		CreditCost       *int    `json:"credit_cost"`
		IsActive         *bool   `json:"is_active"`
		Description      *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.CloudflareZoneID != nil {
		domain.CloudflareZoneID = *body.CloudflareZoneID
	}
	if body.CreditCost != nil {
		domain.CreditCost = *body.CreditCost
	}
	if body.IsActive != nil {
		domain.IsActive = *body.IsActive
	}
	if body.Description != nil {
		domain.Description = *body.Description
	}
	if err := h.repo.UpdateDomain(domain); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update domain")
		return
	}
	response.OK(c, domain)
}
