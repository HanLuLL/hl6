package model

import (
	"encoding/json"
	"time"
)

// RealnameApplicationStatus 申请单状态常量
const (
	RealnameAppStatusPendingPayment = "pending_payment" // 已提交，等待支付
	RealnameAppStatusPaid           = "paid"           // 已支付，待触发验证
	RealnameAppStatusVerifying      = "verifying"      // 正在调用第三方 API
	RealnameAppStatusVerified       = "verified"       // 实名通过
	RealnameAppStatusRejected       = "rejected"       // 实名失败或管理员拒绝
	RealnameAppStatusFailed         = "failed"         // 第三方 API 调用失败
)

// RealnameProvider 实名认证服务商常量
const (
	RealnameProviderAliyun = "aliyun" // 阿里云市场（AppCode 鉴权）
	RealnameProviderJuhe   = "juhe"   // 聚合数据（API Key 鉴权）
	RealnameProviderManual = "manual" // 人工审核（不走第三方 API）
)

// RealnameVerificationType 验证类型常量
const (
	RealnameVerifyIDCard = "idcard" // 二要素核验（姓名+身份证）
	RealnameVerifyFace   = "face"   // 人脸活体比对
)

// RealnameApplication 实名认证申请单。
// 真实姓名和身份证号在数据库中通过 AES-256-GCM 加密存储，
// API 响应时仅返回脱敏字段（IDCardMasked / RealNameMasked）。
type RealnameApplication struct {
	ID               uint            `json:"id" gorm:"primaryKey"`
	UserID           uint            `json:"user_id" gorm:"index;not null"`
	RealName         string          `json:"-" gorm:"type:text;not null"` // 加密存储，不返回前端
	IDCard           string          `json:"-" gorm:"type:text;not null"` // 加密存储，不返回前端
	Provider         string          `json:"provider" gorm:"type:varchar(16);not null;default:manual"`
	VerificationType string          `json:"verification_type" gorm:"type:varchar(16);not null;default:idcard"`
	OrderID          *uint           `json:"order_id" gorm:"index"` // 关联 PaymentOrder.ID
	Status           string          `json:"status" gorm:"type:varchar(20);not null;default:pending_payment;index"`
	ProviderRequest  json.RawMessage `json:"provider_request,omitempty" gorm:"type:jsonb"`
	ProviderResult   json.RawMessage `json:"provider_result,omitempty" gorm:"type:jsonb"`
	RejectReason     string          `json:"reject_reason" gorm:"type:text;default:''"`
	ReviewedBy       *uint           `json:"reviewed_by"`
	ReviewedAt       *time.Time      `json:"reviewed_at"`
	VerifiedAt       *time.Time      `json:"verified_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	User             User            `json:"-" gorm:"foreignKey:UserID"`
}

func (RealnameApplication) TableName() string { return "realname_applications" }

// MaskIDCard 将 18 位身份证号脱敏为 "前6后4" 格式（如 "110101********1234"）。
// 输入非 18 位时仅保留首尾各 1 位。
func MaskIDCard(id string) string {
	r := []rune(id)
	n := len(r)
	if n == 0 {
		return ""
	}
	if n < 6 {
		// 过短：仅保留首字符
		masked := make([]rune, n)
		for i := range masked {
			if i == 0 {
				masked[i] = r[i]
			} else {
				masked[i] = '*'
			}
		}
		return string(masked)
	}
	if n < 10 {
		// 6-9 位：前2后2
		masked := make([]rune, n)
		for i := range masked {
			if i < 2 || i >= n-2 {
				masked[i] = r[i]
			} else {
				masked[i] = '*'
			}
		}
		return string(masked)
	}
	// 10 位以上：前6后4
	masked := make([]rune, n)
	for i := range masked {
		if i < 6 || i >= n-4 {
			masked[i] = r[i]
		} else {
			masked[i] = '*'
		}
	}
	return string(masked)
}

// MaskRealName 将中文姓名脱敏为 "首字+*" 格式（如 "张三" -> "张*"，"欧阳锋" -> "欧**"）。
// 非中文名仅保留首字符。
func MaskRealName(name string) string {
	r := []rune(name)
	n := len(r)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return string(r)
	}
	masked := make([]rune, n)
	masked[0] = r[0]
	for i := 1; i < n; i++ {
		masked[i] = '*'
	}
	return string(masked)
}
