package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
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
	user := mustGetUser(c)
	if user == nil {
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
	user := mustGetUser(c)
	if user == nil {
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

func (h *CreditHandler) GetDailyCheckinStatus(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	cfg, err := loadDailyCheckinRuntimeConfig(h.repo)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get config", "error.failedToGetConfig")
		return
	}

	today := todayInBeijing(time.Now())
	claimed := false
	if cfg.Enabled {
		claimed, err = h.repo.HasDailyCheckinClaim(user.ID, today)
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
			return
		}
	}

	response.OK(c, gin.H{
		"enabled":       cfg.Enabled,
		"reward":        cfg.Reward,
		"claimed_today": claimed,
		"checkin_date":  today,
	})
}

func (h *CreditHandler) DailyCheckin(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	cfg, err := loadDailyCheckinRuntimeConfig(h.repo)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to get config", "error.failedToGetConfig")
		return
	}
	if !cfg.Enabled {
		response.ErrorWithKey(c, http.StatusForbidden, "daily checkin is disabled", "error.dailyCheckinDisabled")
		return
	}

	if user.GroupID == nil || !cfg.IsGroupAllowed(*user.GroupID) {
		response.ErrorWithKey(c, http.StatusForbidden, "group is not allowed for daily checkin", "error.dailyCheckinGroupNotAllowed")
		return
	}

	today := todayInBeijing(time.Now())
	txErr := h.repo.Transaction(func(tx *gorm.DB) error {
		claim := &model.DailyCheckinClaim{
			UserID:      user.ID,
			CheckinDate: today,
			Reward:      cfg.Reward,
		}
		if err := tx.Create(claim).Error; err != nil {
			return err
		}
		return h.repo.GrantCredits(tx, user.ID, cfg.Reward, "txn.dailyCheckin", nil)
	})
	if txErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(txErr, &pgErr) && pgErr.Code == "23505" {
			response.ErrorWithKey(c, http.StatusConflict, "daily checkin already claimed", "error.dailyCheckinAlreadyClaimed")
			return
		}
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to grant credits", "error.failedToGrantCredits")
		return
	}

	balance, err := h.repo.GetCreditBalance(user.ID)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	response.OK(c, gin.H{
		"granted":       cfg.Reward,
		"balance":       balance.Balance,
		"claimed_today": true,
		"checkin_date":  today,
	})
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

	amount, err := model.ParseDisplayCredit(body.Amount, true, true)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid amount", "error.invalidCreditAmount")
		return
	}
	if amount == 0 {
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

	admin := ctxutil.GetUser(c)

	txErr := h.repo.Transaction(func(tx *gorm.DB) error {
		if amount > 0 {
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
		deductAmount := -amount
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
		if errors.Is(txErr, repository.ErrInvalidCreditAmount) {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid amount", "error.invalidCreditAmount")
			return
		}
		if errors.Is(txErr, repository.ErrCreditOverflow) {
			response.ErrorWithKey(c, http.StatusBadRequest, "credit overflow", "error.invalidCreditAmount")
			return
		}
		if amount > 0 {
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
		"granted": amount,
		"balance": balance.Balance,
	})
}
