package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
	"hl6-server/pkg/response"
)

type PaymentHandler struct {
	repo       *repository.Repository
	epaySvc    *service.EpayService
	codepaySvc *service.CodePayService
}

func NewPaymentHandler(repo *repository.Repository, epaySvc *service.EpayService, codepaySvc *service.CodePayService) *PaymentHandler {
	return &PaymentHandler{repo: repo, epaySvc: epaySvc, codepaySvc: codepaySvc}
}

// Credit pricing: 1 CNY = 1 display credit = CreditScale internal units
const (
	defaultMinRechargeCNY = 1.0  // minimum recharge amount in CNY
	defaultMaxRechargeCNY = 10000.0
)

type CreatePaymentRequest struct {
	Gateway       string  `json:"gateway" binding:"required"`        // epay or codepay
	PaymentMethod string  `json:"payment_method" binding:"required"` // alipay, wechat, qq
	Amount        float64 `json:"amount" binding:"required"`         // CNY amount
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
		if h.epaySvc == nil {
			response.ErrorWithKey(c, http.StatusBadRequest, "epay not configured", "error.paymentGatewayNotConfigured")
			return
		}
		method := req.PaymentMethod
		if method == model.PaymentMethodWechat {
			method = "wxpay"
		} else if method == model.PaymentMethodQQ {
			method = "qqpay"
		}
		payURL = h.epaySvc.CreateOrder(service.CreateEpayOrderParams{
			OutTradeNo: orderNo,
			Amount:     req.Amount,
			Method:     method,
			Name:       productName,
		})

	case model.PaymentGatewayCodePay:
		if h.codepaySvc == nil {
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
		payURL = h.codepaySvc.CreateOrder(service.CreateCodePayOrderParams{
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
	if h.epaySvc == nil {
		c.String(http.StatusOK, "epay not configured")
		return
	}

	if !h.epaySvc.VerifyCallback(c.Request.URL.Query()) {
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
	if h.codepaySvc == nil {
		c.String(http.StatusOK, "codepay not configured")
		return
	}

	if !h.codepaySvc.VerifyCallback(c.Request.URL.Query()) {
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

	txErr := h.repo.Transaction(func(tx *gorm.DB) error {
		// Update order status
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"status":      model.PaymentStatusPaid,
			"trade_no":    tradeNo,
			"gateway":     gateway,
			"notify_data": notifyJSON,
			"paid_at":     &now,
		}).Error; err != nil {
			return err
		}

		// Credit the user
		descParams, _ := json.Marshal(map[string]string{"order_no": outTradeNo})
		return h.repo.GrantCredits(tx, order.UserID, order.Credits, "txn.recharge", descParams)
	})

	if txErr != nil {
		// Log error but don't fail the callback (gateway will retry)
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
