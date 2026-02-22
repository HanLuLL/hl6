package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
	"hl6-server/pkg/validator"
)

type DNSHandler struct {
	repo *repository.Repository
	cf   *service.CloudflareService
}

func NewDNSHandler(repo *repository.Repository, cf *service.CloudflareService) *DNSHandler {
	return &DNSHandler{repo: repo, cf: cf}
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
	user := h.getUserFromContext(c)
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

	cfID, err := h.cf.CreateRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, body.Type, sub.FQDN, body.Content, body.TTL, body.Proxied)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadGateway, fmt.Sprintf("cloudflare error: %v", err), "error.cloudflareError")
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
	recordID, _ := strconv.ParseUint(c.Param("recordId"), 10, 64)
	record, err := h.repo.FindDNSRecord(uint(recordID))
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

	if err := h.cf.UpdateRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID, record.Type, sub.FQDN, body.Content, body.TTL, body.Proxied); err != nil {
		response.ErrorWithKey(c, http.StatusBadGateway, fmt.Sprintf("cloudflare error: %v", err), "error.cloudflareError")
		return
	}

	record.Content = body.Content
	record.TTL = body.TTL
	record.Proxied = body.Proxied
	h.repo.UpdateDNSRecord(record)
	response.OK(c, record)
}

func (h *DNSHandler) DeleteRecord(c *gin.Context) {
	sub := h.getSubdomain(c)
	if sub == nil {
		return
	}
	recordID, _ := strconv.ParseUint(c.Param("recordId"), 10, 64)
	record, err := h.repo.FindDNSRecord(uint(recordID))
	if err != nil || record.SubdomainID != sub.ID {
		response.ErrorWithKey(c, http.StatusNotFound, "record not found", "error.recordNotFound")
		return
	}

	if record.CloudflareRecordID != "" {
		if err := h.cf.DeleteRecord(c.Request.Context(), sub.Domain.CloudflareZoneID, record.CloudflareRecordID); err != nil {
			response.ErrorWithKey(c, http.StatusBadGateway, fmt.Sprintf("cloudflare error: %v", err), "error.cloudflareError")
			return
		}
	}

	h.repo.DeleteDNSRecord(record.ID)
	response.OK(c, gin.H{"message": "record deleted"})
}

func (h *DNSHandler) getSubdomain(c *gin.Context) *model.Subdomain {
	user := h.getUserFromContext(c)
	if user == nil {
		return nil
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := h.repo.FindSubdomain(uint(id))
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

func (h *DNSHandler) getUserFromContext(c *gin.Context) *model.User {
	logtoID := c.GetString("user_id")
	user, err := h.repo.FindUserByLogtoID(logtoID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		c.Abort()
		return nil
	}
	return user
}
