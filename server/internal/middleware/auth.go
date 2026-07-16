package middleware

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"hl6-server/internal/clientauth"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

const sessionVersionClaim = "hl6_session_version"

type AuthMiddleware struct {
	sessionKey      jwk.Key
	repo            *repository.Repository
	trustedOrigins  map[string]struct{}
}

func allowBannedAccess(method, path string) bool {
	// 允许登出
	if method == http.MethodPost && path == "/api/v1/auth/logout" {
		return true
	}
	// Allow the shell to resolve the signed-in identity for the restricted ban page.
	if method == http.MethodGet && path == "/api/v1/auth/me" {
		return true
	}
	// 允许获取封禁信息
	if method == http.MethodGet && path == "/api/v1/ban-info" {
		return true
	}
	// 允许查看和提交申诉
	if method == http.MethodGet && path == "/api/v1/appeals" {
		return true
	}
	if method == http.MethodPost && path == "/api/v1/appeals" {
		return true
	}
	return false
}

func clearSessionCookie(c *gin.Context) {
	for _, secure := range []bool{false, true} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "hl6_session",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			Secure:   secure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func NewAuthMiddleware(sessionSecret string, repo *repository.Repository, trustedOrigins []string) *AuthMiddleware {
	key, err := jwk.FromRaw([]byte(sessionSecret))
	if err != nil {
		panic("invalid session secret: " + err.Error())
	}
	originSet := make(map[string]struct{}, len(trustedOrigins))
	for _, origin := range trustedOrigins {
		if normalized := strings.TrimRight(strings.TrimSpace(origin), "/"); normalized != "" {
			originSet[normalized] = struct{}{}
		}
	}
	return &AuthMiddleware{sessionKey: key, repo: repo, trustedOrigins: originSet}
}

func (a *AuthMiddleware) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := c.Cookie("hl6_session")
		cookieSession := err == nil && sessionToken != ""
		if err != nil || sessionToken == "" {
			authorization := c.GetHeader("Authorization")
			if len(authorization) > len("Bearer ") && authorization[:len("Bearer ")] == "Bearer " {
				sessionToken = authorization[len("Bearer "):]
			}
		}
		if sessionToken == "" {
			response.ErrorWithKey(c, http.StatusUnauthorized, "not authenticated", "error.missingToken")
			c.Abort()
			return
		}

		parsed, err := jwt.Parse([]byte(sessionToken),
			jwt.WithKey(jwa.HS256, a.sessionKey),
			jwt.WithValidate(true),
			jwt.WithIssuer("hl6"),
			jwt.WithAudience("hl6"),
		)
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid session", "error.invalidToken")
			c.Abort()
			return
		}
		if nativeClient, _ := parsed.Get(clientauth.NativeSessionClaim); nativeClient == true {
			sessionKeyHash, ok := parsed.Get(clientauth.NativeSessionKeyHashClaim)
			if !ok || !a.isAuthorizedNativeClientKey(c.GetHeader("X-HL6-Client-Key"), sessionKeyHash) {
				response.ErrorWithKey(c, http.StatusUnauthorized, "invalid client key", "error.invalidToken")
				c.Abort()
				return
			}
		}

		userID, err := parseSessionSubject(parsed.Subject())
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid session", "error.invalidToken")
			c.Abort()
			return
		}
		rawSessionVersion, ok := parsed.Get(sessionVersionClaim)
		if !ok {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid session", "error.invalidToken")
			c.Abort()
			return
		}
		sessionVersion, err := parseSessionVersion(rawSessionVersion)
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid session", "error.invalidToken")
			c.Abort()
			return
		}
		credential, err := a.repo.FindCredentialByUserID(userID)
		if err != nil || credential.SessionVersion != sessionVersion {
			response.ErrorWithKey(c, http.StatusUnauthorized, "session is no longer valid", "error.invalidToken")
			c.Abort()
			return
		}
		if cookieSession && isUnsafeMethod(c.Request.Method) && !a.isTrustedBrowserOrigin(c) {
			response.ErrorWithKey(c, http.StatusForbidden, "untrusted browser origin", "error.untrustedOrigin")
			c.Abort()
			return
		}
		c.Set("user_id", userID)

		user, err := a.repo.FindUserByID(userID)
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
			c.Abort()
			return
		}
		if user.IsBanned && user.BannedUntil != nil && !user.BannedUntil.After(time.Now()) {
			if err := a.repo.UnbanUser(user.ID); err != nil {
				response.ErrorWithKey(c, http.StatusInternalServerError, "failed to restore expired ban", "error.databaseError")
				c.Abort()
				return
			}
			user.IsBanned = false
			user.BannedReason = ""
			user.BannedAt = nil
			user.BannedUntil = nil
		}
		if user.IsBanned && !allowBannedAccess(c.Request.Method, c.Request.URL.Path) {
			response.ErrorWithKeyData(c, http.StatusForbidden, "user is banned", "error.userBanned", gin.H{
				"reason":       user.BannedReason,
				"banned_at":    user.BannedAt,
				"banned_until": user.BannedUntil,
			})
			c.Abort()
			return
		}
		ctxutil.SetUser(c, user)

		c.Next()
	}
}

func parseSessionSubject(subject string) (uint, error) {
	if subject == "" {
		return 0, errors.New("session subject is empty")
	}
	parsed, err := strconv.ParseUint(subject, 10, 64)
	if err != nil || parsed == 0 || parsed > uint64(^uint(0)) {
		return 0, errors.New("session subject is invalid")
	}
	return uint(parsed), nil
}

func parseSessionVersion(value interface{}) (uint, error) {
	var parsed uint64
	switch typed := value.(type) {
	case uint:
		parsed = uint64(typed)
	case uint8:
		parsed = uint64(typed)
	case uint16:
		parsed = uint64(typed)
	case uint32:
		parsed = uint64(typed)
	case uint64:
		parsed = typed
	case int:
		if typed < 0 {
			return 0, errors.New("session version is invalid")
		}
		parsed = uint64(typed)
	case int8:
		if typed < 0 {
			return 0, errors.New("session version is invalid")
		}
		parsed = uint64(typed)
	case int16:
		if typed < 0 {
			return 0, errors.New("session version is invalid")
		}
		parsed = uint64(typed)
	case int32:
		if typed < 0 {
			return 0, errors.New("session version is invalid")
		}
		parsed = uint64(typed)
	case int64:
		if typed < 0 {
			return 0, errors.New("session version is invalid")
		}
		parsed = uint64(typed)
	case float64:
		if typed != math.Trunc(typed) || typed < 0 || typed > float64(^uint64(0)) {
			return 0, errors.New("session version is invalid")
		}
		parsed = uint64(typed)
	case json.Number:
		value, err := strconv.ParseUint(string(typed), 10, 64)
		if err != nil {
			return 0, errors.New("session version is invalid")
		}
		parsed = value
	default:
		return 0, errors.New("session version is invalid")
	}
	if parsed == 0 || parsed > uint64(^uint(0)) {
		return 0, errors.New("session version is invalid")
	}
	return uint(parsed), nil
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func (a *AuthMiddleware) isTrustedBrowserOrigin(c *gin.Context) bool {
	// Cookie-backed mutations always need an explicit origin. Native sessions
	// use bearer tokens and pass the separate Android communication-key check.
	origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")
	if origin == "" {
		return false
	}
	_, trusted := a.trustedOrigins[origin]
	return trusted
}

func (a *AuthMiddleware) isAuthorizedNativeClientKey(key string, sessionKeyHash interface{}) bool {
	claimedHash, ok := sessionKeyHash.(string)
	if !ok {
		return false
	}
	storedHash, err := a.repo.GetSystemConfig(clientauth.CommunicationKeyHashConfigKey)
	if err != nil {
		return false
	}
	return clientauth.SameHash(claimedHash, storedHash) && clientauth.IsAuthorized(key, storedHash)
}
