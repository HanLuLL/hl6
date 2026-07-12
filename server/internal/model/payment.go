package model

import (
	"encoding/json"
	"time"
)

// PaymentOrderStatus constants
const (
	PaymentStatusPending = "pending"
	PaymentStatusPaid    = "paid"
	PaymentStatusFailed  = "failed"
	PaymentStatusExpired = "expired"
)

// PaymentGateway constants
const (
	PaymentGatewayEpay    = "epay"
	PaymentGatewayCodePay = "codepay"
)

// PaymentMethod constants
const (
	PaymentMethodAlipay = "alipay"
	PaymentMethodWechat = "wechat"
	PaymentMethodQQ     = "qq"
)

type PaymentOrder struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	UserID         uint            `json:"user_id" gorm:"index;not null"`
	OrderNo        string          `json:"order_no" gorm:"uniqueIndex;size:32;not null"`
	Gateway        string          `json:"gateway" gorm:"type:varchar(16);not null;index"`
	PaymentMethod  string          `json:"payment_method" gorm:"type:varchar(16);not null"`
	Amount         float64         `json:"amount" gorm:"not null"`
	Credits        Credit          `json:"credits" gorm:"not null"`
	Status         string          `json:"status" gorm:"type:varchar(16);not null;default:pending;index"`
	TradeNo        string          `json:"trade_no" gorm:"size:64"`
	GatewayOrderNo string          `json:"gateway_order_no" gorm:"size:64"`
	PayURL         string          `json:"pay_url" gorm:"type:text"`
	NotifyData     json.RawMessage `json:"notify_data,omitempty" gorm:"type:jsonb"`
	ExpiredAt      *time.Time      `json:"expired_at"`
	PaidAt         *time.Time      `json:"paid_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	User           User            `json:"-" gorm:"foreignKey:UserID"`
}

func (PaymentOrder) TableName() string {
	return "payment_orders"
}

// PaymentProduct defines the credit packages users can purchase
type PaymentProduct struct {
	ID      uint   `json:"id"`
	Credits Credit `json:"credits"`
	Price   float64 `json:"price"`
	Name    string  `json:"name"`
}
