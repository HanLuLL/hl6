package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/clientauth"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

const (
	clientLatestVersionConfigKey         = "client_latest_version"
	clientForceUpdateConfigKey           = "client_force_update"
	clientUpdateNoticeConfigKey          = "client_update_notice"
	clientUpdateURLConfigKey             = "client_update_url"
)

var clientVersionPattern = regexp.MustCompile(`^(?:0|[1-9][0-9]{0,2})\.(?:0|[1-9][0-9]{0,2})\.(?:0|[1-9][0-9]{0,2})$`)

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
	currentVersion := strings.TrimSpace(c.Query("current_version"))
	if currentVersion != "" && !clientVersionPattern.MatchString(currentVersion) {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid client version", "error.invalidRequestBody")
		return
	}
	latestVersion := configs[clientLatestVersionConfigKey]
	updateAvailable := currentVersion != "" && clientVersionPattern.MatchString(latestVersion) && compareClientVersions(currentVersion, latestVersion) < 0
	response.OK(c, clientConfigResponse(configs, false, updateAvailable))
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

func (h *ClientHandler) RequireClientKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.isAuthorized(c.GetHeader("X-HL6-Client-Key")) {
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
	response.OK(c, clientConfigResponse(configs, true, false))
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
	configs, err := h.repo.GetSystemConfigsByKeys([]string{clientForceUpdateConfigKey, clientUpdateURLConfigKey})
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get client config", "error.databaseError")
		return
	}
	effectiveForceUpdate := configs[clientForceUpdateConfigKey] == "true"
	if body.ForceUpdate != nil {
		effectiveForceUpdate = *body.ForceUpdate
	}
	effectiveUpdateURL := strings.TrimSpace(configs[clientUpdateURLConfigKey])
	if body.UpdateURL != nil {
		effectiveUpdateURL = strings.TrimSpace(*body.UpdateURL)
	}
	if effectiveForceUpdate && !isHTTPSURL(effectiveUpdateURL) {
		response.ErrorWithKey(c, http.StatusBadRequest, "force update requires an https update url", "error.invalidRequestBody")
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
	h.writeClientConfigAudit(c, body.LatestVersion != nil, body.ForceUpdate != nil, body.UpdateNotice != nil, body.UpdateURL != nil)
	response.OK(c, gin.H{"message": "client config updated"})
}

func (h *ClientHandler) GenerateCommunicationKey(c *gin.Context) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to generate client key", "error.databaseError")
		return
	}
	key := base64.RawURLEncoding.EncodeToString(keyBytes)
	if err := h.repo.SetSystemConfig(clientauth.CommunicationKeyHashConfigKey, clientauth.HashCommunicationKey(key)); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to save client key", "error.databaseError")
		return
	}
	h.writeCommunicationKeyAudit(c, "admin_generate_client_communication_key")
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	response.OK(c, gin.H{"communication_key": key})
}

func (h *ClientHandler) RevokeCommunicationKey(c *gin.Context) {
	if err := h.repo.SetSystemConfig(clientauth.CommunicationKeyHashConfigKey, ""); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to revoke client key", "error.databaseError")
		return
	}
	h.writeCommunicationKeyAudit(c, "admin_revoke_client_communication_key")
	response.OK(c, gin.H{"message": "client communication key revoked"})
}

func (h *ClientHandler) isAuthorized(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	storedHash, err := h.repo.GetSystemConfig(clientauth.CommunicationKeyHashConfigKey)
	if err != nil {
		return false
	}
	return clientauth.IsAuthorized(key, storedHash)
}

func clientConfigKeys() []string {
	return []string{
		clientauth.CommunicationKeyHashConfigKey,
		clientLatestVersionConfigKey,
		clientForceUpdateConfigKey,
		clientUpdateNoticeConfigKey,
		clientUpdateURLConfigKey,
	}
}

func clientConfigResponse(configs map[string]string, includeKeyStatus, updateAvailable bool) gin.H {
	data := gin.H{
		"latest_version": configs[clientLatestVersionConfigKey],
		"force_update":   configs[clientForceUpdateConfigKey] == "true",
		"update_notice":  configs[clientUpdateNoticeConfigKey],
		"update_url":     configs[clientUpdateURLConfigKey],
		"update_available": updateAvailable,
	}
	if includeKeyStatus {
		data["communication_key_configured"] = configs[clientauth.CommunicationKeyHashConfigKey] != ""
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

func (h *ClientHandler) writeCommunicationKeyAudit(c *gin.Context, action string) {
	admin := mustGetUser(c)
	if admin == nil {
		return
	}
	details, _ := json.Marshal(map[string]string{"key": "redacted"})
	_ = h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   action,
		Resource: "client_communication_key",
		Details:  details,
	})
}

func (h *ClientHandler) writeClientConfigAudit(c *gin.Context, version, forceUpdate, notice, updateURL bool) {
	if !version && !forceUpdate && !notice && !updateURL {
		return
	}
	admin := mustGetUser(c)
	if admin == nil {
		return
	}
	details, _ := json.Marshal(map[string]bool{
		"latest_version_changed": version,
		"force_update_changed":   forceUpdate,
		"update_notice_changed":  notice,
		"update_url_changed":     updateURL,
	})
	_ = h.repo.CreateAuditLog(&model.AuditLog{
		UserID:   admin.ID,
		Action:   "admin_update_client_config",
		Resource: "client_config",
		Details:  details,
	})
}

func compareClientVersions(current, latest string) int {
	currentCore := splitClientVersion(current)
	latestCore := splitClientVersion(latest)
	for index := range currentCore {
		if currentCore[index] != latestCore[index] {
			return currentCore[index] - latestCore[index]
		}
	}
	return 0
}

func splitClientVersion(version string) [3]int {
	coreParts := strings.Split(version, ".")
	var core [3]int
	for index, value := range coreParts {
		if index >= len(core) {
			break
		}
		core[index], _ = strconv.Atoi(value)
	}
	return core
}
