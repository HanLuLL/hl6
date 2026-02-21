package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type CreditHandler struct {
	repo *repository.Repository
}

func NewCreditHandler(repo *repository.Repository) *CreditHandler {
	return &CreditHandler{repo: repo}
}

func (h *CreditHandler) GetBalance(c *gin.Context) {
	user := h.getUser(c)
	if user == nil {
		return
	}
	balance, _ := h.repo.EnsureCreditBalance(user.ID)
	txns, _, _ := h.repo.ListTransactions(user.ID, 1, 10)
	response.OK(c, gin.H{
		"balance":      balance.Balance,
		"transactions": txns,
	})
}

func (h *CreditHandler) ListTransactions(c *gin.Context) {
	user := h.getUser(c)
	if user == nil {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	txns, total, err := h.repo.ListTransactions(user.ID, page, perPage)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list transactions")
		return
	}
	response.Paginated(c, txns, total, page, perPage)
}

func (h *CreditHandler) AdminGrant(c *gin.Context) {
	var body struct {
		UserID      uint   `json:"user_id" binding:"required"`
		Amount      int    `json:"amount" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Amount <= 0 {
		response.Error(c, http.StatusBadRequest, "amount must be positive")
		return
	}
	if body.Description == "" {
		body.Description = "Admin grant"
	}

	var user model.User
	if err := h.repo.DB.First(&user, body.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "user not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "database error")
		return
	}

	tx := h.repo.DB.Begin()
	if err := h.repo.GrantCredits(tx, body.UserID, body.Amount, body.Description); err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "failed to grant credits")
		return
	}
	tx.Commit()

	balance, _ := h.repo.GetCreditBalance(body.UserID)
	response.OK(c, gin.H{
		"user_id": body.UserID,
		"granted": body.Amount,
		"balance": balance.Balance,
	})
}

func (h *CreditHandler) getUser(c *gin.Context) *model.User {
	logtoID := c.GetString("user_id")
	user, err := h.repo.FindUserByLogtoID(logtoID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "user not found")
		c.Abort()
		return nil
	}
	return user
}
