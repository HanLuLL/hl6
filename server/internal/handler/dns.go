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
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
	"hl6-server/pkg/validator"
)

type DNSHandler struct {
	repo   *repository.Repository
	cfg    *config.Config
	broker *SSEBroker
}

func NewDNSHandler(repo *repository.Repository, cfg *config.Config, broker *SSEBroker) *DNSHandler {
	return &DNSHandler{repo: repo, cfg: cfg, broker: broker}
}

func (h *DNSHandler) ListRecords(c *gin.Context) {
	sub := h.getSubdomain(c)
	if sub == nil {
		return
	}
	records, err := h.repo.ListDNSRecords(sub.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list records", "error.failedToListRecords")
		return
	}
	response.OK(c, records)
}

func (h *DNSHandler) CreateRecord(c *gin.Context) {
	sub := h.getSubdomain(c)
	if sub == nil {
		return
	}
	var body struct {
		Type    string `json:"type" binding:"required"`
		Content string `json:"content" binding:"required"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	body.Type = strings.ToUpper(body.Type)
	if err := validator.ValidateDNSRecord(body.Type, body.Content); err != nil {
		if ve, ok := err.(*validator.ValidationError); ok {
			response.ErrorWithKey(c, http.StatusBadRequest, ve.Message, ve.Key)
		} else {
			response.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	if body.Type == "CNAME" {
		hasOther, _ := h.repo.HasNonCNAMERecords(sub.ID)
		if hasOther {
			response.ErrorWithKey(c, http.StatusConflict, "CNAME record cannot coexist with other records", "error.cnameConflictWithOther")
			return
		}
		hasCNAME, _ := h.repo.HasCNAMERecord(sub.ID)
		if hasCNAME {
			response.ErrorWithKey(c, http.StatusConflict, "CNAME record already exists", "error.cnameAlreadyExists")
			return
		}
	} else {
		hasCNAME, _ := h.repo.HasCNAMERecord(sub.ID)
		if hasCNAME {
			response.ErrorWithKey(c, http.StatusConflict, "CNAME record cannot coexist with other records", "error.otherConflictWithCname")
			return
		}
	}

	dup, _ := h.repo.HasDuplicateRecord(sub.ID, body.Type, body.Content)
	if dup {
		response.ErrorWithKey(c, http.StatusConflict, "duplicate record", "error.duplicateRecord")
		return
	}

	// 检查域名+用户组的 DNS 记录数上限
	user := ctxutil.GetUser(c)
	if user != nil && user.GroupID != nil {
		access, err := h.repo.FindDomainGroupAccess(sub.DomainID, *user.GroupID)
		if err == nil && access.MaxDNSRecords != nil {
			count, _ := h.repo.CountDNSRecordsBySubdomain(sub.ID)
			if int(count) >= *access.MaxDNSRecords {
				response.ErrorWithKey(c, http.StatusUnprocessableEntity,
					"dns record limit exceeded", "error.dnsRecordLimitExceeded")
				return
			}
		}
	}

	if body.Type == "TXT" {
		body.Proxied = false
	}

	if body.TTL <= 0 {
		body.TTL = 1
	}

	cf, err := cfForAccount(h.repo, h.cfg, sub.Domain.CloudflareAccountID)
	if err != nil {
		if errors.Is(err, service.ErrCloudflareTokenEmpty) {
			response.Error(c, http.StatusInternalServerError, err.Error())
		} else {
			response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
		}
		return
	}

	cfID, err := cf.CreateRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, body.Type, sub.FQDN, body.Content, body.TTL, body.Proxied)
	if err != nil {
		log.Printf("cloudflare CreateRecord error: %v", err)
		response.ErrorWithKey(c, http.StatusBadGateway, "cloudflare operation failed", "error.cloudflareError")
		return
	}

	record := &model.DNSRecord{
		SubdomainID:        sub.ID,
		Type:               body.Type,
		Name:               sub.FQDN,
		Content:            body.Content,
		TTL:                body.TTL,
		Proxied:            body.Proxied,
		CloudflareRecordID: cfID,
	}
	if err := h.repo.CreateDNSRecord(record); err != nil {
		// Rollback Cloudflare record
		if delErr := cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, cfID); delErr != nil {
			log.Printf("failed to rollback cloudflare record %s: %v", cfID, delErr)
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save record", "error.failedToSaveRecord")
		return
	}

	if user != nil {
		details, _ := json.Marshal(map[string]interface{}{
			"type": body.Type, "content": body.Content, "fqdn": sub.FQDN,
		})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:     user.ID,
			Action:     "create_dns_record",
			Resource:   "dns_record",
			ResourceID: record.ID,
			Details:    details,
		})
	}

	response.Created(c, record)
}

func (h *DNSHandler) UpdateRecord(c *gin.Context) {
	sub := h.getSubdomain(c)
	if sub == nil {
		return
	}
	recordID, ok := helpers.ParseUintParam(c, "recordId")
	if !ok {
		return
	}
	record, err := h.repo.FindDNSRecord(recordID)
	if err != nil || record.SubdomainID != sub.ID {
		response.ErrorWithKey(c, http.StatusNotFound, "record not found", "error.recordNotFound")
		return
	}

	var body struct {
		Content string `json:"content" binding:"required"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	if err := validator.ValidateDNSRecord(record.Type, body.Content); err != nil {
		if ve, ok := err.(*validator.ValidationError); ok {
			response.ErrorWithKey(c, http.StatusBadRequest, ve.Message, ve.Key)
		} else {
			response.Error(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	dup, _ := h.repo.HasDuplicateRecordExcluding(sub.ID, record.Type, body.Content, record.ID)
	if dup {
		response.ErrorWithKey(c, http.StatusConflict, "duplicate record", "error.duplicateRecord")
		return
	}

	if record.Type == "TXT" {
		body.Proxied = false
	}

	if body.TTL <= 0 {
		body.TTL = 1
	}

	cf, err := cfForAccount(h.repo, h.cfg, sub.Domain.CloudflareAccountID)
	if err != nil {
		if errors.Is(err, service.ErrCloudflareTokenEmpty) {
			response.Error(c, http.StatusInternalServerError, err.Error())
		} else {
			response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
		}
		return
	}

	if err := cf.UpdateRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID, record.Type, sub.FQDN, body.Content, body.TTL, body.Proxied); err != nil {
		log.Printf("cloudflare UpdateRecord error: %v", err)
		response.ErrorWithKey(c, http.StatusBadGateway, "cloudflare operation failed", "error.cloudflareError")
		return
	}

	record.Content = body.Content
	record.TTL = body.TTL
	record.Proxied = body.Proxied
	if err := h.repo.UpdateDNSRecord(record); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	response.OK(c, record)
}

func (h *DNSHandler) DeleteRecord(c *gin.Context) {
	sub := h.getSubdomain(c)
	if sub == nil {
		return
	}
	recordID, ok := helpers.ParseUintParam(c, "recordId")
	if !ok {
		return
	}
	record, err := h.repo.FindDNSRecord(recordID)
	if err != nil || record.SubdomainID != sub.ID {
		response.ErrorWithKey(c, http.StatusNotFound, "record not found", "error.recordNotFound")
		return
	}

	if record.CloudflareRecordID != "" {
		cf, err := cfForAccount(h.repo, h.cfg, sub.Domain.CloudflareAccountID)
		if err != nil {
			if errors.Is(err, service.ErrCloudflareTokenEmpty) {
				response.Error(c, http.StatusInternalServerError, err.Error())
			} else {
				response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
			}
			return
		}
		if err := cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID); err != nil {
			log.Printf("cloudflare DeleteRecord error: %v", err)
			response.ErrorWithKey(c, http.StatusBadGateway, "cloudflare operation failed", "error.cloudflareError")
			return
		}
	}

	if err := h.repo.DeleteDNSRecord(record.ID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	response.OK(c, gin.H{"message": "record deleted"})
}

func (h *DNSHandler) getSubdomain(c *gin.Context) *model.Subdomain {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		c.Abort()
		return nil
	}
	id, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		c.Abort()
		return nil
	}
	sub, err := h.repo.FindSubdomain(id)
	if err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "subdomain not found", "error.subdomainNotFound")
		c.Abort()
		return nil
	}
	if sub.UserID != user.ID {
		response.ErrorWithKey(c, http.StatusForbidden, "not your subdomain", "error.notYourSubdomain")
		c.Abort()
		return nil
	}
	return sub
}

func (h *DNSHandler) AdminListRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	search := c.Query("search")

	var domainID *uint
	if v := c.Query("domain_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			uid := uint(id)
			domainID = &uid
		}
	}

	var groupID *uint
	if v := c.Query("group_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			uid := uint(id)
			groupID = &uid
		}
	}

	records, total, err := h.repo.AdminListDNSRecords(page, perPage, search, domainID, groupID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list records", "error.failedToListRecords")
		return
	}
	response.Paginated(c, records, total, page, perPage)
}

func (h *DNSHandler) AdminDeleteRecord(c *gin.Context) {
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}

	recordID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var body struct {
		Notify bool   `json:"notify"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		// Allow empty body
		body.Notify = false
		body.Reason = ""
	}

	record, sub, err := h.repo.FindDNSRecordWithSubdomain(recordID)
	if err != nil || sub == nil {
		response.ErrorWithKey(c, http.StatusNotFound, "record not found", "error.recordNotFound")
		return
	}

	// Delete from Cloudflare
	if record.CloudflareRecordID != "" {
		cf, err := cfForAccount(h.repo, h.cfg, sub.Domain.CloudflareAccountID)
		if err != nil {
			if errors.Is(err, service.ErrCloudflareTokenEmpty) {
				response.Error(c, http.StatusInternalServerError, err.Error())
			} else {
				response.ErrorWithKey(c, http.StatusInternalServerError, "cloudflare account not found", "error.cloudflareAccountNotFound")
			}
			return
		}
		if err := cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID); err != nil {
			log.Printf("cloudflare DeleteRecord error: %v", err)
			response.ErrorWithKey(c, http.StatusBadGateway, "cloudflare operation failed", "error.cloudflareError")
			return
		}
	}

	// Delete from database
	if err := h.repo.DeleteDNSRecord(record.ID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	// Send notification if requested
	if body.Notify {
		fqdn := record.Name
		title := fmt.Sprintf("%s 解析已被删除", fqdn)
		content := fmt.Sprintf("您的解析 %s 已被删除。", fqdn)
		if body.Reason != "" {
			content = fmt.Sprintf("您的解析 %s 已被删除。\n原因：%s", fqdn, body.Reason)
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
		} else {
			event := SSEEvent{Event: "new_notification", Data: fmt.Sprintf(`{"id":%d}`, notification.ID)}
			h.broker.SendToUsers([]uint{sub.UserID}, event)
		}
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"fqdn":    record.Name,
		"type":    record.Type,
		"content": record.Content,
		"notify":  body.Notify,
	})
	h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_delete_dns_record",
		Resource:   "dns_record",
		ResourceID: record.ID,
		Details:    details,
	})

	response.OK(c, gin.H{"message": "record deleted"})
}
