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

type DomainHandler struct {
	repo *repository.Repository
	cf   *service.CloudflareService
}

func NewDomainHandler(repo *repository.Repository, cf *service.CloudflareService) *DomainHandler {
	return &DomainHandler{repo: repo, cf: cf}
}

func (h *DomainHandler) List(c *gin.Context) {
	logtoID := c.GetString("user_id")
	user, err := h.repo.FindUserByLogtoID(logtoID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "user not found")
		return
	}

	if user.GroupID == nil {
		response.OK(c, []interface{}{})
		return
	}

	domains, err := h.repo.ListDomainsForGroup(*user.GroupID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list domains")
		return
	}
	response.OK(c, domains)
}

type groupAccessInput struct {
	GroupID    uint    `json:"group_id" binding:"required"`
	CreditCost float64 `json:"credit_cost"`
}

func (h *DomainHandler) AdminCreate(c *gin.Context) {
	var body struct {
		Name             string             `json:"name" binding:"required"`
		CloudflareZoneID string             `json:"cloudflare_zone_id" binding:"required"`
		Description      string             `json:"description"`
		GroupAccess      []groupAccessInput `json:"group_access"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	domain := &model.Domain{
		Name:             body.Name,
		CloudflareZoneID: body.CloudflareZoneID,
		CreditCost:       1,
		IsActive:         true,
		Description:      body.Description,
	}

	tx := h.repo.DB.Begin()
	if err := tx.Create(domain).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to create domain")
		return
	}

	for _, ga := range body.GroupAccess {
		access := model.DomainGroupAccess{
			DomainID:   domain.ID,
			GroupID:    ga.GroupID,
			CreditCost: ga.CreditCost,
		}
		if err := tx.Create(&access).Error; err != nil {
			tx.Rollback()
			response.Error(c, http.StatusInternalServerError, "failed to create group access")
			return
		}
	}

	tx.Commit()

	// Return domain with group access
	accesses, _ := h.repo.ListDomainGroupAccess(domain.ID)
	response.Created(c, gin.H{"domain": domain, "group_access": accesses})
}

func (h *DomainHandler) AdminUpdate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.repo.FindDomain(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "domain not found")
		return
	}
	var body struct {
		CloudflareZoneID *string            `json:"cloudflare_zone_id"`
		IsActive         *bool              `json:"is_active"`
		Description      *string            `json:"description"`
		GroupAccess      []groupAccessInput `json:"group_access"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	tx := h.repo.DB.Begin()

	if body.CloudflareZoneID != nil {
		domain.CloudflareZoneID = *body.CloudflareZoneID
	}
	if body.IsActive != nil {
		domain.IsActive = *body.IsActive
	}
	if body.Description != nil {
		domain.Description = *body.Description
	}
	if err := tx.Save(domain).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to update domain")
		return
	}

	if body.GroupAccess != nil {
		var accesses []model.DomainGroupAccess
		for _, ga := range body.GroupAccess {
			accesses = append(accesses, model.DomainGroupAccess{
				GroupID:    ga.GroupID,
				CreditCost: ga.CreditCost,
			})
		}
		if err := h.repo.ReplaceDomainGroupAccess(tx, domain.ID, accesses); err != nil {
			tx.Rollback()
			response.Error(c, http.StatusInternalServerError, "failed to update group access")
			return
		}
	}

	tx.Commit()

	accessList, _ := h.repo.ListDomainGroupAccess(domain.ID)
	response.OK(c, gin.H{"domain": domain, "group_access": accessList})
}

func (h *DomainHandler) AdminListDomainsFull(c *gin.Context) {
	domains, err := h.repo.ListDomains(false)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list domains")
		return
	}

	type domainWithAccess struct {
		model.Domain
		GroupAccess []model.DomainGroupAccess `json:"group_access"`
	}

	var result []domainWithAccess
	for _, d := range domains {
		accesses, _ := h.repo.ListDomainGroupAccess(d.ID)
		result = append(result, domainWithAccess{Domain: d, GroupAccess: accesses})
	}
	response.OK(c, result)
}

func (h *DomainHandler) AdminListZones(c *gin.Context) {
	zones, err := h.cf.ListZones(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list cloudflare zones")
		return
	}
	response.OK(c, zones)
}
