package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/crypto"
	"hl6-server/pkg/response"
)

// RealnameHandler 实名认证 HTTP handler。
type RealnameHandler struct {
	repo        *repository.Repository
	cfg         *config.Config
	realnameSvc *service.RealnameService
	payment     *PaymentHandler
}

func NewRealnameHandler(repo *repository.Repository, cfg *config.Config, realnameSvc *service.RealnameService, payment *PaymentHandler) *RealnameHandler {
	return &RealnameHandler{repo: repo, cfg: cfg, realnameSvc: realnameSvc, payment: payment}
}

// --- 数据视图 ---

// realnameApplicationView 申请单视图（脱敏）。
type realnameApplicationView struct {
	ID               uint            `json:"id"`
	UserID           uint            `json:"user_id"`
	IDCardMasked     string          `json:"id_card_masked"`
	RealNameMasked   string          `json:"real_name_masked"`
	Provider         string          `json:"provider"`
	VerificationType string          `json:"verification_type"`
	OrderID          *uint           `json:"order_id"`
	Status           string          `json:"status"`
	ProviderResult   json.RawMessage `json:"provider_result,omitempty"`
	RejectReason     string          `json:"reject_reason"`
	ReviewedBy       *uint           `json:"reviewed_by"`
	ReviewedAt       *time.Time      `json:"reviewed_at"`
	VerifiedAt       *time.Time      `json:"verified_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	UserEmail        string          `json:"user_email,omitempty"`
	UserName         string          `json:"user_name,omitempty"`
}

func (h *RealnameHandler) buildApplicationView(app *model.RealnameApplication, includeUser bool) realnameApplicationView {
	realName := crypto.DecryptOrPlaintext(app.RealName, h.cfg.EncryptionKey)
	idCard := crypto.DecryptOrPlaintext(app.IDCard, h.cfg.EncryptionKey)
	view := realnameApplicationView{
		ID:               app.ID,
		UserID:           app.UserID,
		IDCardMasked:     model.MaskIDCard(idCard),
		RealNameMasked:   model.MaskRealName(realName),
		Provider:         app.Provider,
		VerificationType: app.VerificationType,
		OrderID:          app.OrderID,
		Status:           app.Status,
		ProviderResult:   app.ProviderResult,
		RejectReason:     app.RejectReason,
		ReviewedBy:       app.ReviewedBy,
		ReviewedAt:       app.ReviewedAt,
		VerifiedAt:       app.VerifiedAt,
		CreatedAt:        app.CreatedAt,
		UpdatedAt:        app.UpdatedAt,
	}
	if includeUser {
		view.UserEmail = app.User.Email
		view.UserName = app.User.Name
	}
	return view
}

// --- 用户接口 ---

type submitRealnameRequest struct {
	RealName        string `json:"real_name" binding:"required"`
	IDCard          string `json:"id_card" binding:"required"`
	VerificationType string `json:"verification_type"`
	Gateway         string `json:"gateway"`
	PaymentMethod   string `json:"payment_method"`
}

// SubmitApplication POST /api/v1/realname/apply
func (h *RealnameHandler) SubmitApplication(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	var req submitRealnameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	// 加载配置判断是否需要付费
	cfg, err := service.LoadRealnameConfigGlobal(h.repo, h.cfg.EncryptionKey)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load config", "error.databaseError")
		return
	}
	if !cfg.Enabled {
		response.ErrorWithKey(c, http.StatusForbidden, "realname disabled", "error.realnameDisabled")
		return
	}

	// 已实名用户不允许重复提交
	if user.RealnameStatus == model.RealnameStatusVerified {
		response.ErrorWithKey(c, http.StatusBadRequest, "already verified", "error.realnameAlreadyVerified")
		return
	}

	var builder service.PayURLBuilder
	needPay := cfg.Fee > 0
	if needPay {
		// 校验支付方式
		if req.Gateway != model.PaymentGatewayEpay && req.Gateway != model.PaymentGatewayCodePay {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid gateway", "error.invalidPaymentGateway")
			return
		}
		validMethods := map[string]bool{model.PaymentMethodAlipay: true, model.PaymentMethodWechat: true, model.PaymentMethodQQ: true}
		if !validMethods[req.PaymentMethod] {
			response.ErrorWithKey(c, http.StatusBadRequest, "invalid payment method", "error.invalidPaymentMethod")
			return
		}
		payCfg, err := h.payment.loadPaymentConfig()
		if err != nil {
			response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load payment config", "error.databaseError")
			return
		}
		if !payCfg.methodEnabled(req.Gateway, req.PaymentMethod) {
			response.ErrorWithKey(c, http.StatusBadRequest, "payment method not enabled", "error.paymentMethodNotEnabled")
			return
		}
		builder = &realnamePayBuilder{gateway: req.Gateway, method: req.PaymentMethod, payCfg: payCfg, h: h.payment}
	}

	result, err := h.realnameSvc.SubmitApplication(c.Request.Context(), service.SubmitApplicationInput{
		UserID:           user.ID,
		RealName:         req.RealName,
		IDCard:           req.IDCard,
		VerificationType: req.VerificationType,
	}, builder)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	resp := gin.H{
		"application_id": result.ApplicationID,
		"need_pay":       result.NeedPay,
		"fee":            result.Fee,
		"message":        result.Message,
		"verified":       result.Verified,
	}
	if result.NeedPay {
		resp["order_no"] = result.OrderNo
		resp["pay_url"] = result.PayURL
	}
	response.OK(c, resp)
}

// GetStatus GET /api/v1/realname/status
func (h *RealnameHandler) GetStatus(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}
	// 刷新 user 对象拿到最新 realname 字段
	fresh, err := h.repo.FindUserByID(user.ID)
	if err != nil || fresh == nil {
		fresh = user
	}

	app, _ := h.repo.FindLatestRealnameApplication(user.ID)
	resp := gin.H{
		"status":          fresh.RealnameStatus,
		"verified_at":     fresh.RealnameVerifiedAt,
		"realname_name":   fresh.RealnameName,
	}
	if app != nil {
		resp["latest_application"] = h.buildApplicationView(app, false)
	}
	response.OK(c, resp)
}

// GetHistory GET /api/v1/realname/history
func (h *RealnameHandler) GetHistory(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}
	page, perPage := helpers.ParsePageParams(c, 10, 50)
	apps, total, err := h.repo.ListRealnameApplicationsByUser(user.ID, page, perPage)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	views := make([]realnameApplicationView, 0, len(apps))
	for i := range apps {
		views = append(views, h.buildApplicationView(&apps[i], false))
	}
	response.Paginated(c, views, total, page, perPage)
}

// RetryVerification POST /api/v1/realname/retry
// 用户对失败的第三方验证手动触发重试。
func (h *RealnameHandler) RetryVerification(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}
	app, err := h.repo.FindLatestRealnameApplication(user.ID)
	if err != nil || app == nil {
		response.ErrorWithKey(c, http.StatusNotFound, "no application found", "error.realnameNotFound")
		return
	}
	if app.Status != model.RealnameAppStatusFailed {
		response.ErrorWithKey(c, http.StatusBadRequest, "not in failed state", "error.realnameInvalidStatus")
		return
	}
	if err := h.realnameSvc.RetryVerification(c.Request.Context(), app.ID); err != nil {
		slog.Warn("realname: user retry failed", "app_id", app.ID, "err", err)
		response.ErrorWithKey(c, http.StatusInternalServerError, "retry failed", "error.realnameRetryFailed")
		return
	}
	response.OK(c, gin.H{"message": "retry triggered"})
}

// --- 管理员接口 ---

// AdminListApplications GET /api/v1/admin/realname/applications
func (h *RealnameHandler) AdminListApplications(c *gin.Context) {
	page, perPage := helpers.ParsePageParams(c, 20, 100)

	filter := repository.RealnameListFilter{}
	if status := c.Query("status"); status != "" {
		filter.Statuses = strings.Split(status, ",")
	}
	if provider := c.Query("provider"); provider != "" {
		filter.Provider = provider
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := parseUint(userIDStr); err == nil {
			filter.UserID = &uid
		}
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	apps, total, err := h.repo.AdminListRealnameApplications(page, perPage, filter)
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	views := make([]realnameApplicationView, 0, len(apps))
	for i := range apps {
		views = append(views, h.buildApplicationView(&apps[i], true))
	}
	response.Paginated(c, views, total, page, perPage)
}

// AdminGetApplication GET /api/v1/admin/realname/applications/:id
func (h *RealnameHandler) AdminGetApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid id", "error.invalidId")
		return
	}
	app, err := h.repo.FindRealnameApplication(uint(id))
	if err != nil || app == nil {
		response.ErrorWithKey(c, http.StatusNotFound, "not found", "error.realnameNotFound")
		return
	}
	// 预加载 user
	if err := h.repo.GetDB().First(&app.User, app.UserID).Error; err != nil {
		slog.Warn("realname: load user failed", "app_id", app.ID, "err", err)
	}
	response.OK(c, h.buildApplicationView(app, true))
}

type adminReviewRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

// AdminReview PUT /api/v1/admin/realname/applications/:id/review
func (h *RealnameHandler) AdminReview(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid id", "error.invalidId")
		return
	}
	var req adminReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}
	if err := h.realnameSvc.AdminReview(c.Request.Context(), service.AdminReviewInput{
		ApplicationID: uint(id),
		ReviewerID:    admin.ID,
		Approved:      req.Approved,
		Reason:        req.Reason,
	}); err != nil {
		h.handleServiceError(c, err)
		return
	}
	// 记录审计
	detailJSON, _ := json.Marshal(map[string]interface{}{
		"application_id": id,
		"approved":       req.Approved,
		"reason":         req.Reason,
	})
	_ = h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_review_realname",
		Resource:   "realname_application",
		ResourceID: uint(id),
		Details:    detailJSON,
	})
	response.OK(c, gin.H{"message": "reviewed"})
}

// AdminRetryVerification POST /api/v1/admin/realname/applications/:id/retry
// 管理员对失败的第三方验证触发重试。
func (h *RealnameHandler) AdminRetryVerification(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid id", "error.invalidId")
		return
	}
	app, err := h.repo.FindRealnameApplication(uint(id))
	if err != nil || app == nil {
		response.ErrorWithKey(c, http.StatusNotFound, "not found", "error.realnameNotFound")
		return
	}
	if app.Status != model.RealnameAppStatusFailed && app.Status != model.RealnameAppStatusRejected {
		response.ErrorWithKey(c, http.StatusBadRequest, "not retryable", "error.realnameInvalidStatus")
		return
	}
	if err := h.realnameSvc.RetryVerification(c.Request.Context(), app.ID); err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "retry failed", "error.realnameRetryFailed")
		return
	}
	response.OK(c, gin.H{"message": "retry triggered"})
}

// AdminGetStats GET /api/v1/admin/realname/stats
func (h *RealnameHandler) AdminGetStats(c *gin.Context) {
	statusCounts, err := h.repo.CountRealnameApplicationsByStatus()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	verifiedUsers, err := h.repo.CountVerifiedRealnameUsers()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}
	response.OK(c, gin.H{
		"status_counts":  statusCounts,
		"verified_users": verifiedUsers,
	})
}

type adminUpdateUserRealnameRequest struct {
	Action string `json:"action" binding:"required"`
	Reason string `json:"reason"`
}

// AdminUpdateUserRealname PUT /api/v1/admin/users/:id/realname
// 管理员直接修改用户的实名状态，无需走申请单流程。
// action: verify / reject / reset
func (h *RealnameHandler) AdminUpdateUserRealname(c *gin.Context) {
	userID, ok := helpers.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req adminUpdateUserRealnameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}
	action := strings.TrimSpace(req.Action)
	if action != "verify" && action != "reject" && action != "reset" {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid action", "error.invalidRequestBody")
		return
	}
	admin := ctxutil.GetUser(c)
	if admin == nil {
		response.ErrorWithKey(c, http.StatusUnauthorized, "unauthorized", "error.unauthorized")
		return
	}
	if err := h.realnameSvc.AdminUpdateUserRealname(c.Request.Context(), service.AdminUpdateUserRealnameInput{
		UserID:  userID,
		AdminID: admin.ID,
		Action:  action,
		Reason:  req.Reason,
	}); err != nil {
		h.handleServiceError(c, err)
		return
	}
	// 记录审计
	detailJSON, _ := json.Marshal(map[string]interface{}{
		"target_user_id": userID,
		"action":         action,
		"reason":         req.Reason,
	})
	_ = h.repo.CreateAuditLog(&model.AuditLog{
		UserID:     admin.ID,
		Action:     "admin_update_user_realname",
		Resource:   "user",
		ResourceID: userID,
		Details:    detailJSON,
	})
	response.OK(c, gin.H{"message": "realname status updated"})
}

// --- 工具 ---

// realnamePayBuilder 实现 service.PayURLBuilder，复用 PaymentHandler 的支付网关配置。
type realnamePayBuilder struct {
	gateway string
	method  string
	payCfg  *paymentConfig
	h       *PaymentHandler
}

func (b *realnamePayBuilder) Gateway() string { return b.gateway }
func (b *realnamePayBuilder) Method() string  { return b.method }

func (b *realnamePayBuilder) BuildURL(orderNo string, amount float64, productName string) (string, error) {
	switch b.gateway {
	case model.PaymentGatewayEpay:
		svc := b.h.buildEpayService(b.payCfg)
		if svc == nil {
			return "", errors.New("epay not configured")
		}
		method := b.method
		if method == model.PaymentMethodWechat {
			method = "wxpay"
		} else if method == model.PaymentMethodQQ {
			method = "qqpay"
		}
		return svc.CreateOrder(service.CreateEpayOrderParams{
			OutTradeNo: orderNo,
			Amount:     amount,
			Method:     method,
			Name:       productName,
		}), nil
	case model.PaymentGatewayCodePay:
		svc := b.h.buildCodePayService(b.payCfg)
		if svc == nil {
			return "", errors.New("codepay not configured")
		}
		m := 1
		switch b.method {
		case model.PaymentMethodWechat:
			m = 2
		case model.PaymentMethodQQ:
			m = 3
		}
		return svc.CreateOrder(service.CreateCodePayOrderParams{
			OutTradeNo: orderNo,
			Amount:     amount,
			Method:     m,
			Name:       productName,
		}), nil
	}
	return "", fmt.Errorf("unsupported gateway: %s", b.gateway)
}

// handleServiceError 将 service 层错误映射为 HTTP 响应。
func (h *RealnameHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRealnameDisabled):
		response.ErrorWithKey(c, http.StatusForbidden, "realname disabled", "error.realnameDisabled")
	case errors.Is(err, service.ErrRealnameInvalidInput):
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid input", "error.realnameInvalidInput")
	case errors.Is(err, service.ErrRealnameInvalidIDCard):
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid id card", "error.realnameInvalidIDCard")
	case errors.Is(err, service.ErrRealnameInvalidName):
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid name", "error.realnameInvalidName")
	case errors.Is(err, service.ErrRealnamePendingExists):
		response.ErrorWithKey(c, http.StatusConflict, "pending application exists", "error.realnamePendingExists")
	case errors.Is(err, service.ErrRealnameNotFound):
		response.ErrorWithKey(c, http.StatusNotFound, "not found", "error.realnameNotFound")
	case errors.Is(err, service.ErrRealnameInvalidStatus):
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid status", "error.realnameInvalidStatus")
	case errors.Is(err, service.ErrRealnamePayURLFailed):
		response.ErrorWithKey(c, http.StatusInternalServerError, "pay url failed", "error.realnamePayURLFailed")
	default:
		slog.Warn("realname: unhandled service error", "err", err)
		response.ErrorWithKey(c, http.StatusInternalServerError, "internal error", "error.databaseError")
	}
}

func parseUint(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	return uint(n), err
}
