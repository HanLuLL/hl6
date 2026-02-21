package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/model"
	"hl6-server/pkg/response"
)

func AdminRequired(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		logtoID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		var user model.User
		if err := db.Where("logto_id = ?", logtoID).First(&user).Error; err != nil {
			response.Error(c, http.StatusForbidden, "user not found")
			c.Abort()
			return
		}

		if user.Role != "admin" {
			response.Error(c, http.StatusForbidden, "admin access required")
			c.Abort()
			return
		}

		c.Set("db_user", user)
		c.Next()
	}
}
