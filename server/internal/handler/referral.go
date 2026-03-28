package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/repository"
	"hl6-server/pkg/response"
)

type ReferralHandler struct {
	repo *repository.Repository
}

func NewReferralHandler(repo *repository.Repository) *ReferralHandler {
	return &ReferralHandler{repo: repo}
}

func (h *ReferralHandler) GetReferralInfo(c *gin.Context) {
	user := ctxutil.GetUser(c)
	if user == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
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

	enabledStr, cfgErr := h.repo.GetSystemConfig("referral_enabled")
	referralEnabled := cfgErr == nil && enabledStr == "true"

	referrals, total, err := h.repo.ListReferralsByInviter(user.ID, page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to list referrals", "error.databaseError")
		return
	}

	type referralItem struct {
		ID               uint    `json:"id"`
		InviteeName      string  `json:"invitee_name"`
		InviteeCreatedAt string  `json:"invitee_created_at"`
		InviterCredits   float64 `json:"inviter_credits"`
		CreatedAt        string  `json:"created_at"`
	}

	items := make([]referralItem, 0, len(referrals))
	for _, r := range referrals {
		name := r.Invitee.Name
		if name == "" {
			name = "—"
		}
		items = append(items, referralItem{
			ID:               r.ID,
			InviteeName:      name,
			InviteeCreatedAt: r.Invitee.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			InviterCredits:   r.InviterCredits.ToFloat(),
			CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":             0,
		"message":          "ok",
		"referral_code":    user.ReferralCode,
		"referral_enabled": referralEnabled,
		"data":             items,
		"total":            total,
		"page":             page,
		"per_page":         perPage,
	})
}
