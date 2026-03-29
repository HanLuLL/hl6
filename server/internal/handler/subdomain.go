package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
	"hl6-server/pkg/validator"
)

type SubdomainHandler struct {
	repo   *repository.Repository
	cfg    *config.Config
	broker *SSEBroker
}

func NewSubdomainHandler(repo *repository.Repository, cfg *config.Config, broker *SSEBroker) *SubdomainHandler {
	return &SubdomainHandler{repo: repo, cfg: cfg, broker: broker}
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
	reservedPrefixes, err := loadReservedSubdomainPrefixes(h.repo)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load reserved subdomain prefixes", "error.databaseError")
		return
	}
	if isReservedSubdomainPrefix(body.Name, reservedPrefixes) {
		response.ErrorWithKey(c, http.StatusForbidden, "subdomain cannot be claimed", "error.subdomainNotClaimable")
		return
	}
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
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	fqdn := fmt.Sprintf("%s.%s", body.Name, domain.Name)
	fqdnParams, _ := json.Marshal(map[string]string{"fqdn": fqdn})

	sub := &model.Subdomain{
		DomainID:  domain.ID,
		UserID:    user.ID,
		Name:      body.Name,
		FQDN:      fqdn,
		ClaimCost: creditCost,
	}

	txErr := h.repo.Transaction(func(tx *gorm.DB) error {
		if creditCost > 0 {
			if err := h.repo.DeductCredits(tx, user.ID, creditCost, "txn.claimSubdomain", fqdnParams); err != nil {
				return err
			}
		} else if creditCost < 0 {
			if err := h.repo.GrantCredits(tx, user.ID, -creditCost, "txn.rewardClaimSubdomain", fqdnParams); err != nil {
				return err
			}
		}
		if err := tx.Create(sub).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]interface{}{"fqdn": fqdn, "cost": creditCost})
		return tx.Create(&model.AuditLog{
			UserID:     user.ID,
			Action:     "claim_subdomain",
			Resource:   "subdomain",
			ResourceID: sub.ID,
			Details:    details,
		}).Error
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrInvalidData) {
			response.ErrorWithKey(c, http.StatusPaymentRequired, "insufficient credits", "error.insufficientCredits")
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(txErr, &pgErr) && pgErr.Code == "23505" {
			response.ErrorWithKey(c, http.StatusConflict, "subdomain already taken", "error.subdomainAlreadyTaken")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create subdomain", "error.failedToCreateSubdomain")
		return
	}
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
			if errors.Is(err, service.ErrCloudflareTokenEmpty) {
				response.Error(c, http.StatusInternalServerError, err.Error())
			} else {
				response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
			}
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

	if err := h.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subdomain_id = ?", sub.ID).Delete(&model.DNSRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(sub).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]interface{}{"fqdn": sub.FQDN})
		return tx.Create(&model.AuditLog{
			UserID:     user.ID,
			Action:     "release_subdomain",
			Resource:   "subdomain",
			ResourceID: sub.ID,
			Details:    details,
		}).Error
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to release subdomain", "error.databaseError")
		return
	}

	response.OK(c, gin.H{"message": "subdomain released"})
}

func (h *SubdomainHandler) AdminListClaimed(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	search := strings.TrimSpace(c.Query("search"))
	subs, total, err := h.repo.AdminListClaimedSubdomains(page, perPage, search)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list claimed subdomains", "error.failedToListSubdomains")
		return
	}
	response.Paginated(c, subs, total, page, perPage)
}

func (h *SubdomainHandler) AdminRelease(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var body struct {
		Notify bool   `json:"notify"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		// Allow empty body.
		body.Notify = false
		body.Reason = ""
	}
	reason := strings.TrimSpace(body.Reason)

	sub, err := h.repo.FindSubdomain(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "subdomain not found", "error.subdomainNotFound")
		return
	}

	// Delete all Cloudflare records first; any failure aborts local cleanup.
	if len(sub.DNSRecords) > 0 {
		cf, err := cfForAccount(h.repo, h.cfg, sub.Domain.CloudflareAccountID)
		if err != nil {
			if errors.Is(err, service.ErrCloudflareTokenEmpty) {
				response.Error(c, http.StatusInternalServerError, err.Error())
			} else {
				response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
			}
			return
		}
		for _, record := range sub.DNSRecords {
			if record.CloudflareRecordID == "" {
				continue
			}
			if err := cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID); err != nil {
				log.Printf("cloudflare DeleteRecord error: %v", err)
				response.ErrorWithKey(c, http.StatusBadGateway, "cloudflare operation failed", "error.cloudflareError")
				return
			}
		}
	}

	if err := h.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subdomain_id = ?", sub.ID).Delete(&model.DNSRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Subdomain{}, sub.ID).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]interface{}{
			"fqdn":    sub.FQDN,
			"user_id": sub.UserID,
			"notify":  body.Notify,
		})
		return tx.Create(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_release_subdomain",
			Resource:   "subdomain",
			ResourceID: sub.ID,
			Details:    details,
		}).Error
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to release subdomain", "error.databaseError")
		return
	}

	if body.Notify {
		title := fmt.Sprintf("%s 认领已被管理员释放", sub.FQDN)
		content := fmt.Sprintf("您的认领 %s 已被管理员释放，相关解析记录已被删除。", sub.FQDN)
		if reason != "" {
			content = fmt.Sprintf("您的认领 %s 已被管理员释放，相关解析记录已被删除。\n原因：%s", sub.FQDN, reason)
		}
		targetIDs, _ := json.Marshal([]uint{sub.UserID})
		notification := &model.Notification{
			Title:      title,
			Content:    content,
			Type:       "urgent",
			TargetType: "users",
			TargetIDs:  targetIDs,
			CreatedBy:  admin.ID,
		}
		if err := h.repo.CreateNotificationWithImages(notification); err != nil {
			log.Printf("failed to create notification: %v", err)
		} else if h.broker != nil {
			event := SSEEvent{Event: "new_notification", Data: fmt.Sprintf(`{"id":%d}`, notification.ID)}
			h.broker.SendToUsers([]uint{sub.UserID}, event)
		}
	}

	response.OK(c, gin.H{"message": "subdomain released"})
}
