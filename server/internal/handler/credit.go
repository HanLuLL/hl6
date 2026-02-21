package handler

import (
	"encoding/json"
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
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list transactions", "error.failedToListTransactions")
		return
	}
	response.Paginated(c, txns, total, page, perPage)
}

func (h *CreditHandler) AdminGrant(c *gin.Context) {
	var body struct {
		UserID      uint    `json:"user_id" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
		Description string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	if body.Amount <= 0 {
		response.ErrorWithKey(c, http.StatusBadRequest, "amount must be positive", "error.amountMustBePositive")
		return
	}

	descKey := "txn.adminGrant"
	var descParams json.RawMessage
	if body.Description != "" {
		descKey = "txn.adminGrantCustom"
		descParams, _ = json.Marshal(map[string]string{"description": body.Description})
	}

	var user model.User
	if err := h.repo.DB.First(&user, body.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorWithKey(c, http.StatusNotFound, "user not found", "error.userNotFound")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	tx := h.repo.DB.Begin()
	if err := h.repo.GrantCredits(tx, body.UserID, body.Amount, descKey, descParams); err != nil {
		tx.Rollback()
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to grant credits", "error.failedToGrantCredits")
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
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		c.Abort()
		return nil
	}
	return user
}
