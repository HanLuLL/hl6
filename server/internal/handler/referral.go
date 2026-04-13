package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/helpers"
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
	user := mustGetUser(c)
	if user == nil {
		return
	}

	page, perPage := helpers.ParsePageParams(c, 20, 100)

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

	response.OK(c, gin.H{
		"referral_code":    user.ReferralCode,
		"referral_enabled": referralEnabled,
		"records":          items,
		"total":            total,
		"page":             page,
		"per_page":         perPage,
	})
}
