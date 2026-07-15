package middleware

import (
	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = true
	}
	// Capacitor's local Android origin. Native requests still require the
	// communication key and native bearer session, so this does not grant API access.
	originSet["http://localhost"] = true
	originSet["https://localhost"] = true
	originSet["capacitor://localhost"] = true

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && originSet[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-HL6-Client-Key")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
