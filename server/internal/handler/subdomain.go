package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
	"hl6-server/pkg/validator"
)

type SubdomainHandler struct {
	repo *repository.Repository
	cf   *service.CloudflareService
}

func NewSubdomainHandler(repo *repository.Repository, cf *service.CloudflareService) *SubdomainHandler {
	return &SubdomainHandler{repo: repo, cf: cf}
}

func (h *SubdomainHandler) List(c *gin.Context) {
	user := h.getUser(c)
	if user == nil {
		return
	}
	subs, err := h.repo.ListSubdomainsByUser(user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list subdomains")
		return
	}
	response.OK(c, subs)
}

func (h *SubdomainHandler) Get(c *gin.Context) {
	user := h.getUser(c)
	if user == nil {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := h.repo.FindSubdomain(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "subdomain not found")
		return
	}
	if sub.UserID != user.ID {
		response.Error(c, http.StatusForbidden, "not your subdomain")
		return
	}
	response.OK(c, sub)
}

func (h *SubdomainHandler) Claim(c *gin.Context) {
	user := h.getUser(c)
	if user == nil {
		return
	}
	var body struct {
		DomainID uint   `json:"domain_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	body.Name = strings.ToLower(strings.TrimSpace(body.Name))
	if err := validator.ValidateSubdomainName(body.Name); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	domain, err := h.repo.FindDomain(body.DomainID)
	if err != nil || !domain.IsActive {
		response.Error(c, http.StatusNotFound, "domain not found or inactive")
		return
	}

	if _, err := h.repo.FindSubdomainByName(domain.ID, body.Name); err == nil {
		response.Error(c, http.StatusConflict, "subdomain already taken")
		return
	}

	fqdn := fmt.Sprintf("%s.%s", body.Name, domain.Name)
	tx := h.repo.DB.Begin()
	desc := fmt.Sprintf("Claim subdomain %s", fqdn)
	if err := h.repo.DeductCredits(tx, user.ID, domain.CreditCost, desc); err != nil {
		tx.Rollback()
		if err == gorm.ErrInvalidData {
			response.Error(c, http.StatusPaymentRequired, "insufficient credits")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to deduct credits")
		return
	}

	sub := &model.Subdomain{
		DomainID: domain.ID,
		UserID:   user.ID,
		Name:     body.Name,
		FQDN:     fqdn,
	}
	if err := tx.Create(sub).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to create subdomain")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{"fqdn": fqdn, "cost": domain.CreditCost})
	tx.Create(&model.AuditLog{
		UserID:     user.ID,
		Action:     "claim_subdomain",
		Resource:   "subdomain",
		ResourceID: sub.ID,
		Details:    details,
	})

	tx.Commit()
	response.Created(c, sub)
}

func (h *SubdomainHandler) Release(c *gin.Context) {
	user := h.getUser(c)
	if user == nil {
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := h.repo.FindSubdomain(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "subdomain not found")
		return
	}
	if sub.UserID != user.ID {
		response.Error(c, http.StatusForbidden, "not your subdomain")
		return
	}

	for _, record := range sub.DNSRecords {
		if record.CloudflareRecordID != "" {
			_ = h.cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID)
		}
	}

	tx := h.repo.DB.Begin()
	tx.Where("subdomain_id = ?", sub.ID).Delete(&model.DNSRecord{})
	tx.Delete(sub)

	details, _ := json.Marshal(map[string]interface{}{"fqdn": sub.FQDN})
	tx.Create(&model.AuditLog{
		UserID:     user.ID,
		Action:     "release_subdomain",
		Resource:   "subdomain",
		ResourceID: sub.ID,
		Details:    details,
	})
	tx.Commit()

	response.OK(c, gin.H{"message": "subdomain released"})
}

func (h *SubdomainHandler) getUser(c *gin.Context) *model.User {
	logtoID := c.GetString("user_id")
	user, err := h.repo.FindUserByLogtoID(logtoID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "user not found")
		c.Abort()
		return nil
	}
	return user
}
