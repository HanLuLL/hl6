package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type AuthMiddleware struct {
	sessionKey jwk.Key
	repo       *repository.Repository
}

func allowBannedAccess(method, path string) bool {
	// 允许登出
	if method == http.MethodPost && path == "/api/v1/auth/logout" {
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

func NewAuthMiddleware(sessionSecret string, repo *repository.Repository) *AuthMiddleware {
	key, err := jwk.FromRaw([]byte(sessionSecret))
	if err != nil {
		panic("invalid session secret: " + err.Error())
	}
	return &AuthMiddleware{sessionKey: key, repo: repo}
}

func (a *AuthMiddleware) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("hl6_session")
		if err != nil || cookie == "" {
			response.ErrorWithKey(c, http.StatusUnauthorized, "not authenticated", "error.missingToken")
			c.Abort()
			return
		}

		parsed, err := jwt.Parse([]byte(cookie),
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

		externalID := parsed.Subject()
		c.Set("user_id", externalID)

		user, err := a.repo.FindUserByExternalID(externalID)
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
			c.Abort()
			return
		}
		if user.IsBanned && !allowBannedAccess(c.Request.Method, c.Request.URL.Path) {
			response.ErrorWithKeyData(c, http.StatusForbidden, "user is banned", "error.userBanned", gin.H{
				"reason": user.BannedReason,
			})
			c.Abort()
			return
		}
		ctxutil.SetUser(c, user)

		c.Next()
	}
}
