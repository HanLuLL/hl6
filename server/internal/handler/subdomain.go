package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/pkg/response"
	"hl6-server/pkg/validator"
)

type SubdomainHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewSubdomainHandler(repo *repository.Repository, cfg *config.Config) *SubdomainHandler {
	return &SubdomainHandler{repo: repo, cfg: cfg}
}

func (h *SubdomainHandler) List(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	subs, err := h.repo.ListSubdomainsByUser(user.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list subdomains", "error.failedToListSubdomains")
		return
	}
	response.OK(c, subs)
}

func (h *SubdomainHandler) Get(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	sub, err := h.repo.FindSubdomain(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "subdomain not found", "error.subdomainNotFound")
		return
	}
	if sub.UserID != user.ID {
		response.ErrorWithKey(c, http.StatusForbidden, "not your subdomain", "error.notYourSubdomain")
		return
	}
	response.OK(c, sub)
}

func (h *SubdomainHandler) Claim(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	var body struct {
		DomainID uint   `json:"domain_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	body.Name = strings.ToLower(strings.TrimSpace(body.Name))
	if err := validator.ValidateSubdomainName(body.Name); err != nil {
		if ve, ok := err.(*validator.ValidationError); ok {
			response.ErrorWithKey(c, http.StatusBadRequest, ve.Message, ve.Key)
		} else {
			response.ErrorWithKey(c, http.StatusBadRequest, err.Error(), "error.invalidSubdomainName")
		}
		return
	}

	domain, err := h.repo.FindDomain(body.DomainID)
	if err != nil || !domain.IsActive {
		response.ErrorWithKey(c, http.StatusNotFound, "domain not found or inactive", "error.domainNotFoundOrInactive")
		return
	}

	// Check group access and get group-specific cost
	if user.GroupID == nil {
		response.ErrorWithKey(c, http.StatusForbidden, "user has no group assigned", "error.noGroupAssigned")
		return
	}
	access, err := h.repo.FindDomainGroupAccess(domain.ID, *user.GroupID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusForbidden, "your group does not have access to this domain", "error.groupNoAccess")
		return
	}
	creditCost := access.CreditCost

	if _, err := h.repo.FindSubdomainByName(domain.ID, body.Name); err == nil {
		response.ErrorWithKey(c, http.StatusConflict, "subdomain already taken", "error.subdomainAlreadyTaken")
		return
	}

	fqdn := fmt.Sprintf("%s.%s", body.Name, domain.Name)
	tx := h.repo.GetDB().Begin()

	fqdnParams, _ := json.Marshal(map[string]string{"fqdn": fqdn})

	if creditCost > 0 {
		if err := h.repo.DeductCredits(tx, user.ID, creditCost, "txn.claimSubdomain", fqdnParams); err != nil {
			tx.Rollback()
			if err == gorm.ErrInvalidData {
				response.ErrorWithKey(c, http.StatusPaymentRequired, "insufficient credits", "error.insufficientCredits")
				return
			}
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to deduct credits", "error.failedToDeductCredits")
			return
		}
	} else if creditCost < 0 {
		if err := h.repo.GrantCredits(tx, user.ID, -creditCost, "txn.rewardClaimSubdomain", fqdnParams); err != nil {
			tx.Rollback()
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to grant credits", "error.failedToGrantCredits")
			return
		}
	}

	sub := &model.Subdomain{
		DomainID:  domain.ID,
		UserID:    user.ID,
		Name:      body.Name,
		FQDN:      fqdn,
		ClaimCost: creditCost,
	}
	if err := tx.Create(sub).Error; err != nil {
		tx.Rollback()
		// Check for PostgreSQL unique constraint violation (code 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.ErrorWithKey(c, http.StatusConflict, "subdomain already taken", "error.subdomainAlreadyTaken")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create subdomain", "error.failedToCreateSubdomain")
		return
	}

	details, _ := json.Marshal(map[string]interface{}{"fqdn": fqdn, "cost": creditCost})
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
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	sub, err := h.repo.FindSubdomain(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "subdomain not found", "error.subdomainNotFound")
		return
	}
	if sub.UserID != user.ID {
		response.ErrorWithKey(c, http.StatusForbidden, "not your subdomain", "error.notYourSubdomain")
		return
	}

	// Delete all CF records first - all must succeed before DB cleanup
	if len(sub.DNSRecords) > 0 {
		cf, err := cfForAccount(h.repo, h.cfg, sub.Domain.CloudflareAccountID)
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
			return
		}
		for _, record := range sub.DNSRecords {
			if record.CloudflareRecordID != "" {
				if err := cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID); err != nil {
					response.ErrorWithKey(c, http.StatusBadGateway, "failed to delete DNS record from Cloudflare", "error.cloudflareDeleteFailed")
					return
				}
			}
		}
	}

	tx := h.repo.GetDB().Begin()
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
