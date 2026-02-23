package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"hl6-server/internal/repository"
	"hl6-server/internal/ctxutil"
	"hl6-server/pkg/response"
)

type AuthMiddleware struct {
	sessionKey jwk.Key
	repo       *repository.Repository
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

		logtoID := parsed.Subject()
		c.Set("user_id", logtoID)

		user, err := a.repo.FindUserByLogtoID(logtoID)
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
			c.Abort()
			return
		}
		ctxutil.SetUser(c, user)

		c.Next()
	}
}
