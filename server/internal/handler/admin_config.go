package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/model"
	"hl6-server/internal/oidc"
	"hl6-server/pkg/crypto"
	"hl6-server/pkg/response"
)

var allowedConfigKeys = map[string]bool{
	"registration_bonus_credits": true,
	"referral_enabled":           true,
	"referral_inviter_credits":   true,
	"referral_invitee_credits":   true,
	"daily_checkin_enabled":      true,
	"daily_checkin_credits":      true,
	"daily_checkin_group_ids":    true,
	"frontend_urls":              true,
	"frontend_url":               true,
	"backend_urls":               true,
	"backend_url":                true,
	"oidc_issuer":                true,
	"oidc_client_id":             true,
	"oidc_client_secret":         true,
}

func (h *AdminHandler) GetConfig(c *gin.Context) {
	keys := make([]string, 0, len(allowedConfigKeys))
	for key := range allowedConfigKeys {
		keys = append(keys, key)
	}
	configs, err := h.repo.GetSystemConfigsByKeys(keys)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get config", "error.failedToGetConfig")
		return
	}
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime url config", "error.failedToGetConfig")
		return
	}
	oidcState, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime oidc config", "error.failedToGetConfig")
		return
	}

	// Never return client secret value to frontend.
	delete(configs, configKeyOIDCClientSecret)

	response.OK(c, gin.H{
		"values": configs,
		"url_runtime": gin.H{
			"frontend_urls":       urlState.FrontendURLs,
			"backend_urls":        urlState.BackendURLs,
			"frontend_url":        urlState.FrontendURL,
			"backend_url":         urlState.BackendURL,
			"frontend_source":     urlState.FrontendSource,
			"backend_source":      urlState.BackendSource,
			"frontend_env_locked": urlState.FrontendEnvLocked,
			"backend_env_locked":  urlState.BackendEnvLocked,
			"confirmed":           urlState.Confirmed,
		},
		"oidc_runtime": gin.H{
			"configured":               oidcState.Configured,
			"missing_fields":           oidcState.MissingFields,
			"issuer":                   oidcState.Issuer,
			"client_id":                oidcState.ClientID,
			"issuer_source":            oidcState.IssuerSource,
			"client_id_source":         oidcState.ClientIDSource,
			"client_secret_source":     oidcState.ClientSecretSource,
			"issuer_env_locked":        oidcState.IssuerEnvLocked,
			"client_id_env_locked":     oidcState.ClientIDEnvLocked,
			"client_secret_env_locked": oidcState.ClientSecretEnvLocked,
			"client_secret_configured": oidcState.ClientSecretConfigured,
		},
	})
}

func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	for key := range body {
		if !allowedConfigKeys[key] {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("config key %q is not allowed", key))
			return
		}
	}

	details := make(map[string]string, len(body)+4)
	normalizedConfigValues := make(map[string]string, 3)

	if raw, ok := pickConfigInput(body, configKeyFrontendURLs, configKeyFrontendURL); ok {
		if h.urlResolver.cfg.FrontendURLEnvSet {
			response.Error(c, http.StatusBadRequest, "frontend_urls is controlled by env and cannot be changed")
			return
		}
		parsed, err := config.ParsePublicURLList(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid frontend_urls")
			return
		}
		if err := persistURLList(h.repo, configKeyFrontendURLs, configKeyFrontendURL, parsed); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
		details[configKeyFrontendURLs] = strings.Join(parsed, ",")
		details[configKeyFrontendURL] = firstOrEmptyString(parsed)
	}

	if raw, ok := pickConfigInput(body, configKeyBackendURLs, configKeyBackendURL); ok {
		if h.urlResolver.cfg.BackendURLEnvSet {
			response.Error(c, http.StatusBadRequest, "backend_urls is controlled by env and cannot be changed")
			return
		}
		parsed, err := config.ParsePublicURLList(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid backend_urls")
			return
		}
		if err := persistURLList(h.repo, configKeyBackendURLs, configKeyBackendURL, parsed); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
		details[configKeyBackendURLs] = strings.Join(parsed, ",")
		details[configKeyBackendURL] = firstOrEmptyString(parsed)
	}

	oidcState, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve oidc config", "error.failedToUpdateConfig")
		return
	}

	oidcChanged := false
	candidate := *oidcState

	if raw, ok := body[configKeyOIDCIssuer]; ok {
		trimmed := strings.TrimSpace(raw)
		if oidcState.IssuerEnvLocked {
			if trimmed != "" && trimmed != oidcState.Issuer {
				response.ErrorWithKey(c, http.StatusBadRequest, "oidc_issuer is controlled by env and cannot be changed", "error.oidcEnvLocked")
				return
			}
		} else {
			candidate.Issuer = trimmed
			oidcChanged = true
			details[configKeyOIDCIssuer] = trimmed
		}
	}

	if raw, ok := body[configKeyOIDCClientID]; ok {
		trimmed := strings.TrimSpace(raw)
		if oidcState.ClientIDEnvLocked {
			if trimmed != "" && trimmed != oidcState.ClientID {
				response.ErrorWithKey(c, http.StatusBadRequest, "oidc_client_id is controlled by env and cannot be changed", "error.oidcEnvLocked")
				return
			}
		} else {
			candidate.ClientID = trimmed
			oidcChanged = true
			details[configKeyOIDCClientID] = trimmed
		}
	}

	secretUpdated := false
	if raw, ok := body[configKeyOIDCClientSecret]; ok {
		trimmed := strings.TrimSpace(raw)
		if oidcState.ClientSecretEnvLocked {
			if trimmed != "" && trimmed != oidcState.ClientSecret {
				response.ErrorWithKey(c, http.StatusBadRequest, "oidc_client_secret is controlled by env and cannot be changed", "error.oidcEnvLocked")
				return
			}
		} else if trimmed != "" {
			candidate.ClientSecret = trimmed
			oidcChanged = true
			secretUpdated = true
			details[configKeyOIDCClientSecret] = "***"
		}
	}

	if oidcChanged {
		missing := collectOIDCMissingFields(&candidate)
		if len(missing) > 0 {
			response.ErrorWithKey(c, http.StatusBadRequest, "oidc config is incomplete", "error.oidcMissingFields")
			return
		}

		issuer, err := validateOIDCIssuer(candidate.Issuer)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid oidc issuer", "error.invalidOIDCIssuer")
			return
		}
		if _, err := oidc.Discover(c.Request.Context(), issuer); err != nil {
			response.ErrorWithKey(c, http.StatusBadGateway, "oidc discovery failed", "error.oidcDiscoveryFailed")
			return
		}

		if !oidcState.IssuerEnvLocked {
			if err := h.repo.SetSystemConfig(configKeyOIDCIssuer, issuer); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
				return
			}
			details[configKeyOIDCIssuer] = issuer
		}
		if !oidcState.ClientIDEnvLocked {
			if err := h.repo.SetSystemConfig(configKeyOIDCClientID, candidate.ClientID); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
				return
			}
		}
		if !oidcState.ClientSecretEnvLocked && secretUpdated {
			encSecret, err := crypto.EncryptIfKey(candidate.ClientSecret, h.cfg.EncryptionKey)
			if err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt oidc client secret", "error.encryptionFailed")
				return
			}
			if err := h.repo.SetSystemConfig(configKeyOIDCClientSecret, encSecret); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
				return
			}
		}
	}

	if raw, ok := body[configKeyDailyCheckinEnabled]; ok {
		normalized, err := normalizeBooleanConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_enabled", "error.invalidDailyCheckinEnabled")
			return
		}
		normalizedConfigValues[configKeyDailyCheckinEnabled] = normalized
		details[configKeyDailyCheckinEnabled] = normalized
	}

	if raw, ok := body["registration_bonus_credits"]; ok {
		normalized, err := normalizeNonNegativeCreditConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid registration_bonus_credits", "error.invalidCreditAmount")
			return
		}
		normalizedConfigValues["registration_bonus_credits"] = normalized
		details["registration_bonus_credits"] = normalized
	}

	if raw, ok := body["referral_inviter_credits"]; ok {
		normalized, err := normalizeNonNegativeCreditConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid referral_inviter_credits", "error.invalidCreditAmount")
			return
		}
		normalizedConfigValues["referral_inviter_credits"] = normalized
		details["referral_inviter_credits"] = normalized
	}

	if raw, ok := body["referral_invitee_credits"]; ok {
		normalized, err := normalizeNonNegativeCreditConfig(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid referral_invitee_credits", "error.invalidCreditAmount")
			return
		}
		normalizedConfigValues["referral_invitee_credits"] = normalized
		details["referral_invitee_credits"] = normalized
	}

	if raw, ok := body[configKeyDailyCheckinCredits]; ok {
		normalized, err := normalizeDailyCheckinCredits(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_credits", "error.invalidDailyCheckinCredits")
			return
		}
		normalizedConfigValues[configKeyDailyCheckinCredits] = normalized
		details[configKeyDailyCheckinCredits] = normalized
	}

	if raw, ok := body[configKeyDailyCheckinGroupIDs]; ok {
		groupIDs, err := parseDailyCheckinGroupIDs(raw)
		if err != nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_group_ids", "error.invalidDailyCheckinGroupIDs")
			return
		}
		if err := validateDailyCheckinGroupsExist(h.repo, groupIDs); err != nil {
			if errors.Is(err, errDailyCheckinGroupsNotFound) {
				response.ErrorWithKey(c, http.StatusBadRequest, "invalid daily_checkin_group_ids", "error.invalidDailyCheckinGroupIDs")
				return
			}
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to validate daily_checkin_group_ids", "error.failedToUpdateConfig")
			return
		}
		normalized := formatDailyCheckinGroupIDs(groupIDs)
		normalizedConfigValues[configKeyDailyCheckinGroupIDs] = normalized
		details[configKeyDailyCheckinGroupIDs] = normalized
	}

	for key, value := range body {
		if isURLConfigKey(key) || isOIDCConfigKey(key) {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if normalized, ok := normalizedConfigValues[key]; ok {
			trimmed = normalized
		}
		if err := h.repo.SetSystemConfig(key, trimmed); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update config", "error.failedToUpdateConfig")
			return
		}
		details[key] = trimmed
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		detailBytes, _ := json.Marshal(details)
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_update_config",
			Resource: "system_config",
			Details:  detailBytes,
		})
	}
	response.OK(c, gin.H{"message": "config updated"})
}

func (h *AdminHandler) ConfirmURLConfig(c *gin.Context) {
	urlState, err := h.urlResolver.Resolve(c)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve runtime url config", "error.failedToUpdateConfig")
		return
	}

	if err := h.repo.SetSystemConfig(configKeyURLConfirmedSignature, urlState.Signature); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save config confirmation", "error.failedToUpdateConfig")
		return
	}

	if admin := ctxutil.GetUser(c); admin != nil {
		details, _ := json.Marshal(map[string]interface{}{
			"frontend_urls":   urlState.FrontendURLs,
			"backend_urls":    urlState.BackendURLs,
			"frontend_url":    urlState.FrontendURL,
			"backend_url":     urlState.BackendURL,
			"frontend_source": urlState.FrontendSource,
			"backend_source":  urlState.BackendSource,
			"signature":       urlState.Signature,
		})
		h.repo.CreateAuditLog(&model.AuditLog{
			UserID:   admin.ID,
			Action:   "admin_confirm_runtime_url_config",
			Resource: "system_config",
			Details:  details,
		})
	}

	response.OK(c, gin.H{"message": "url config confirmed"})
}

func pickConfigInput(body map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			return value, true
		}
	}
	return "", false
}

func isURLConfigKey(key string) bool {
	switch key {
	case configKeyFrontendURL, configKeyFrontendURLs, configKeyBackendURL, configKeyBackendURLs:
		return true
	default:
		return false
	}
}

func isOIDCConfigKey(key string) bool {
	switch key {
	case configKeyOIDCIssuer, configKeyOIDCClientID, configKeyOIDCClientSecret:
		return true
	default:
		return false
	}
}

func firstOrEmptyString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
