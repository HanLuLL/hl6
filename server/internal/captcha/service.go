// Package captcha 提供基于 mojocn/base64Captcha 的自托管图形验证码服务。
//
// 设计要点：
//   - 完全自托管，无第三方依赖，国内国外都稳定
//   - 验证码存储使用内存（base64Captcha 默认 store），单实例部署足够
//   - 多实例部署可改用 Redis store（base64Captcha 支持）
//   - 验证码 10 分钟过期，验证后自动销毁（一次性）
//   - 通过 SystemConfig 控制启用/禁用，管理员可在后台切换
package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/repository"
)

const (
	// SystemConfig 配置键
	ConfigKeyEnabled = "auth.captcha.enabled"
	// 验证码默认有效期
	defaultTTL = 10 * time.Minute
)

// Service 提供验证码生成、校验、配置加载能力。
// 注意：本服务依赖 base64Captcha 全局 store（内存），不需要 db 连接即可工作；
// 但加载配置需要 repository。
type Service struct {
	repo *repository.Repository
}

// NewService 构造验证码服务。
func NewService(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// IsEnabled 返回当前验证码是否启用。
// 未配置时默认禁用（兼容现有部署）。
func (s *Service) IsEnabled(ctx context.Context) bool {
	if s == nil || s.repo == nil {
		return false
	}
	val, err := s.repo.GetSystemConfig(ConfigKeyEnabled)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false
		}
		return false
	}
	return strings.EqualFold(strings.TrimSpace(val), "true") ||
		strings.TrimSpace(val) == "1"
}

// GenerateResponse 是返回给前端的验证码数据。
type GenerateResponse struct {
	CaptchaID  string `json:"captcha_id"`  // 验证码 ID（前端提交时回传）
	Image      string `json:"image"`       // base64 编码的 PNG 图片（含 data: 前缀，可直接 <img src=>）
	Enabled    bool   `json:"enabled"`     // 当前是否启用（前端据此决定是否显示）
	TTLSeconds int    `json:"ttl_seconds"` // 有效期（秒）
}

// Generate 生成一个新验证码并返回前端展示所需数据。
// 即使 IsEnabled=false 也应返回 enabled=false，让前端知道不用显示。
func (s *Service) Generate(ctx context.Context) (*GenerateResponse, error) {
	if !s.IsEnabled(ctx) {
		return &GenerateResponse{Enabled: false}, nil
	}
	id, imageBase64, err := generateCaptcha()
	if err != nil {
		return nil, fmt.Errorf("generate captcha: %w", err)
	}
	return &GenerateResponse{
		CaptchaID:  id,
		Image:      imageBase64,
		Enabled:    true,
		TTLSeconds: int(defaultTTL.Seconds()),
	}, nil
}

// Verify 校验用户输入的验证码。
// verify=true 时自动销毁验证码（一次性），verify=false 时仅查询不销毁（可用于前端实时校验）。
// 如果验证码服务未启用，直接返回 true（放行）。
func (s *Service) Verify(ctx context.Context, captchaID, captchaCode string) bool {
	if !s.IsEnabled(ctx) {
		return true
	}
	if captchaID == "" || captchaCode == "" {
		return false
	}
	return verifyCaptcha(captchaID, captchaCode)
}

// PublicConfig 返回给前端的公开配置（不含敏感信息）。
type PublicConfig struct {
	Enabled bool `json:"enabled"`
}

// GetPublicConfig 返回验证码是否启用。
func (s *Service) GetPublicConfig(ctx context.Context) PublicConfig {
	return PublicConfig{Enabled: s.IsEnabled(ctx)}
}

// adminConfigItem 用于后台管理面板展示/编辑配置。
type AdminConfig struct {
	Enabled bool `json:"enabled"`
}

// GetAdminConfig 加载后台管理用的完整配置。
func (s *Service) GetAdminConfig(ctx context.Context) (*AdminConfig, error) {
	val, err := s.repo.GetSystemConfig(ConfigKeyEnabled)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &AdminConfig{
		Enabled: strings.EqualFold(strings.TrimSpace(val), "true") || strings.TrimSpace(val) == "1",
	}, nil
}

// SetAdminConfig 保存后台管理配置。
func (s *Service) SetAdminConfig(ctx context.Context, cfg *AdminConfig) error {
	value := "false"
	if cfg.Enabled {
		value = "true"
	}
	return s.repo.SetSystemConfig(ConfigKeyEnabled, value)
}

// MarshalJSON 辅助序列化 SystemConfig 列表（供审计日志使用）。
func MarshalConfigForAudit(cfg *AdminConfig) json.RawMessage {
	b, _ := json.Marshal(cfg)
	return b
}
