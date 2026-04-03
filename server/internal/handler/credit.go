package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
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
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	balance, err := h.repo.EnsureCreditBalance(user.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	txns, _, err := h.repo.ListTransactions(user.ID, 1, 10)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list transactions", "error.failedToListTransactions")
		return
	}
	response.OK(c, gin.H{
		"balance":      balance.Balance,
		"transactions": txns,
	})
}

func (h *CreditHandler) ListTransactions(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "user not found", "error.userNotFound")
		return
	}
	page, perPage := helpers.ParsePageParams(c, 20, 100)

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
	if body.Amount == 0 {
		response.ErrorWithKey(c, http.StatusBadRequest, "amount cannot be zero", "error.amountCannotBeZero")
		return
	}

	var user model.User
	if err := h.repo.GetDB().First(&user, body.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorWithKey(c, http.StatusNotFound, "user not found", "error.userNotFound")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	amount := model.CreditFromFloat(body.Amount)

	admin := ctxutil.GetUser(c)

	txErr := h.repo.Transaction(func(tx *gorm.DB) error {
		if body.Amount > 0 {
			descKey := "txn.adminGrant"
			var descParams json.RawMessage
			if body.Description != "" {
				descKey = "txn.adminGrantCustom"
				descParams, _ = json.Marshal(map[string]string{"description": body.Description})
			}
			if err := h.repo.GrantCredits(tx, body.UserID, amount, descKey, descParams); err != nil {
				return err
			}
			if admin != nil {
				details, _ := json.Marshal(map[string]interface{}{"target_user_id": body.UserID, "amount": body.Amount, "description": body.Description})
				return tx.Create(&model.AuditLog{
					UserID:     admin.ID,
					Action:     "admin_grant_credits",
					Resource:   "credit",
					ResourceID: body.UserID,
					Details:    details,
				}).Error
			}
			return nil
		}
		deductAmount := model.CreditFromFloat(-body.Amount)
		descKey := "txn.adminDeduct"
		var descParams json.RawMessage
		if body.Description != "" {
			descKey = "txn.adminDeductCustom"
			descParams, _ = json.Marshal(map[string]string{"description": body.Description})
		}
		if err := h.repo.DeductCredits(tx, body.UserID, deductAmount, descKey, descParams); err != nil {
			return err
		}
		if admin != nil {
			details, _ := json.Marshal(map[string]interface{}{"target_user_id": body.UserID, "amount": body.Amount, "description": body.Description})
			return tx.Create(&model.AuditLog{
				UserID:     admin.ID,
				Action:     "admin_deduct_credits",
				Resource:   "credit",
				ResourceID: body.UserID,
				Details:    details,
			}).Error
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, gorm.ErrInvalidData) {
			response.ErrorWithKey(c, http.StatusPaymentRequired, "insufficient credits", "error.insufficientCredits")
			return
		}
		if body.Amount > 0 {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to grant credits", "error.failedToGrantCredits")
		} else {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to deduct credits", "error.failedToDeductCredits")
		}
		return
	}

	balance, err := h.repo.GetCreditBalance(body.UserID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	response.OK(c, gin.H{
		"user_id": body.UserID,
		"granted": body.Amount,
		"balance": balance.Balance,
	})
}
