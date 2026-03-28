package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/oidc"
	"hl6-server/pkg/crypto"
	"hl6-server/pkg/response"
)

func (h *OIDCHandler) Status(c *gin.Context) {
	state, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get oidc status", "error.failedToGetConfig")
		return
	}

	users, err := h.repo.CountUsers()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to count users", "error.databaseError")
		return
	}

	setupAllowed := users == 0 && !state.Configured
	response.OK(c, buildOIDCStatusPayload(state, setupAllowed))
}

func (h *OIDCHandler) Bootstrap(c *gin.Context) {
	var body struct {
		Issuer       string `json:"oidc_issuer"`
		ClientID     string `json:"oidc_client_id"`
		ClientSecret string `json:"oidc_client_secret"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	users, err := h.repo.CountUsers()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to count users", "error.databaseError")
		return
	}
	if users > 0 {
		response.ErrorWithKey(c, http.StatusForbidden, "oidc bootstrap is not allowed after users are created", "error.oidcBootstrapNotAllowed")
		return
	}

	current, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to resolve oidc config", "error.failedToGetConfig")
		return
	}
	if current.Configured {
		response.ErrorWithKey(c, http.StatusConflict, "oidc is already configured", "error.oidcAlreadyConfigured")
		return
	}

	candidate := *current

	issuerInput := strings.TrimSpace(body.Issuer)
	clientIDInput := strings.TrimSpace(body.ClientID)
	clientSecretInput := strings.TrimSpace(body.ClientSecret)

	if current.IssuerEnvLocked {
		if issuerInput != "" && issuerInput != current.Issuer {
			response.ErrorWithKey(c, http.StatusBadRequest, "oidc_issuer is controlled by env and cannot be changed", "error.oidcEnvLocked")
			return
		}
	} else if issuerInput != "" {
		candidate.Issuer = issuerInput
	}

	if current.ClientIDEnvLocked {
		if clientIDInput != "" && clientIDInput != current.ClientID {
			response.ErrorWithKey(c, http.StatusBadRequest, "oidc_client_id is controlled by env and cannot be changed", "error.oidcEnvLocked")
			return
		}
	} else if clientIDInput != "" {
		candidate.ClientID = clientIDInput
	}

	if current.ClientSecretEnvLocked {
		if clientSecretInput != "" && clientSecretInput != current.ClientSecret {
			response.ErrorWithKey(c, http.StatusBadRequest, "oidc_client_secret is controlled by env and cannot be changed", "error.oidcEnvLocked")
			return
		}
	} else if clientSecretInput != "" {
		candidate.ClientSecret = clientSecretInput
	}

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

	if !current.IssuerEnvLocked {
		if err := h.repo.SetSystemConfig(configKeyOIDCIssuer, issuer); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save oidc config", "error.failedToUpdateConfig")
			return
		}
	}
	if !current.ClientIDEnvLocked {
		if err := h.repo.SetSystemConfig(configKeyOIDCClientID, strings.TrimSpace(candidate.ClientID)); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save oidc config", "error.failedToUpdateConfig")
			return
		}
	}
	if !current.ClientSecretEnvLocked {
		encSecret, err := crypto.EncryptIfKey(candidate.ClientSecret, h.cfg.EncryptionKey)
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to encrypt oidc client secret", "error.encryptionFailed")
			return
		}
		if err := h.repo.SetSystemConfig(configKeyOIDCClientSecret, encSecret); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save oidc config", "error.failedToUpdateConfig")
			return
		}
	}

	latest, err := h.oidcResolver.Resolve()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get oidc status", "error.failedToGetConfig")
		return
	}
	response.OK(c, buildOIDCStatusPayload(latest, false))
}

func buildOIDCStatusPayload(state *OIDCRuntimeState, setupAllowed bool) gin.H {
	return gin.H{
		"configured":               state.Configured,
		"setup_allowed":            setupAllowed,
		"missing_fields":           state.MissingFields,
		"issuer":                   state.Issuer,
		"client_id":                state.ClientID,
		"issuer_source":            state.IssuerSource,
		"client_id_source":         state.ClientIDSource,
		"client_secret_source":     state.ClientSecretSource,
		"issuer_env_locked":        state.IssuerEnvLocked,
		"client_id_env_locked":     state.ClientIDEnvLocked,
		"client_secret_env_locked": state.ClientSecretEnvLocked,
		"client_secret_configured": state.ClientSecretConfigured,
	}
}
