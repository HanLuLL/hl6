package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

const (
	clientCommunicationKeyHashConfigKey = "client_communication_key_hash"
	clientLatestVersionConfigKey         = "client_latest_version"
	clientForceUpdateConfigKey           = "client_force_update"
	clientUpdateNoticeConfigKey          = "client_update_notice"
	clientUpdateURLConfigKey             = "client_update_url"
)

var clientVersionPattern = regexp.MustCompile(`^[0-9A-Za-z.+-]{1,32}$`)

type ClientHandler struct {
	repo *repository.Repository
}

func NewClientHandler(repo *repository.Repository) *ClientHandler {
	return &ClientHandler{repo: repo}
}

func (h *ClientHandler) GetVersion(c *gin.Context) {
	if !h.isAuthorized(c.GetHeader("X-HL6-Client-Key")) {
		response.ErrorWithKey(c, http.StatusUnauthorized, "invalid client key", "error.invalidToken")
		return
	}
	configs, err := h.repo.GetSystemConfigsByKeys(clientConfigKeys())
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get client version", "error.databaseError")
		return
	}
	response.OK(c, clientConfigResponse(configs, false))
}

func (h *ClientHandler) ValidatePresentedKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-HL6-Client-Key")
		if key != "" && !h.isAuthorized(key) {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid client key", "error.invalidToken")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *ClientHandler) GetAdminConfig(c *gin.Context) {
	configs, err := h.repo.GetSystemConfigsByKeys(clientConfigKeys())
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get client config", "error.databaseError")
		return
	}
	response.OK(c, clientConfigResponse(configs, true))
}

func (h *ClientHandler) UpdateAdminConfig(c *gin.Context) {
	var body struct {
		LatestVersion *string `json:"latest_version"`
		ForceUpdate   *bool   `json:"force_update"`
		UpdateNotice  *string `json:"update_notice"`
		UpdateURL     *string `json:"update_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	if body.LatestVersion != nil {
		version := strings.TrimSpace(*body.LatestVersion)
		if !clientVersionPattern.MatchString(version) {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid client version", "error.invalidRequestBody")
			return
		}
		if err := h.repo.SetSystemConfig(clientLatestVersionConfigKey, version); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update client version", "error.databaseError")
			return
		}
	}
	if body.ForceUpdate != nil {
		if err := h.repo.SetSystemConfig(clientForceUpdateConfigKey, boolConfigValue(*body.ForceUpdate)); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update client config", "error.databaseError")
			return
		}
	}
	if body.UpdateNotice != nil {
		notice := strings.TrimSpace(*body.UpdateNotice)
		if len([]rune(notice)) > 2000 {
			response.ErrorWithKey(c, http.StatusBadRequest, "update notice is too long", "error.invalidRequestBody")
			return
		}
		if err := h.repo.SetSystemConfig(clientUpdateNoticeConfigKey, notice); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update client config", "error.databaseError")
			return
		}
	}
	if body.UpdateURL != nil {
		updateURL := strings.TrimSpace(*body.UpdateURL)
		if updateURL != "" && !isHTTPSURL(updateURL) {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid update url", "error.invalidRequestBody")
			return
		}
		if err := h.repo.SetSystemConfig(clientUpdateURLConfigKey, updateURL); err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update client config", "error.databaseError")
			return
		}
	}
	response.OK(c, gin.H{"message": "client config updated"})
}

func (h *ClientHandler) GenerateCommunicationKey(c *gin.Context) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to generate client key", "error.databaseError")
		return
	}
	key := base64.RawURLEncoding.EncodeToString(keyBytes)
	hash := sha256.Sum256([]byte(key))
	if err := h.repo.SetSystemConfig(clientCommunicationKeyHashConfigKey, base64.RawStdEncoding.EncodeToString(hash[:])); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save client key", "error.databaseError")
		return
	}
	response.OK(c, gin.H{"communication_key": key})
}

func (h *ClientHandler) isAuthorized(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	storedHash, err := h.repo.GetSystemConfig(clientCommunicationKeyHashConfigKey)
	if err != nil || storedHash == "" {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(storedHash)
	if err != nil {
		return false
	}
	actual := sha256.Sum256([]byte(key))
	return subtle.ConstantTimeCompare(actual[:], expected) == 1
}

func clientConfigKeys() []string {
	return []string{
		clientCommunicationKeyHashConfigKey,
		clientLatestVersionConfigKey,
		clientForceUpdateConfigKey,
		clientUpdateNoticeConfigKey,
		clientUpdateURLConfigKey,
	}
}

func clientConfigResponse(configs map[string]string, includeKeyStatus bool) gin.H {
	data := gin.H{
		"latest_version": configs[clientLatestVersionConfigKey],
		"force_update":   configs[clientForceUpdateConfigKey] == "true",
		"update_notice":  configs[clientUpdateNoticeConfigKey],
		"update_url":     configs[clientUpdateURLConfigKey],
	}
	if includeKeyStatus {
		data["communication_key_configured"] = configs[clientCommunicationKeyHashConfigKey] != ""
	}
	return data
}

func isHTTPSURL(rawURL string) bool {
	parsed, err := url.ParseRequestURI(rawURL)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func boolConfigValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
