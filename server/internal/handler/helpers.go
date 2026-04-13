package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/model"
	"hl6-server/pkg/response"
)

// mustGetUser 从 context 获取用户，若为 nil 则写入 401 响应并返回 nil。
// 调用方检查返回值为 nil 时直接 return。
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
