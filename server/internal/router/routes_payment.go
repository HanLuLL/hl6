package router

import (
	"github.com/gin-gonic/gin"
	"hl6-server/internal/middleware"
)

func registerPaymentRoutes(api *gin.RouterGroup, auth *middleware.AuthMiddleware, h *Handlers) {
	// Public callback endpoints (no auth)
	api.GET("/payment/epay/notify", h.Payment.EpayCallback)
	api.POST("/payment/epay/notify", h.Payment.EpayCallback)
	api.GET("/payment/codepay/notify", h.Payment.CodePayCallback)
	api.POST("/payment/codepay/notify", h.Payment.CodePayCallback)
	api.GET("/payment/return", h.Payment.PaymentReturn)

	// Authenticated endpoints
	authed := api.Group("", auth.Required())
	authed.GET("/payment/products", h.Payment.GetProducts)
	authed.POST("/payment/orders", h.Payment.CreateOrder)
	authed.GET("/payment/orders", h.Payment.GetOrders)

	// Admin endpoints
	admin := api.Group("/admin", auth.Required(), middleware.AdminRequired())
	admin.GET("/payment/orders", h.Payment.AdminGetOrders)
}
