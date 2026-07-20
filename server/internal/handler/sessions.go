package handler

import (
	"net/http"
	"strconv"

	"hl6-server/internal/auth"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/model"
	"hl6-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	repo SessionRepository
}

type SessionRepository interface {
	ListUserSessions(userID uint) ([]model.UserSession, error)
	DeleteUserSession(userID, sessionID uint) error
	DeleteUserSessionByJTI(userID uint, jtiHash string) error
	DeleteAllUserSessions(userID uint) error
}

func NewSessionHandler(repo SessionRepository) *SessionHandler {
	return &SessionHandler{repo: repo}
}

// ListSessions 列出当前用户的所有活跃会话
func (h *SessionHandler) ListSessions(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "not authenticated", "error.missingToken")
		return
	}

	sessions, err := h.repo.ListUserSessions(user.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list sessions", "error.databaseError")
		return
	}

	// 获取当前会话的 JTI，标记当前设备
	currentJTI, _ := c.Get("session_jti")
	currentJTIHash := ""
	if currentJTI != nil {
		currentJTIHash = currentJTI.(string)
	}

	type sessionResponse struct {
		ID           uint   `json:"id"`
		DeviceName   string `json:"device_name"`
		DeviceType   string `json:"device_type"`
		LastActiveAt string `json:"last_active_at"`
		ExpiresAt    string `json:"expires_at"`
		IsCurrent    bool   `json:"is_current"`
	}

	result := make([]sessionResponse, len(sessions))
	for i, s := range sessions {
		result[i] = sessionResponse{
			ID:           s.ID,
			DeviceName:   s.DeviceName,
			DeviceType:   s.DeviceType,
			LastActiveAt: s.LastActiveAt.Format("2006-01-02T15:04:05Z07:00"),
			ExpiresAt:    s.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			IsCurrent:    s.SessionJTI == currentJTIHash,
		}
	}

	response.OK(c, result)
}

// DeleteSession 踢出指定设备
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "not authenticated", "error.missingToken")
		return
	}

	sessionIDStr := c.Param("id")
	if sessionIDStr == "" {
		response.ErrorWithKey(c, http.StatusBadRequest, "session id is required", "error.invalidRequestBody")
		return
	}

	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid session id", "error.invalidRequestBody")
		return
	}

	if err := h.repo.DeleteUserSession(user.ID, uint(sessionID)); err != nil {
		response.ErrorWithKey(c, http.StatusNotFound, "session not found", "error.sessionNotFound")
		return
	}

	response.OK(c, gin.H{"message": "session deleted"})
}

// LogoutAll 登出所有设备
func (h *SessionHandler) LogoutAll(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "not authenticated", "error.missingToken")
		return
	}

	if err := h.repo.DeleteAllUserSessions(user.ID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to logout all sessions", "error.databaseError")
		return
	}

	response.OK(c, gin.H{"message": "all sessions logged out"})
}

// HashJTI 计算 JTI 的 SHA-256 哈希
func HashJTI(jti string) string {
	if jti == "" {
		return ""
	}
	return auth.HashToken(jti)
}