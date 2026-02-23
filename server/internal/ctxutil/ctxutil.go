package ctxutil

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/model"
)

const userKey = "ctx_user"

// SetUser stores the user in the gin context.
func SetUser(c *gin.Context, user *model.User) {
	c.Set(userKey, user)
}

// GetUser retrieves the user from the gin context. Returns nil if not set.
func GetUser(c *gin.Context) *model.User {
	v, exists := c.Get(userKey)
	if !exists {
		return nil
	}
	user, ok := v.(*model.User)
	if !ok {
		return nil
	}
	return user
}
