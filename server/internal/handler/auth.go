package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type AuthHandler struct {
	repo *repository.Repository
}

func NewAuthHandler(repo *repository.Repository) *AuthHandler {
	return &AuthHandler{repo: repo}
}

func (h *AuthHandler) Me(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}
	balance, err := h.repo.EnsureCreditBalance(user.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	response.OK(c, gin.H{
		"user":    user,
		"credits": balance.Balance,
	})
}
