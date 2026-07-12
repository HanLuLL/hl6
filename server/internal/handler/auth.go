package handler

import (
	"net/http"
	"strings"

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

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	var body struct {
		Name      *string `json:"name"`
		AvatarURL *string `json:"avatar_url"`
		Bio       *string `json:"bio"`
		Website   *string `json:"website"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	if body.Name != nil {
		trimmed := strings.TrimSpace(*body.Name)
		if trimmed == "" {
			response.ErrorWithKey(c, http.StatusBadRequest, "name cannot be empty", "error.nameCannotBeEmpty")
			return
		}
		user.Name = trimmed
	}
	if body.AvatarURL != nil {
		user.AvatarURL = *body.AvatarURL
	}
	if body.Bio != nil {
		user.Bio = *body.Bio
	}
	if body.Website != nil {
		user.Website = *body.Website
	}

	if err := h.repo.UpdateUser(user); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update profile", "error.failedToUpdateProfile")
		return
	}

	response.OK(c, gin.H{"user": user})
}
