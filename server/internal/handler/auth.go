package handler

import (
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

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
		if trimmed == "" || utf8.RuneCountInString(trimmed) > 100 {
			response.ErrorWithKey(c, http.StatusBadRequest, "name cannot be empty", "error.nameCannotBeEmpty")
			return
		}
		user.Name = trimmed
	}
	if body.AvatarURL != nil {
		avatarURL := strings.TrimSpace(*body.AvatarURL)
		if utf8.RuneCountInString(avatarURL) > 2048 || !isHTTPURLOrEmpty(avatarURL) {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid avatar url", "error.invalidRequestBody")
			return
		}
		user.AvatarURL = avatarURL
	}
	if body.Bio != nil {
		bio := strings.TrimSpace(*body.Bio)
		if utf8.RuneCountInString(bio) > 1000 {
			response.ErrorWithKey(c, http.StatusBadRequest, "bio is too long", "error.invalidRequestBody")
			return
		}
		user.Bio = bio
	}
	if body.Website != nil {
		website := strings.TrimSpace(*body.Website)
		if utf8.RuneCountInString(website) > 255 || !isHTTPURLOrEmpty(website) {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid website url", "error.invalidRequestBody")
			return
		}
		user.Website = website
	}

	if err := h.repo.UpdateUser(user); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to update profile", "error.failedToUpdateProfile")
		return
	}

	response.OK(c, gin.H{"user": user})
}

func isHTTPURLOrEmpty(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(rawURL)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
