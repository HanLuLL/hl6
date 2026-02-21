package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type AuthHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewAuthHandler(repo *repository.Repository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{repo: repo, cfg: cfg}
}

func (h *AuthHandler) Me(c *gin.Context) {
	logtoID := c.GetString("user_id")
	user, err := h.repo.FindUserByLogtoID(logtoID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": -1, "message": "user not found, please sync first"})
		return
	}
	balance, _ := h.repo.EnsureCreditBalance(user.ID)
	response.OK(c, gin.H{
		"user":    user,
		"credits": balance.Balance,
	})
}

func (h *AuthHandler) Sync(c *gin.Context) {
	logtoID := c.GetString("user_id")

	var body struct {
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.repo.FindUserByLogtoID(logtoID)
	if err == gorm.ErrRecordNotFound {
		user = &model.User{
			LogtoID:   logtoID,
			Email:     body.Email,
			Name:      body.Name,
			AvatarURL: body.AvatarURL,
			Role:      "user",
		}
		if h.cfg.IsAdminEmail(body.Email) {
			user.Role = "admin"
		}

		// Assign default user group
		if defaultGroup, err := h.repo.GetDefaultUserGroup(); err == nil {
			user.GroupID = &defaultGroup.ID
		}

		if err := h.repo.CreateUser(user); err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to create user")
			return
		}
		h.repo.EnsureCreditBalance(user.ID)

		// Grant registration bonus credits
		if bonusStr, err := h.repo.GetSystemConfig("registration_bonus_credits"); err == nil {
			if bonus, err := strconv.Atoi(bonusStr); err == nil && bonus > 0 {
				tx := h.repo.DB.Begin()
				if err := h.repo.GrantCredits(tx, user.ID, bonus, "Registration bonus"); err != nil {
					tx.Rollback()
				} else {
					tx.Commit()
				}
			}
		}

		// Reload user with group
		user, _ = h.repo.FindUserByLogtoID(logtoID)
		response.Created(c, user)
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	user.Email = body.Email
	user.Name = body.Name
	user.AvatarURL = body.AvatarURL
	if h.cfg.IsAdminEmail(body.Email) {
		user.Role = "admin"
	}
	h.repo.UpdateUser(user)
	response.OK(c, user)
}
