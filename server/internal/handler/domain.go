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
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

// cfFailureRecord 记录一条 CF DNS 删除失败的信息
type cfFailureRecord struct {
	SubdomainFQDN      string `json:"subdomain_fqdn"`
	RecordType         string `json:"record_type"`
	RecordContent      string `json:"record_content"`
	CloudflareRecordID string `json:"cloudflare_record_id"`
	Error              string `json:"error"`
}

type DomainHandler struct {
	repo *repository.Repository
}

func NewDomainHandler(repo *repository.Repository) *DomainHandler {
	return &DomainHandler{repo: repo}
}

func (h *DomainHandler) List(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
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
		Name                string             `json:"name" binding:"required"`
		CloudflareZoneID    string             `json:"cloudflare_zone_id" binding:"required"`
		CloudflareAccountID uint               `json:"cloudflare_account_id" binding:"required"`
		Description         string             `json:"description"`
		CreditCost          *float64           `json:"credit_cost"`
		GroupAccess         []groupAccessInput `json:"group_access"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	body.CloudflareZoneID = strings.TrimSpace(body.CloudflareZoneID)
	if body.Name == "" || body.CloudflareZoneID == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	exists, err := h.repo.DomainExistsByZoneIDOrName(body.CloudflareZoneID, body.Name)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	if exists {
		response.ErrorWithKey(c, http.StatusConflict, "domain already exists", "error.domainAlreadyExists")
		return
	}

	defaultCreditCost := 1.0
	if body.CreditCost != nil {
		defaultCreditCost = *body.CreditCost
	} else if len(body.GroupAccess) > 0 {
		defaultCreditCost = body.GroupAccess[0].CreditCost
	}

	domain := &model.Domain{
		Name:                body.Name,
		CloudflareZoneID:    body.CloudflareZoneID,
		CloudflareAccountID: body.CloudflareAccountID,
		CreditCost:          model.CreditFromFloat(defaultCreditCost),
		IsActive:            true,
		Description:         body.Description,
	}

	txErr := h.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(domain).Error; err != nil {
			return err
		}
		for _, ga := range body.GroupAccess {
			access := model.DomainGroupAccess{
				DomainID:      domain.ID,
				GroupID:       ga.GroupID,
				CreditCost:    model.CreditFromFloat(ga.CreditCost),
				MaxDNSRecords: ga.MaxDNSRecords,
			}
			if err := tx.Create(&access).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(txErr, &pgErr) && pgErr.Code == "23505" {
			response.ErrorWithKey(c, http.StatusConflict, "domain already exists", "error.domainAlreadyExists")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create domain", "error.failedToCreateDomain")
		return
	}

	// Audit log
	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{"domain_name": body.Name, "zone_id": body.CloudflareZoneID})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_create_domain",
			Resource:   "domain",
			ResourceID: domain.ID,
			Details:    details,
		})
	}

	accesses, err := h.repo.ListDomainGroupAccess(domain.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load group access", "error.failedToLoadGroupAccess")
		return
	}
	response.Created(c, gin.H{"domain": domain, "group_access": accesses})
}

func (h *DomainHandler) AdminUpdate(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	domain, err := h.repo.FindDomain(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "domain not found", "error.domainNotFound")
		return
	}
	var body struct {
		CloudflareZoneID    *string            `json:"cloudflare_zone_id"`
		CloudflareAccountID *uint              `json:"cloudflare_account_id"`
		IsActive            *bool              `json:"is_active"`
		Description         *string            `json:"description"`
		GroupAccess         []groupAccessInput `json:"group_access"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	if err := h.repo.Transaction(func(tx *gorm.DB) error {
		if body.CloudflareZoneID != nil {
			domain.CloudflareZoneID = *body.CloudflareZoneID
		}
		if body.CloudflareAccountID != nil {
			domain.CloudflareAccountID = *body.CloudflareAccountID
		}
		if body.IsActive != nil {
			domain.IsActive = *body.IsActive
		}
		if body.Description != nil {
			domain.Description = *body.Description
		}
		if err := tx.Save(domain).Error; err != nil {
			return err
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
				return err
			}
		}
		return nil
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update domain", "error.failedToUpdateDomain")
		return
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{"domain_name": domain.Name})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     admin.ID,
			Action:     "admin_update_domain",
			Resource:   "domain",
			ResourceID: domain.ID,
			Details:    details,
		})
	}

	accessList, err := h.repo.ListDomainGroupAccess(domain.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load group access", "error.failedToLoadGroupAccess")
		return
	}
	response.OK(c, gin.H{"domain": domain, "group_access": accessList})
}

func (h *DomainHandler) AdminListDomainsFull(c *gin.Context) {
	domains, err := h.repo.ListDomains(false)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list domains", "error.failedToListDomains")
		return
	}

	accessMap, err := h.repo.ListAllDomainGroupAccess()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list domains", "error.failedToListDomains")
		return
	}

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

func (h *DomainHandler) AdminGetReservedPrefixes(c *gin.Context) {
	prefixes, err := loadReservedSubdomainPrefixes(h.repo)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load reserved subdomain prefixes", "error.failedToGetConfig")
		return
	}
	lengthSettings, err := loadSubdomainLengthSettings(h.repo)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load subdomain length settings", "error.failedToGetConfig")
		return
	}
	response.OK(c, gin.H{
		"prefixes":   prefixes,
		"min_length": lengthSettings.MinLength,
		"max_length": lengthSettings.MaxLength,
	})
}

func (h *DomainHandler) AdminUpdateReservedPrefixes(c *gin.Context) {
	var body struct {
		Prefixes  []string `json:"prefixes"`
		MinLength int      `json:"min_length" binding:"required"`
		MaxLength int      `json:"max_length" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	normalized, err := normalizeReservedSubdomainPrefixes(body.Prefixes)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid reserved prefix", "error.invalidReservedPrefix")
		return
	}
	lengthSettings := subdomainLengthSettings{
		MinLength: body.MinLength,
		MaxLength: body.MaxLength,
	}
	if err := validateLengthSettings(lengthSettings.MinLength, lengthSettings.MaxLength); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid subdomain length settings", "error.invalidSubdomainLengthConfig")
		return
	}

	if err := saveReservedSubdomainPrefixes(h.repo, normalized); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save reserved subdomain prefixes", "error.failedToUpdateConfig")
		return
	}
	if err := saveSubdomainLengthSettings(h.repo, lengthSettings); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save subdomain length settings", "error.failedToUpdateConfig")
		return
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{
			"prefixes":   normalized,
			"min_length": lengthSettings.MinLength,
			"max_length": lengthSettings.MaxLength,
		})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_update_reserved_subdomain_prefixes",
			Resource: "system_config",
			Details:  details,
		})
	}

	response.OK(c, gin.H{
		"prefixes":   normalized,
		"min_length": lengthSettings.MinLength,
		"max_length": lengthSettings.MaxLength,
	})
}

func (h *DomainHandler) AdminDelete(c *gin.Context) {
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	refund := c.Query("refund") == "true"

	domain, err := h.repo.FindDomain(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "domain not found", "error.domainNotFound")
		return
	}

	// 查询所有子域（含 DNS 记录）
	subdomains, err := h.repo.ListSubdomainsByDomain(domain.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	type deleteCandidate struct {
		sub model.Subdomain
		rec model.DNSRecord
	}
	candidates := make([]deleteCandidate, 0)
	for _, sub := range subdomains {
		for _, rec := range sub.DNSRecords {
			candidates = append(candidates, deleteCandidate{sub: sub, rec: rec})
		}
	}

	if len(candidates) > 0 {
		if domain.CloudflareAccountID == 0 {
			response.ErrorWithKeyData(c, http.StatusConflict, "cloudflare account is missing", "error.cloudflareAccountNotFound", gin.H{
				"pending_delete_count": len(candidates),
			})
			return
		}
		if _, err := h.repo.FindCloudflareAccount(domain.CloudflareAccountID); err != nil {
			response.ErrorWithKeyData(c, http.StatusConflict, "cloudflare account not found", "error.cloudflareAccountNotFound", gin.H{
				"pending_delete_count": len(candidates),
			})
			return
		}
	}

	// 收集退还积分信息 — 使用 ClaimCost 而非 re-query 当前组价格
	type refundItem struct {
		userID    uint
		claimCost model.Credit
		subFQDN   string
	}
	var refundItems []refundItem
	if refund {
		for _, sub := range subdomains {
			if sub.UserID == 0 {
				continue
			}
			cost := sub.ClaimCost
			// Fallback for historical subdomains created before ClaimCost was added
			if cost == 0 {
				user, uErr := h.repo.FindUserByID(sub.UserID)
				if uErr != nil || user.GroupID == nil {
					continue
				}
				access, aErr := h.repo.FindDomainGroupAccess(domain.ID, *user.GroupID)
				if aErr != nil {
					continue
				}
				cost = access.CreditCost
			}
			if cost == 0 {
				continue
			}
			refundItems = append(refundItems, refundItem{
				userID:    sub.UserID,
				claimCost: cost,
				subFQDN:   sub.FQDN,
			})
		}
	}

	// 收集子域 ID 列表
	subdomainIDs := make([]uint, len(subdomains))
	for i, s := range subdomains {
		subdomainIDs[i] = s.ID
	}

	queuedOps := 0
	if err := h.repo.Transaction(func(tx *gorm.DB) error {
		if refund {
			for _, item := range refundItems {
				descParams, _ := json.Marshal(map[string]interface{}{"fqdn": item.subFQDN})
				if item.claimCost > 0 {
					if err := h.repo.RefundCredits(tx, item.userID, item.claimCost, "txn.subdomainDeletedRefund", descParams); err != nil {
						return err
					}
				} else {
					if err := h.repo.DeductCredits(tx, item.userID, -item.claimCost, "txn.subdomainDeletedDeduct", descParams); err != nil {
						return err
					}
				}
			}
		}
		if len(candidates) > 0 {
			for _, item := range candidates {
				if _, err := enqueueCloudflareTask(
					h.repo,
					tx,
					cloudflareTaskResourceDNSRecord,
					item.rec.ID,
					model.CloudflareTaskActionDeleteDNSRecord,
					model.CloudflareTaskPayload{
						CloudflareAccountID: domain.CloudflareAccountID,
						ZoneID:              domain.CloudflareZoneID,
						RecordID:            item.rec.CloudflareRecordID,
						RecordType:          item.rec.Type,
						Name:                item.rec.Name,
						Content:             item.rec.Content,
					},
					fmt.Sprintf("domain:delete:%d:%s", item.rec.ID, item.rec.CloudflareRecordID),
				); err != nil {
					return err
				}
				queuedOps++
			}
		}
		if len(subdomainIDs) > 0 {
			if err := tx.Where("subdomain_id IN ?", subdomainIDs).Delete(&model.DNSRecord{}).Error; err != nil {
				return err
			}
			if err := tx.Where("domain_id = ?", domain.ID).Delete(&model.Subdomain{}).Error; err != nil {
				return err
			}
		}
		if err := h.repo.DeleteDomain(tx, domain.ID); err != nil {
			return err
		}
		if admin := ctxutil.GetUser(c); admin != nil {
			details, _ := json.Marshal(map[string]interface{}{
				"domain_name":     domain.Name,
				"subdomain_count": len(subdomains),
				"dns_record_count": func() int {
					n := 0
					for _, s := range subdomains {
						n += len(s.DNSRecords)
					}
					return n
				}(),
				"refunded":         refund,
				"cf_queued_count":  queuedOps,
				"cf_skipped_count": len(candidates) - queuedOps,
			})
			if err := tx.Create(&model.AuditLog{
				UserID:     admin.ID,
				Action:     "admin_delete_domain",
				Resource:   "domain",
				ResourceID: domain.ID,
				Details:    details,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrInvalidData) {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to deduct credits", "error.failedToDeductCredits")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to delete domain", "error.failedToDeleteDomain")
		return
	}
	data := gin.H{"message": "domain deleted"}
	if queuedOps > 0 {
		data["sync_status"] = "processing"
		data["operation_count"] = queuedOps
	}
	response.OK(c, data)
}
