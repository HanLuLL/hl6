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

// PaymentOrderType 标识订单业务类型，用于 processPayment 分发。
const (
	PaymentOrderTypeCredits  = "credits"  // 积分充值（默认，向后兼容空值）
	PaymentOrderTypeRealname = "realname" // 实名认证费用
)

type PaymentOrder struct {
	ID            uint            `json:"id" gorm:"primaryKey"`
	UserID        uint            `json:"user_id" gorm:"index;not null"`
	OrderNo       string          `json:"order_no" gorm:"uniqueIndex;size:32;not null"`
	Gateway       string          `json:"gateway" gorm:"type:varchar(16);not null;index"`
	PaymentMethod string          `json:"payment_method" gorm:"type:varchar(16);not null"`
	Amount        float64         `json:"amount" gorm:"not null"`
	Credits       Credit          `json:"credits" gorm:"not null"`
	// Type 标识订单业务类型：credits（积分充值，默认）或 realname（实名认证费）。
	// 空值视为 credits，保持对历史订单的向后兼容。
	Type          string          `json:"type" gorm:"type:varchar(16);not null;default:credits;index"`
	// ReferenceID 当 Type=realname 时指向 realname_applications.id；其他类型为 nil。
	ReferenceID   *uint           `json:"reference_id" gorm:"index"`
	Status        string          `json:"status" gorm:"type:varchar(16);not null;default:pending;index"`
	TradeNo       string          `json:"trade_no" gorm:"size:64"`
	GatewayOrderNo string         `json:"gateway_order_no" gorm:"size:64"`
	PayURL        string          `json:"pay_url" gorm:"type:text"`
	NotifyData    json.RawMessage `json:"notify_data,omitempty" gorm:"type:jsonb"`
	ExpiredAt     *time.Time      `json:"expired_at"`
	PaidAt        *time.Time      `json:"paid_at"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	User          User            `json:"-" gorm:"foreignKey:UserID"`
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
