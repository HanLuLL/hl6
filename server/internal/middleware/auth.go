package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"hl6-server/pkg/response"
)

type AuthMiddleware struct {
	jwksCache *jwk.Cache
	endpoint  string
	resource  string
}

func NewAuthMiddleware(logtoEndpoint, apiResource string) *AuthMiddleware {
	ctx := context.Background()
	jwksURL := logtoEndpoint + "/oidc/jwks"
	c := jwk.NewCache(ctx)
	c.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute))
	c.Refresh(ctx, jwksURL)

	return &AuthMiddleware{
		jwksCache: c,
		endpoint:  logtoEndpoint,
		resource:  apiResource,
	}
}

func (a *AuthMiddleware) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, http.StatusUnauthorized, "missing authorization token")
			c.Abort()
			return
		}

		jwksURL := a.endpoint + "/oidc/jwks"
		keySet, err := a.jwksCache.Get(c.Request.Context(), jwksURL)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "failed to fetch JWKS")
			c.Abort()
			return
		}

		parsed, err := jwt.Parse([]byte(token),
			jwt.WithKeySet(keySet),
			jwt.WithIssuer(a.endpoint+"/oidc"),
			jwt.WithAudience(a.resource),
		)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		c.Set("user_id", parsed.Subject())
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}
