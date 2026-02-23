package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	LogtoID   string     `json:"logto_id" gorm:"uniqueIndex;not null"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	AvatarURL string     `json:"avatar_url"`
	Role      string     `json:"role" gorm:"default:user"`
	GroupID   *uint      `json:"group_id" gorm:"index"`
	Group     *UserGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type UserGroup struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DomainGroupAccess struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	DomainID      uint      `json:"domain_id" gorm:"uniqueIndex:idx_domain_group;not null"`
	GroupID       uint      `json:"group_id" gorm:"uniqueIndex:idx_domain_group;not null"`
	CreditCost    Credit    `json:"credit_cost" gorm:"type:bigint;not null;default:10"`
	MaxDNSRecords *int      `json:"max_dns_records" gorm:"default:null"` // nil = 无限制
	Group         UserGroup `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SystemConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	Value     string    `json:"value" gorm:"not null"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Domain struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	Name                string    `json:"name" gorm:"uniqueIndex;not null"`
	CloudflareZoneID    string    `json:"cloudflare_zone_id" gorm:"not null"`
	CloudflareAccountID uint      `json:"cloudflare_account_id" gorm:"not null;default:0"`
	CreditCost          Credit    `json:"credit_cost" gorm:"type:bigint;default:10"`
	IsActive            bool      `json:"is_active" gorm:"default:true"`
	Description         string    `json:"description"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Subdomain struct {
	ID         uint        `json:"id" gorm:"primaryKey"`
	DomainID   uint        `json:"domain_id" gorm:"uniqueIndex:idx_domain_name;not null"`
	UserID     uint        `json:"user_id" gorm:"index;not null"`
	Name       string      `json:"name" gorm:"uniqueIndex:idx_domain_name;not null"`
	FQDN       string      `json:"fqdn" gorm:"uniqueIndex;not null"`
	ClaimCost  Credit      `json:"claim_cost" gorm:"type:bigint;default:0"`
	Domain     Domain      `json:"domain" gorm:"foreignKey:DomainID"`
	User       User        `json:"-" gorm:"foreignKey:UserID"`
	DNSRecords []DNSRecord `json:"dns_records,omitempty" gorm:"foreignKey:SubdomainID"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type DNSRecord struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	SubdomainID        uint      `json:"subdomain_id" gorm:"index;not null"`
	Type               string    `json:"type" gorm:"not null"`
	Name               string    `json:"name" gorm:"not null"`
	Content            string    `json:"content" gorm:"not null"`
	TTL                int       `json:"ttl" gorm:"default:1"`
	Proxied            bool      `json:"proxied" gorm:"default:false"`
	CloudflareRecordID string    `json:"cloudflare_record_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreditBalance struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"uniqueIndex;not null"`
	Balance   Credit    `json:"balance" gorm:"type:bigint;default:0"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreditTransaction struct {
	ID                uint            `json:"id" gorm:"primaryKey"`
	UserID            uint            `json:"user_id" gorm:"index;not null"`
	Amount            Credit          `json:"amount" gorm:"type:bigint;not null"`
	Type              string          `json:"type" gorm:"not null"`
	DescriptionKey    string          `json:"description_key"`
	DescriptionParams json.RawMessage `json:"description_params,omitempty" gorm:"type:jsonb"`
	BalanceAfter      Credit          `json:"balance_after" gorm:"type:bigint"`
	CreatedAt         time.Time       `json:"created_at"`
}

type AuditLog struct {
	ID         uint             `json:"id" gorm:"primaryKey"`
	UserID     uint             `json:"user_id" gorm:"index;not null"`
	Action     string           `json:"action" gorm:"not null"`
	Resource   string           `json:"resource"`
	ResourceID uint             `json:"resource_id"`
	Details    json.RawMessage  `json:"details" gorm:"type:jsonb"`
	User       User             `json:"user,omitempty" gorm:"foreignKey:UserID"`
	CreatedAt  time.Time        `json:"created_at"`
}

type CloudflareAccount struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	ApiToken  string    `json:"-" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CloudflareAccountView struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	TokenHint string    `json:"token_hint"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *CloudflareAccount) ToView() CloudflareAccountView {
	hint := ""
	if len(a.ApiToken) >= 4 {
		hint = "..." + a.ApiToken[len(a.ApiToken)-4:]
	}
	return CloudflareAccountView{
		ID:        a.ID,
		Name:      a.Name,
		TokenHint: hint,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
