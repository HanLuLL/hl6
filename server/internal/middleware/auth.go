package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"hl6-server/pkg/response"
)

type AuthMiddleware struct {
	sessionKey jwk.Key
}

func NewAuthMiddleware(sessionSecret string) *AuthMiddleware {
	key, err := jwk.FromRaw([]byte(sessionSecret))
	if err != nil {
		panic("invalid session secret: " + err.Error())
	}
	return &AuthMiddleware{sessionKey: key}
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
		)
		if err != nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "invalid session", "error.invalidToken")
			c.Abort()
			return
		}

		c.Set("user_id", parsed.Subject())
		c.Next()
	}
}
