package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/pkg/response"
)

type maintenanceGate interface {
	IsRestoring() bool
}

// MaintenanceMode blocks API traffic during the destructive portion of a
// database restore. The restore-job read endpoint remains available so an
// operator can see the final state before restarting the application.
func MaintenanceMode(gate maintenanceGate) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gate == nil || !gate.IsRestoring() || (c.Request.Method == http.MethodGet && c.Request.URL.Path == "/api/v1/admin/maintenance/restores") {
			c.Next()
			return
		}
		response.ErrorWithKey(c, http.StatusServiceUnavailable, "database maintenance is in progress", "error.databaseMaintenance")
		c.Abort()
	}
}
