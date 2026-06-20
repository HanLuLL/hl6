package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/apperr"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/model"
	"hl6-server/pkg/response"
)

// currentUser 从上下文返回已认证用户（可能为 nil）。
func currentUser(c *gin.Context) *model.User {
	return ctxutil.GetUser(c)
}

// requireUser 返回已认证用户，否则写入 401 并中止。
func requireUser(c *gin.Context) (*model.User, bool) {
	u := ctxutil.GetUser(c)
	if u == nil {
		response.Unauthorized(c, apperr.KeyUnauthorized)
		c.Abort()
		return nil, false
	}
	return u, true
}

// mustGetUser 从 context 获取用户，若为 nil 则写入 401 响应并返回 nil。
func mustGetUser(c *gin.Context) *model.User {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
	}
	return user
}

// userGroupID 安全提取用户的 GroupID，GroupID 为 nil 时返回 0。
func userGroupID(user *model.User) uint {
	if user.GroupID == nil {
		return 0
	}
	return *user.GroupID
}
