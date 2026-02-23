package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/repository"
	"hl6-server/internal/ctxutil"
	"hl6-server/pkg/response"
)

type AuthHandler struct {
	repo *repository.Repository
}

func NewAuthHandler(repo *repository.Repository) *AuthHandler {
	return &AuthHandler{repo: repo}
}

func (h *AuthHandler) Me(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	balance, _ := h.repo.EnsureCreditBalance(user.ID)
	response.OK(c, gin.H{
		"user":    user,
		"credits": balance.Balance,
	})
}
