package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/auth"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/model"
	"hl6-server/pkg/response"
)

const (
	authRegistrationEnabledConfigKey = "auth.registration.enabled"
	authEmailDomainModeConfigKey     = "auth.email_domain.mode"
	authEmailDomainDomainsConfigKey  = "auth.email_domain.domains"
	authLocalEnabledConfigKey        = "auth.local.enabled"
	legacyRegistrationEnabledKey     = "registration_enabled"
)

// AccessSettingsPayload is the complete server-owned registration policy. It
// deliberately exposes no raw SystemConfig map to the browser.
type AccessSettingsPayload struct {
	RegistrationEnabled bool     `json:"registration_enabled"`
	DomainPolicyMode    string   `json:"domain_policy_mode"`
	DomainPolicyDomains []string `json:"domain_policy_domains"`
}

type accessSettingsResponse struct {
	AccessSettingsPayload
	LocalAuthEnabled bool `json:"local_auth_enabled"`
}

func (h *AdminHandler) GetAccessSettings(c *gin.Context) {
	settings, localEnabled, err := h.loadAccessSettings(true)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load access settings", "error.databaseError")
		return
	}
	response.OK(c, accessSettingsResponse{AccessSettingsPayload: settings, LocalAuthEnabled: localEnabled})
}

func (h *AdminHandler) UpdateAccessSettings(c *gin.Context) {
	var payload AccessSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid access settings", "error.invalidRequestBody")
		return
	}
	policy, err := auth.NormalizeDomainPolicy(auth.DomainPolicy{
		Mode:    payload.DomainPolicyMode,
		Domains: payload.DomainPolicyDomains,
	})
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid email domain policy", "error.invalidRequestBody")
		return
	}
	payload.DomainPolicyMode = policy.Mode
	payload.DomainPolicyDomains = policy.Domains
	domainsJSON, err := json.Marshal(policy.Domains)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encode email domain policy", "error.databaseError")
		return
	}

	if err := h.repo.GetDB().Transaction(func(tx *gorm.DB) error {
		for key, value := range map[string]string{
			authRegistrationEnabledConfigKey: boolConfigValue(payload.RegistrationEnabled),
			authEmailDomainModeConfigKey:     policy.Mode,
			authEmailDomainDomainsConfigKey:  string(domainsJSON),
		} {
			var existing model.SystemConfig
			result := tx.Where("\"key\" = ?", key).First(&existing)
			if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			}
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := tx.Create(&model.SystemConfig{Key: key, Value: value}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&existing).Update("value", value).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update access settings", "error.databaseError")
		return
	}
	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(payload)
		_ = h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_update_access_settings",
			Resource: "authentication",
			Details:  details,
		})
	}
	response.OK(c, accessSettingsResponse{AccessSettingsPayload: payload, LocalAuthEnabled: h.localAuthEnabled()})
}

func (h *AdminHandler) ListAuthSecurityEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	events, total, err := h.repo.ListAuthSecurityEvents(page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list authentication security events", "error.databaseError")
		return
	}
	response.Paginated(c, events, total, page, perPage)
}

func (h *AdminHandler) loadAccessSettings(migrateLegacy bool) (AccessSettingsPayload, bool, error) {
	configs, err := h.repo.GetSystemConfigsByKeys([]string{
		authRegistrationEnabledConfigKey,
		authEmailDomainModeConfigKey,
		authEmailDomainDomainsConfigKey,
		authLocalEnabledConfigKey,
		legacyRegistrationEnabledKey,
	})
	if err != nil {
		return AccessSettingsPayload{}, false, err
	}
	settings := AccessSettingsPayload{
		RegistrationEnabled: true,
		DomainPolicyMode:    auth.DomainPolicyUnrestricted,
		DomainPolicyDomains: []string{},
	}
	legacyUsed := false
	if raw, exists := configs[authRegistrationEnabledConfigKey]; exists {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return AccessSettingsPayload{}, false, parseErr
		}
		settings.RegistrationEnabled = parsed
	} else if raw, exists := configs[legacyRegistrationEnabledKey]; exists {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return AccessSettingsPayload{}, false, parseErr
		}
		settings.RegistrationEnabled = parsed
		legacyUsed = true
	}
	if raw := strings.TrimSpace(configs[authEmailDomainModeConfigKey]); raw != "" {
		settings.DomainPolicyMode = raw
	}
	if raw := strings.TrimSpace(configs[authEmailDomainDomainsConfigKey]); raw != "" {
		if err := json.Unmarshal([]byte(raw), &settings.DomainPolicyDomains); err != nil {
			return AccessSettingsPayload{}, false, err
		}
	}
	policy, err := auth.NormalizeDomainPolicy(auth.DomainPolicy{Mode: settings.DomainPolicyMode, Domains: settings.DomainPolicyDomains})
	if err != nil {
		return AccessSettingsPayload{}, false, err
	}
	settings.DomainPolicyMode = policy.Mode
	settings.DomainPolicyDomains = policy.Domains
	if migrateLegacy && legacyUsed {
		if err := h.repo.SetSystemConfig(authRegistrationEnabledConfigKey, boolConfigValue(settings.RegistrationEnabled)); err != nil {
			return AccessSettingsPayload{}, false, err
		}
	}
	return settings, configs[authLocalEnabledConfigKey] == "true", nil
}

func (h *AdminHandler) localAuthEnabled() bool {
	value, err := h.repo.GetSystemConfig(authLocalEnabledConfigKey)
	return err == nil && value == "true"
}
