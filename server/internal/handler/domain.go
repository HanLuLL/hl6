package handler

import (
	"encoding/json"
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
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}

	if user.GroupID == nil {
		response.OK(c, []interface{}{})
		return
	}

	domains, err := h.repo.ListDomainsForGroup(*user.GroupID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list domains", "error.failedToListDomains")
		return
	}
	response.OK(c, domains)
}

type groupAccessInput struct {
	GroupID       uint    `json:"group_id" binding:"required"`
	CreditCost    float64 `json:"credit_cost"`
	MaxDNSRecords *int    `json:"max_dns_records"`
}

func (h *DomainHandler) AdminCreate(c *gin.Context) {
	var body struct {
		Name             string             `json:"name" binding:"required"`
		CloudflareZoneID string             `json:"cloudflare_zone_id" binding:"required"`
		Description      string             `json:"description"`
		GroupAccess      []groupAccessInput `json:"group_access"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	domain := &model.Domain{
		Name:             body.Name,
		CloudflareZoneID: body.CloudflareZoneID,
		CreditCost:       model.CreditFromFloat(1),
		IsActive:         true,
		Description:      body.Description,
	}

	tx := h.repo.DB.Begin()
	if err := tx.Create(domain).Error; err != nil {
		tx.Rollback()
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create domain", "error.failedToCreateDomain")
		return
	}

	for _, ga := range body.GroupAccess {
		access := model.DomainGroupAccess{
			DomainID:      domain.ID,
			GroupID:       ga.GroupID,
			CreditCost:    model.CreditFromFloat(ga.CreditCost),
			MaxDNSRecords: ga.MaxDNSRecords,
		}
		if err := tx.Create(&access).Error; err != nil {
			tx.Rollback()
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create group access", "error.failedToCreateGroupAccess")
			return
		}
	}

	tx.Commit()

	// Audit log
	adminUser, _ := c.Get("db_user")
	if admin, ok := adminUser.(model.User); ok {
		details, _ := json.Marshal(map[string]interface{}{"domain_name": body.Name, "zone_id": body.CloudflareZoneID})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_create_domain",
			Resource:   "domain",
			ResourceID: domain.ID,
			Details:    details,
		})
	}

	// Return domain with group access
	accesses, _ := h.repo.ListDomainGroupAccess(domain.ID)
	response.Created(c, gin.H{"domain": domain, "group_access": accesses})
}

func (h *DomainHandler) AdminUpdate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.repo.FindDomain(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "domain not found", "error.domainNotFound")
		return
	}
	var body struct {
		CloudflareZoneID *string            `json:"cloudflare_zone_id"`
		IsActive         *bool              `json:"is_active"`
		Description      *string            `json:"description"`
		GroupAccess      []groupAccessInput `json:"group_access"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
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
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update domain", "error.failedToUpdateDomain")
		return
	}

	if body.GroupAccess != nil {
		var accesses []model.DomainGroupAccess
		for _, ga := range body.GroupAccess {
			accesses = append(accesses, model.DomainGroupAccess{
				GroupID:       ga.GroupID,
				CreditCost:    model.CreditFromFloat(ga.CreditCost),
				MaxDNSRecords: ga.MaxDNSRecords,
			})
		}
		if err := h.repo.ReplaceDomainGroupAccess(tx, domain.ID, accesses); err != nil {
			tx.Rollback()
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update group access", "error.failedToUpdateGroupAccess")
			return
		}
	}

	tx.Commit()

	adminUser, _ := c.Get("db_user")
	if admin, ok := adminUser.(model.User); ok {
		details, _ := json.Marshal(map[string]interface{}{"domain_name": domain.Name})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_update_domain",
			Resource:   "domain",
			ResourceID: domain.ID,
			Details:    details,
		})
	}

	accessList, _ := h.repo.ListDomainGroupAccess(domain.ID)
	response.OK(c, gin.H{"domain": domain, "group_access": accessList})
}

func (h *DomainHandler) AdminListDomainsFull(c *gin.Context) {
	domains, err := h.repo.ListDomains(false)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list domains", "error.failedToListDomains")
		return
	}

	accessMap, _ := h.repo.ListAllDomainGroupAccess()

	type domainWithAccess struct {
		model.Domain
		GroupAccess []model.DomainGroupAccess `json:"group_access"`
	}

	result := make([]domainWithAccess, len(domains))
	for i, d := range domains {
		accesses := accessMap[d.ID]
		if accesses == nil {
			accesses = []model.DomainGroupAccess{}
		}
		result[i] = domainWithAccess{Domain: d, GroupAccess: accesses}
	}
	response.OK(c, result)
}

func (h *DomainHandler) AdminListZones(c *gin.Context) {
	zones, err := h.cf.ListZones(c.Request.Context())
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list cloudflare zones", "error.failedToListCloudflareZones")
		return
	}
	response.OK(c, zones)
}

func (h *DomainHandler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid ID", "error.invalidID")
		return
	}

	domain, err := h.repo.FindDomain(uint(id))
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "domain not found", "error.domainNotFound")
		return
	}

	// Check no active subdomains
	count, err := h.repo.CountSubdomainsByDomain(domain.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	if count > 0 {
		response.ErrorWithKey(c, http.StatusConflict, "cannot delete domain with active subdomains", "error.domainHasSubdomains")
		return
	}

	tx := h.repo.DB.Begin()
	if err := h.repo.DeleteDomain(tx, domain.ID); err != nil {
		tx.Rollback()
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete domain", "error.failedToDeleteDomain")
		return
	}

	// Audit log
	adminUser, _ := c.Get("db_user")
	if admin, ok := adminUser.(model.User); ok {
		details, _ := json.Marshal(map[string]interface{}{"domain_name": domain.Name})
		tx.Create(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_delete_domain",
			Resource:   "domain",
			ResourceID: domain.ID,
			Details:    details,
		})
	}

	tx.Commit()
	response.OK(c, gin.H{"message": "domain deleted"})
}
