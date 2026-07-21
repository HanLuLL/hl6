package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

type PaymentHandler struct {
	repo        *repository.Repository
	cfg         *config.Config
	realnameSvc *service.RealnameService // 可选：注入后 processPayment 会根据订单类型分发
}

func NewPaymentHandler(repo *repository.Repository, cfg *config.Config) *PaymentHandler {
	return &PaymentHandler{repo: repo, cfg: cfg}
}

// SetRealnameService 注入实名认证服务，启用支付成功回调按订单类型分发。
func (h *PaymentHandler) SetRealnameService(s *service.RealnameService) {
	h.realnameSvc = s
}

// Credit pricing: 1 CNY = 1 display credit = CreditScale internal units
const (
	defaultMinRechargeCNY = 1.0  // minimum recharge amount in CNY
	defaultMaxRechargeCNY = 10000.0
)

// 支付配置 key 常量
const (
	configKeyEpayURL            = "epay_url"
	configKeyEpayPID            = "epay_pid"
	configKeyEpayKey            = "epay_key"
	configKeyCodePayURL         = "codepay_url"
	configKeyCodePayID          = "codepay_id"
	configKeyCodePayKey         = "codepay_key"
	configKeyEpayAlipayEnabled  = "epay_alipay_enabled"
	configKeyEpayWechatEnabled  = "epay_wechat_enabled"
	configKeyEpayQQEnabled      = "epay_qq_enabled"
	configKeyCodePayAlipayEnabled = "codepay_alipay_enabled"
	configKeyCodePayWechatEnabled = "codepay_wechat_enabled"
	configKeyCodePayQQEnabled     = "codepay_qq_enabled"
)

type paymentConfig struct {
	EpayURL            string
	EpayPID            string
	EpayKey            string
	CodePayURL         string
	CodePayID          string
	CodePayKey         string
	EpayAlipayEnabled  bool
	EpayWechatEnabled  bool
	EpayQQEnabled      bool
	CodePayAlipayEnabled bool
	CodePayWechatEnabled bool
	CodePayQQEnabled     bool
}

func (h *PaymentHandler) loadPaymentConfig() (*paymentConfig, error) {
	keys := []string{
		configKeyEpayURL, configKeyEpayPID, configKeyEpayKey,
		configKeyCodePayURL, configKeyCodePayID, configKeyCodePayKey,
		configKeyEpayAlipayEnabled, configKeyEpayWechatEnabled, configKeyEpayQQEnabled,
		configKeyCodePayAlipayEnabled, configKeyCodePayWechatEnabled, configKeyCodePayQQEnabled,
	}
	values, err := h.repo.GetSystemConfigsByKeys(keys)
	if err != nil {
		return nil, err
	}
	return &paymentConfig{
		EpayURL:              values[configKeyEpayURL],
		EpayPID:              values[configKeyEpayPID],
		EpayKey:              values[configKeyEpayKey],
		CodePayURL:           values[configKeyCodePayURL],
		CodePayID:            values[configKeyCodePayID],
		CodePayKey:           values[configKeyCodePayKey],
		EpayAlipayEnabled:    values[configKeyEpayAlipayEnabled] == "true",
		EpayWechatEnabled:    values[configKeyEpayWechatEnabled] == "true",
		EpayQQEnabled:        values[configKeyEpayQQEnabled] == "true",
		CodePayAlipayEnabled: values[configKeyCodePayAlipayEnabled] == "true",
		CodePayWechatEnabled: values[configKeyCodePayWechatEnabled] == "true",
		CodePayQQEnabled:     values[configKeyCodePayQQEnabled] == "true",
	}, nil
}

func (h *PaymentHandler) resolveBackendURL() string {
	// 优先使用 DB 中的 backend_url，其次使用配置
	if u, err := h.repo.GetSystemConfig("backend_url"); err == nil && strings.TrimSpace(u) != "" {
		return strings.TrimRight(u, "/")
	}
	if h.cfg.BackendURL != "" {
		return strings.TrimRight(h.cfg.BackendURL, "/")
	}
	return "http://localhost:8081"
}

func (h *PaymentHandler) buildEpayService(cfg *paymentConfig) *service.EpayService {
	if cfg.EpayURL == "" || cfg.EpayPID == "" || cfg.EpayKey == "" {
		return nil
	}
	backendURL := h.resolveBackendURL()
	return service.NewEpayService(
		cfg.EpayURL, cfg.EpayPID, cfg.EpayKey,
		backendURL+"/api/v1/payment/epay/notify",
		backendURL+"/api/v1/payment/return",
	)
}

func (h *PaymentHandler) buildCodePayService(cfg *paymentConfig) *service.CodePayService {
	if cfg.CodePayURL == "" || cfg.CodePayID == "" || cfg.CodePayKey == "" {
		return nil
	}
	backendURL := h.resolveBackendURL()
	return service.NewCodePayService(
		cfg.CodePayURL, cfg.CodePayID, cfg.CodePayKey,
		backendURL+"/api/v1/payment/codepay/notify",
		backendURL+"/api/v1/payment/return",
	)
}

// paymentMethodEnabled 判断指定网关+渠道是否启用
func (cfg *paymentConfig) methodEnabled(gateway, method string) bool {
	switch gateway {
	case model.PaymentGatewayEpay:
		switch method {
		case model.PaymentMethodAlipay:
			return cfg.EpayAlipayEnabled
		case model.PaymentMethodWechat:
			return cfg.EpayWechatEnabled
		case model.PaymentMethodQQ:
			return cfg.EpayQQEnabled
		}
	case model.PaymentGatewayCodePay:
		switch method {
		case model.PaymentMethodAlipay:
			return cfg.CodePayAlipayEnabled
		case model.PaymentMethodWechat:
			return cfg.CodePayWechatEnabled
		case model.PaymentMethodQQ:
			return cfg.CodePayQQEnabled
		}
	}
	return false
}

type CreatePaymentRequest struct {
	Gateway       string  `json:"gateway" binding:"required"`        // epay or codepay
	PaymentMethod string  `json:"payment_method" binding:"required"` // alipay, wechat, qq
	Amount        float64 `json:"amount" binding:"required"`         // CNY amount
}

// PaymentMethodOption 描述一个可用的支付方式
type PaymentMethodOption struct {
	Gateway string `json:"gateway"`
	Method  string `json:"method"`
}

// GetPaymentMethods 返回当前启用的支付方式列表（公开接口）
func (h *PaymentHandler) GetPaymentMethods(c *gin.Context) {
	cfg, err := h.loadPaymentConfig()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load payment config", "error.databaseError")
		return
	}

	methods := []PaymentMethodOption{}
	if cfg.EpayAlipayEnabled {
		methods = append(methods, PaymentMethodOption{Gateway: model.PaymentGatewayEpay, Method: model.PaymentMethodAlipay})
	}
	if cfg.EpayWechatEnabled {
		methods = append(methods, PaymentMethodOption{Gateway: model.PaymentGatewayEpay, Method: model.PaymentMethodWechat})
	}
	if cfg.EpayQQEnabled {
		methods = append(methods, PaymentMethodOption{Gateway: model.PaymentGatewayEpay, Method: model.PaymentMethodQQ})
	}
	if cfg.CodePayAlipayEnabled {
		methods = append(methods, PaymentMethodOption{Gateway: model.PaymentGatewayCodePay, Method: model.PaymentMethodAlipay})
	}
	if cfg.CodePayWechatEnabled {
		methods = append(methods, PaymentMethodOption{Gateway: model.PaymentGatewayCodePay, Method: model.PaymentMethodWechat})
	}
	if cfg.CodePayQQEnabled {
		methods = append(methods, PaymentMethodOption{Gateway: model.PaymentGatewayCodePay, Method: model.PaymentMethodQQ})
	}

	response.OK(c, gin.H{"methods": methods})
}

func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid request body", "error.invalidRequestBody")
		return
	}

	// Validate gateway
	if req.Gateway != model.PaymentGatewayEpay && req.Gateway != model.PaymentGatewayCodePay {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid payment gateway", "error.invalidPaymentGateway")
		return
	}

	// Validate payment method
	validMethods := map[string]bool{model.PaymentMethodAlipay: true, model.PaymentMethodWechat: true, model.PaymentMethodQQ: true}
	if !validMethods[req.PaymentMethod] {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid payment method", "error.invalidPaymentMethod")
		return
	}

	// Load config and validate the gateway+method is enabled
	payCfg, err := h.loadPaymentConfig()
	if err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to load payment config", "error.databaseError")
		return
	}
	if !payCfg.methodEnabled(req.Gateway, req.PaymentMethod) {
		response.ErrorWithKey(c, http.StatusBadRequest, "payment method not enabled", "error.paymentMethodNotEnabled")
		return
	}

	// Validate amount
	if req.Amount < defaultMinRechargeCNY || req.Amount > defaultMaxRechargeCNY {
		response.ErrorWithKey(c, http.StatusBadRequest, "invalid payment amount", "error.invalidPaymentAmount")
		return
	}

	// Round amount to 2 decimal places
	req.Amount = math.Round(req.Amount*100) / 100

	// Convert CNY to display credits (1 CNY = 1 display credit), then to internal units
	credits := model.CreditFromFloat(req.Amount)

	// Generate order number
	orderNo := fmt.Sprintf("P%d%d", time.Now().UnixMilli(), user.ID)

	// Calculate expiry (30 minutes)
	expiredAt := time.Now().Add(30 * time.Minute)

	order := &model.PaymentOrder{
		UserID:        user.ID,
		OrderNo:       orderNo,
		Gateway:       req.Gateway,
		PaymentMethod: req.PaymentMethod,
		Amount:        req.Amount,
		Credits:       credits,
		Status:        model.PaymentStatusPending,
		ExpiredAt:     &expiredAt,
	}

	// Get payment URL based on gateway
	var payURL string
	productName := fmt.Sprintf("积分充值 %.1f 积分", credits.ToFloat())

	switch req.Gateway {
	case model.PaymentGatewayEpay:
		epaySvc := h.buildEpayService(payCfg)
		if epaySvc == nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "epay not configured", "error.paymentGatewayNotConfigured")
			return
		}
		method := req.PaymentMethod
		if method == model.PaymentMethodWechat {
			method = "wxpay"
		} else if method == model.PaymentMethodQQ {
			method = "qqpay"
		}
		payURL = epaySvc.CreateOrder(service.CreateEpayOrderParams{
			OutTradeNo: orderNo,
			Amount:     req.Amount,
			Method:     method,
			Name:       productName,
		})

	case model.PaymentGatewayCodePay:
		codepaySvc := h.buildCodePayService(payCfg)
		if codepaySvc == nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "codepay not configured", "error.paymentGatewayNotConfigured")
			return
		}
		method := 1 // alipay
		switch req.PaymentMethod {
		case model.PaymentMethodWechat:
			method = 2
		case model.PaymentMethodQQ:
			method = 3
		}
		payURL = codepaySvc.CreateOrder(service.CreateCodePayOrderParams{
			OutTradeNo: orderNo,
			Amount:     req.Amount,
			Method:     method,
			Name:       productName,
		})
	}

	order.PayURL = payURL

	if err := h.repo.GetDB().Create(order).Error; err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "failed to create order", "error.failedToCreateOrder")
		return
	}

	response.OK(c, gin.H{
		"order_no": orderNo,
		"pay_url":  payURL,
		"credits":  credits,
		"amount":   req.Amount,
	})
}

func (h *PaymentHandler) GetOrders(c *gin.Context) {
	user := mustGetUser(c)
	if user == nil {
		return
	}

	var orders []model.PaymentOrder
	if err := h.repo.GetDB().Where("user_id = ?", user.ID).Order("created_at DESC").Limit(20).Find(&orders).Error; err != nil {
		response.ErrorWithKey(c, http.StatusInternalServerError, "database error", "error.databaseError")
		return
	}

	response.OK(c, orders)
}

func (h *PaymentHandler) GetProducts(c *gin.Context) {
	// Define available credit packages (price in CNY, credits in display units)
	products := []model.PaymentProduct{
		{ID: 1, Credits: model.CreditFromFloat(10), Price: 10.00, Name: "10 积分"},
		{ID: 2, Credits: model.CreditFromFloat(50), Price: 50.00, Name: "50 积分"},
		{ID: 3, Credits: model.CreditFromFloat(100), Price: 100.00, Name: "100 积分"},
		{ID: 4, Credits: model.CreditFromFloat(500), Price: 500.00, Name: "500 积分"},
		{ID: 5, Credits: model.CreditFromFloat(1000), Price: 1000.00, Name: "1000 积分"},
	}
	response.OK(c, products)
}

// EpayCallback handles 易支付 payment notification
func (h *PaymentHandler) EpayCallback(c *gin.Context) {
	payCfg, err := h.loadPaymentConfig()
	if err != nil {
		c.String(http.StatusOK, "config error")
		return
	}
	epaySvc := h.buildEpayService(payCfg)
	if epaySvc == nil {
		c.String(http.StatusOK, "epay not configured")
		return
	}

	if !epaySvc.VerifyCallback(c.Request.URL.Query()) {
		c.String(http.StatusOK, "sign error")
		return
	}

	tradeStatus := c.Query("trade_status")
	if tradeStatus != "TRADE_SUCCESS" {
		c.String(http.StatusOK, "fail")
		return
	}

	outTradeNo := c.Query("out_trade_no")
	tradeNo := c.Query("trade_no")

	h.processPayment(outTradeNo, tradeNo, model.PaymentGatewayEpay, c.Request.URL.Query())

	c.String(http.StatusOK, "success")
}

// CodePayCallback handles 码支付 payment notification
func (h *PaymentHandler) CodePayCallback(c *gin.Context) {
	payCfg, err := h.loadPaymentConfig()
	if err != nil {
		c.String(http.StatusOK, "config error")
		return
	}
	codepaySvc := h.buildCodePayService(payCfg)
	if codepaySvc == nil {
		c.String(http.StatusOK, "codepay not configured")
		return
	}

	if !codepaySvc.VerifyCallback(c.Request.URL.Query()) {
		c.String(http.StatusOK, "sign error")
		return
	}

	payNo := c.Query("pay_no")
	outTradeNo := c.Query("pay_id")

	h.processPayment(outTradeNo, payNo, model.PaymentGatewayCodePay, c.Request.URL.Query())

	c.String(http.StatusOK, "success")
}

func (h *PaymentHandler) processPayment(outTradeNo, tradeNo, gateway string, notifyData url.Values) {
	var order model.PaymentOrder
	if err := h.repo.GetDB().Where("order_no = ? AND status = ?", outTradeNo, model.PaymentStatusPending).First(&order).Error; err != nil {
		return
	}

	now := time.Now()
	notifyJSON, _ := json.Marshal(notifyData)

	// 确定订单业务类型：空值视为 credits（向后兼容历史订单）
	orderType := order.Type
	if orderType == "" {
		orderType = model.PaymentOrderTypeCredits
	}

	// realname 类型订单：仅更新支付状态，第三方实名验证由 RealnameService 异步触发。
	// 此处不调用 realnameSvc.StartVerification 以避免在支付回调事务中阻塞于第三方 API。
	var txErr error
	if orderType == model.PaymentOrderTypeRealname && h.realnameSvc != nil {
		txErr = h.repo.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&order).Updates(map[string]interface{}{
				"status":      model.PaymentStatusPaid,
				"trade_no":    tradeNo,
				"gateway":     gateway,
				"notify_data": notifyJSON,
				"paid_at":     &now,
			}).Error; err != nil {
				return err
			}
			return h.realnameSvc.HandlePaymentSuccess(tx, order.ID, order.ReferenceID)
		})
		// 事务提交后异步触发第三方验证（失败不影响回调响应）
		if txErr == nil && order.ReferenceID != nil {
			go func(appID uint) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := h.realnameSvc.StartVerification(ctx, appID); err != nil {
					slog.Warn("realname: async verification failed", "app_id", appID, "err", err)
				}
			}(*order.ReferenceID)
		}
	} else {
		// 默认：积分充值
		txErr = h.repo.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&order).Updates(map[string]interface{}{
				"status":      model.PaymentStatusPaid,
				"trade_no":    tradeNo,
				"gateway":     gateway,
				"notify_data": notifyJSON,
				"paid_at":     &now,
			}).Error; err != nil {
				return err
			}
			descParams, _ := json.Marshal(map[string]string{"order_no": outTradeNo})
			return h.repo.GrantCredits(tx, order.UserID, order.Credits, "txn.recharge", descParams)
		})
	}

	if txErr != nil {
		// Log error but don't fail the callback (gateway will retry)
		slog.Warn("payment: processPayment failed", "order_no", outTradeNo, "err", txErr)
		return
	}
}

// PaymentReturn handles the user return from payment page
func (h *PaymentHandler) PaymentReturn(c *gin.Context) {
	// Redirect to the frontend credits page
	c.Redirect(http.StatusFound, "/credits")
}

// AdminGetOrders lists all payment orders (admin only)
func (h *PaymentHandler) AdminGetOrders(c *gin.Context) {
	page, perPage := helpers.ParsePageParams(c, 20, 100)

	var orders []model.PaymentOrder
	var total int64

	q := h.repo.GetDB().Model(&model.PaymentOrder{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if gateway := c.Query("gateway"); gateway != "" {
		q = q.Where("gateway = ?", gateway)
	}

	q.Count(&total)
	q.Preload("User").Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&orders)

	response.Paginated(c, orders, total, page, perPage)
}
