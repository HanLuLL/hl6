package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/ctxutil"
	"hl6-server/pkg/response"
)

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := ctxutil.GetUser(c)
		if user == nil {
			response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
			c.Abort()
			return
		}

		// 管理员判定：role=admin 或者 所属用户组标记为管理员
		if user.Role != "admin" && (user.Group == nil || !user.Group.IsAdmin) {
			response.ErrorWithKey(c, http.StatusForbidden, "admin access required", "error.adminRequired")
			c.Abort()
			return
		}

		c.Next()
	}
}
