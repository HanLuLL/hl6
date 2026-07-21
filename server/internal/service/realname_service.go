package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"
)

// 实名认证配置 key 常量
const (
	ConfigKeyRealnameEnabled            = "realname_enabled"
	ConfigKeyRealnameRequiredForClaim   = "realname_required_for_claim"
	ConfigKeyRealnameRequiredForPayment = "realname_required_for_payment"
	ConfigKeyRealnameFee                = "realname_fee"
	ConfigKeyRealnameProvider           = "realname_provider"
	ConfigKeyRealnameAppCode            = "realname_app_code"   // 阿里云市场 AppCode
	ConfigKeyRealnameAppKey             = "realname_app_key"     // 聚合数据 API Key
	ConfigKeyRealnameFaceEnabled        = "realname_face_enabled"
)

// RealnameConfig 实名认证运行时配置。
type RealnameConfig struct {
	Enabled            bool
	RequiredForClaim   bool
	RequiredForPayment bool
	Fee                float64
	Provider           string // aliyun / juhe / manual
	AppCode            string // 阿里云市场 AppCode（明文）
	AppKey             string // 聚合数据 API Key（明文）
	FaceEnabled        bool
}

// LoadRealnameConfig 从 SystemConfig 读取实名配置并解密敏感字段。
func (s *RealnameService) LoadRealnameConfig() (*RealnameConfig, error) {
	return loadRealnameConfig(s.repo, s.encryptionKey)
}

// loadRealnameConfig 独立函数，便于在 RealnameService 实例化前（如 handler 校验场景）调用。
func loadRealnameConfig(repo *repository.Repository, encryptionKey []byte) (*RealnameConfig, error) {
	keys := []string{
		ConfigKeyRealnameEnabled,
		ConfigKeyRealnameRequiredForClaim,
		ConfigKeyRealnameRequiredForPayment,
		ConfigKeyRealnameFee,
		ConfigKeyRealnameProvider,
		ConfigKeyRealnameAppCode,
		ConfigKeyRealnameAppKey,
		ConfigKeyRealnameFaceEnabled,
	}
	values, err := repo.GetSystemConfigsByKeys(keys)
	if err != nil {
		return nil, err
	}
	cfg := &RealnameConfig{
		Enabled:            values[ConfigKeyRealnameEnabled] == "true",
		RequiredForClaim:   values[ConfigKeyRealnameRequiredForClaim] == "true",
		RequiredForPayment: values[ConfigKeyRealnameRequiredForPayment] == "true",
		Provider:           strings.TrimSpace(values[ConfigKeyRealnameProvider]),
		FaceEnabled:        values[ConfigKeyRealnameFaceEnabled] == "true",
	}
	if fee, err := strconv.ParseFloat(strings.TrimSpace(values[ConfigKeyRealnameFee]), 64); err == nil {
		cfg.Fee = fee
	}
	if cfg.Provider == "" {
		cfg.Provider = model.RealnameProviderManual
	}
	// 解密 AppCode / AppKey（管理员保存时已通过 crypto.EncryptIfKey 加密入库）
	if enc := values[ConfigKeyRealnameAppCode]; enc != "" {
		cfg.AppCode = crypto.DecryptOrPlaintext(enc, encryptionKey)
	}
	if enc := values[ConfigKeyRealnameAppKey]; enc != "" {
		cfg.AppKey = crypto.DecryptOrPlaintext(enc, encryptionKey)
	}
	return cfg, nil
}

// LoadRealnameConfigGlobal 顶层函数：供 handler 在没有 RealnameService 的场景下读取配置使用。
func LoadRealnameConfigGlobal(repo *repository.Repository, encryptionKey []byte) (*RealnameConfig, error) {
	return loadRealnameConfig(repo, encryptionKey)
}

// RealnameService 实名认证服务。
type RealnameService struct {
	repo          *repository.Repository
	encryptionKey []byte
	httpClient    *http.Client
}

// NewRealnameService 创建实名认证服务。
func NewRealnameService(repo *repository.Repository, encryptionKey []byte) *RealnameService {
	return &RealnameService{
		repo:          repo,
		encryptionKey: encryptionKey,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}
}

// SubmitApplicationInput 提交申请的输入参数。
type SubmitApplicationInput struct {
	UserID          uint
	RealName        string
	IDCard          string
	VerificationType string // idcard / face
}

// SubmitApplicationResult 提交申请的结果。
type SubmitApplicationResult struct {
	ApplicationID uint
	OrderID       *uint
	OrderNo       string
	PayURL        string
	Fee           float64
	NeedPay       bool
	// 直接验证结果（NeedPay=false 且 provider!=manual 时返回）
	Verified bool
	Message  string
}

// SubmitApplication 用户提交实名申请。
// 流程：
//  1. 校验输入与用户当前状态
//  2. 创建 RealnameApplication
//  3. 若 Fee>0：创建 PaymentOrder(Type=realname) 关联申请单，返回支付链接
//  4. 若 Fee==0：直接触发验证（同步返回结果或进入人工审核队列）
func (s *RealnameService) SubmitApplication(ctx context.Context, input SubmitApplicationInput, pay PayURLBuilder) (*SubmitApplicationResult, error) {
	// 校验
	realName := strings.TrimSpace(input.RealName)
	idCard := strings.TrimSpace(input.IDCard)
	if realName == "" || idCard == "" {
		return nil, ErrRealnameInvalidInput
	}
	if !isValidIDCard(idCard) {
		return nil, ErrRealnameInvalidIDCard
	}
	if !isValidRealName(realName) {
		return nil, ErrRealnameInvalidName
	}
	// 检查是否存在未终态申请
	hasPending, err := s.repo.HasPendingRealnameApplication(input.UserID)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, ErrRealnamePendingExists
	}

	cfg, err := s.LoadRealnameConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrRealnameDisabled
	}

	// 加密姓名和身份证
	encName, err := crypto.EncryptIfKey(realName, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt realname failed: %w", err)
	}
	encIDCard, err := crypto.EncryptIfKey(idCard, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt idcard failed: %w", err)
	}

	verificationType := input.VerificationType
	if verificationType == "" {
		verificationType = model.RealnameVerifyIDCard
	}
	if verificationType == model.RealnameVerifyFace && !cfg.FaceEnabled {
		verificationType = model.RealnameVerifyIDCard // 降级
	}

	app := &model.RealnameApplication{
		UserID:           input.UserID,
		RealName:         encName,
		IDCard:           encIDCard,
		Provider:         cfg.Provider,
		VerificationType: verificationType,
		Status:           model.RealnameAppStatusPendingPayment,
	}

	// 不需要付费：直接进入验证流程
	if cfg.Fee <= 0 {
		app.Status = model.RealnameAppStatusVerifying
		if err := s.repo.CreateRealnameApplication(app); err != nil {
			return nil, err
		}
		// 同步触发验证（manual 模式会直接落库为 pending 等待管理员审核）
		verified, msg, vErr := s.verifyAndFinalize(ctx, app, realName, idCard)
		if vErr != nil {
			slog.Warn("realname: verify failed", "app_id", app.ID, "err", vErr)
			return &SubmitApplicationResult{
				ApplicationID: app.ID,
				Fee:           0,
				NeedPay:       false,
				Verified:      false,
				Message:       msg,
			}, nil
		}
		return &SubmitApplicationResult{
			ApplicationID: app.ID,
			Fee:           0,
			NeedPay:       false,
			Verified:      verified,
			Message:       msg,
		}, nil
	}

	// 需要付费：创建支付订单 + 申请单
	orderNo := fmt.Sprintf("R%d%d", time.Now().UnixMilli(), input.UserID)
	expiredAt := time.Now().Add(30 * time.Minute)
	order := &model.PaymentOrder{
		UserID:        input.UserID,
		OrderNo:       orderNo,
		Gateway:       pay.Gateway(),
		PaymentMethod: pay.Method(),
		Amount:        cfg.Fee,
		Credits:       0,
		Type:          model.PaymentOrderTypeRealname,
		Status:        model.PaymentStatusPending,
		ExpiredAt:     &expiredAt,
	}

	app.OrderID = &order.ID // 占位，事务内会回填

	// 事务：创建订单 + 创建申请单 + 回填 order_id + 生成支付 URL
	var payURL string
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		app.OrderID = &order.ID
		if err := tx.Create(app).Error; err != nil {
			return err
		}
		// 回填 reference_id 到订单
		if err := tx.Model(order).Update("reference_id", app.ID).Error; err != nil {
			return err
		}
		order.ReferenceID = &app.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 生成支付 URL（在事务外调用，避免支付网关 IO 拖长事务）
	payURL, err = pay.BuildURL(orderNo, cfg.Fee, fmt.Sprintf("实名认证费 %.2f", cfg.Fee))
	if err != nil {
		slog.Warn("realname: build pay url failed", "app_id", app.ID, "err", err)
		return nil, ErrRealnamePayURLFailed
	}
	// 持久化 payURL
	if err := s.repo.GetDB().Model(order).Update("pay_url", payURL).Error; err != nil {
		slog.Warn("realname: persist pay url failed", "app_id", app.ID, "err", err)
	}

	return &SubmitApplicationResult{
		ApplicationID: app.ID,
		OrderID:       &order.ID,
		OrderNo:       orderNo,
		PayURL:        payURL,
		Fee:           cfg.Fee,
		NeedPay:       true,
	}, nil
}

// PayURLBuilder 由 handler 层提供，用于生成支付链接。
type PayURLBuilder interface {
	Gateway() string
	Method() string
	BuildURL(orderNo string, amount float64, productName string) (string, error)
}

// HandlePaymentSuccess 支付成功后触发实名验证。
// 在 processPayment 的事务内调用，仅更新状态；实际第三方 API 调用在事务外异步执行。
func (s *RealnameService) HandlePaymentSuccess(tx *gorm.DB, orderID uint, referenceID *uint) error {
	if referenceID == nil {
		return errors.New("realname: reference_id is nil")
	}
	app, err := s.repo.FindRealnameApplicationForUpdate(tx, *referenceID)
	if err != nil {
		return fmt.Errorf("find application failed: %w", err)
	}
	if app == nil {
		return errors.New("realname: application not found")
	}
	if app.OrderID == nil || *app.OrderID != orderID {
		return errors.New("realname: order_id mismatch")
	}
	if app.Status != model.RealnameAppStatusPendingPayment {
		// 已处理过的订单（重复回调），幂等返回
		return nil
	}
	// 仅更新状态为 paid，第三方 API 调用由异步任务触发
	now := time.Now()
	return tx.Model(app).Updates(map[string]interface{}{
		"status":     model.RealnameAppStatusPaid,
		"updated_at": now,
	}).Error
}

// StartVerification 触发第三方 API 验证（事务外调用）。
// 应在 HandlePaymentSuccess 之后由调用方异步调用。
func (s *RealnameService) StartVerification(ctx context.Context, applicationID uint) error {
	app, err := s.repo.FindRealnameApplication(applicationID)
	if err != nil {
		return err
	}
	if app == nil {
		return errors.New("realname: application not found")
	}
	if app.Status != model.RealnameAppStatusPaid && app.Status != model.RealnameAppStatusFailed {
		return nil // 非 paid/failed 状态不重复触发
	}

	// 解密姓名和身份证
	realName := crypto.DecryptOrPlaintext(app.RealName, s.encryptionKey)
	idCard := crypto.DecryptOrPlaintext(app.IDCard, s.encryptionKey)

	_, _, err = s.verifyAndFinalize(ctx, app, realName, idCard)
	return err
}

// verifyAndFinalize 调用第三方 API 并落库最终结果。
func (s *RealnameService) verifyAndFinalize(ctx context.Context, app *model.RealnameApplication, realName, idCard string) (bool, string, error) {
	cfg, err := s.LoadRealnameConfig()
	if err != nil {
		return false, "", err
	}

	// manual 模式：标记为 paid 等待管理员审核
	if cfg.Provider == model.RealnameProviderManual || cfg.Provider == "" {
		now := time.Now()
		if err := s.repo.UpdateRealnameApplicationStatus(app.ID, model.RealnameAppStatusPaid, ""); err != nil {
			slog.Warn("realname: mark as paid failed", "app_id", app.ID, "err", err)
		}
		if err := s.repo.Transaction(func(tx *gorm.DB) error {
			return s.repo.UpdateUserRealnameStatus(tx, app.UserID, model.RealnameStatusPending, "", nil)
		}); err != nil {
			slog.Warn("realname: update user status to pending failed", "app_id", app.ID, "err", err)
		}
		return false, "waiting_admin_review", nil
	}

	// 更新状态为 verifying
	if err := s.repo.UpdateRealnameApplicationStatus(app.ID, model.RealnameAppStatusVerifying, ""); err != nil {
		slog.Warn("realname: mark as verifying failed", "app_id", app.ID, "err", err)
	}

	// 调用第三方 API
	result, err := s.callProvider(ctx, cfg, app.VerificationType, realName, idCard)
	if err != nil {
		// 网络错误等：标记为 failed，可重试
		if err := s.repo.UpdateRealnameApplicationStatus(app.ID, model.RealnameAppStatusFailed, err.Error()); err != nil {
			slog.Warn("realname: mark as failed failed", "app_id", app.ID, "err", err)
		}
		slog.Warn("realname: provider call failed", "app_id", app.ID, "provider", cfg.Provider, "err", err)
		return false, "provider_error", err
	}

	// 持久化 provider 返回
	resultJSON, _ := json.Marshal(result)
	now := time.Now()

	if !result.Matched {
		// 身份证不匹配：rejected
		if err := s.repo.GetDB().Model(app).Updates(map[string]interface{}{
			"status":          model.RealnameAppStatusRejected,
			"provider_result": resultJSON,
			"reject_reason":   result.Message,
			"updated_at":      now,
		}).Error; err != nil {
			slog.Warn("realname: mark as rejected failed", "app_id", app.ID, "err", err)
		}
		if err := s.repo.Transaction(func(tx *gorm.DB) error {
			return s.repo.UpdateUserRealnameStatus(tx, app.UserID, model.RealnameStatusRejected, "", nil)
		}); err != nil {
			slog.Warn("realname: update user status to rejected failed", "app_id", app.ID, "err", err)
		}
		return false, "idcard_not_match", nil
	}

	// 验证通过
	verifiedAt := now
	maskedName := model.MaskRealName(realName)
	if err := s.repo.GetDB().Model(app).Updates(map[string]interface{}{
		"status":          model.RealnameAppStatusVerified,
		"provider_result": resultJSON,
		"verified_at":     &verifiedAt,
		"updated_at":      now,
	}).Error; err != nil {
		slog.Warn("realname: mark as verified failed", "app_id", app.ID, "err", err)
	}
	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateUserRealnameStatus(tx, app.UserID, model.RealnameStatusVerified, maskedName, &verifiedAt)
	}); err != nil {
		slog.Warn("realname: update user status to verified failed", "app_id", app.ID, "err", err)
	}
	return true, "verified", nil
}

// ProviderVerifyResult 第三方 API 返回的统一结果。
type ProviderVerifyResult struct {
	Matched bool   `json:"matched"`
	Message string `json:"message"`
	Raw     any    `json:"raw,omitempty"`
}

// callProvider 根据配置分发到对应的第三方 API。
func (s *RealnameService) callProvider(ctx context.Context, cfg *RealnameConfig, verifyType, realName, idCard string) (*ProviderVerifyResult, error) {
	switch cfg.Provider {
	case model.RealnameProviderAliyun:
		return s.callAliyun(ctx, cfg, realName, idCard)
	case model.RealnameProviderJuhe:
		return s.callJuhe(ctx, cfg, realName, idCard)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

// callAliyun 调用阿里云市场身份证二要素核验。
// API 文档：https://market.aliyun.com/products/57126001/cmapi00040192.html
func (s *RealnameService) callAliyun(ctx context.Context, cfg *RealnameConfig, realName, idCard string) (*ProviderVerifyResult, error) {
	if cfg.AppCode == "" {
		return nil, errors.New("aliyun appcode not configured")
	}
	endpoint := "https://idcert.market.alicloudapi.com/idcard"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "APPCODE "+cfg.AppCode)
	q := req.URL.Query()
	q.Set("idCard", idCard)
	q.Set("name", realName)
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aliyun api status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			IsMatch int    `json:"isMatch"`
			Sex     string `json:"sex"`
			Birth   string `json:"birth"`
			Address string `json:"address"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse aliyun response failed: %w", err)
	}
	matched := parsed.Code == "0" && parsed.Data.IsMatch == 1
	msg := parsed.Msg
	if !matched && msg == "" {
		msg = "idcard_not_match"
	}
	return &ProviderVerifyResult{
		Matched: matched,
		Message: msg,
		Raw:     parsed,
	}, nil
}

// callJuhe 调用聚合数据身份证实名认证。
// API 文档：https://www.juhe.cn/docs/api/id/103
func (s *RealnameService) callJuhe(ctx context.Context, cfg *RealnameConfig, realName, idCard string) (*ProviderVerifyResult, error) {
	if cfg.AppKey == "" {
		return nil, errors.New("juhe appkey not configured")
	}
	endpoint := "http://op.juhe.cn/idcard/query"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("key", cfg.AppKey)
	q.Set("idcard", idCard)
	q.Set("realname", realName)
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("juhe api status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		ResultCode string `json:"resultcode"`
		Reason     string `json:"reason"`
		Result     struct {
			Res int    `json:"res"`
			Sex string `json:"sex"`
			Birth string `json:"birth"`
			Address string `json:"address"`
		} `json:"result"`
		ErrorCode int `json:"error_code"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse juhe response failed: %w", err)
	}
	matched := parsed.ResultCode == "200" && parsed.Result.Res == 1
	msg := parsed.Reason
	if !matched && msg == "" {
		msg = "idcard_not_match"
	}
	return &ProviderVerifyResult{
		Matched: matched,
		Message: msg,
		Raw:     parsed,
	}, nil
}

// AdminReviewInput 管理员人工审核输入。
type AdminReviewInput struct {
	ApplicationID uint
	ReviewerID    uint
	Approved      bool
	Reason        string
}

// AdminReview 管理员人工审核（仅对 manual 模式或 failed 状态有效）。
func (s *RealnameService) AdminReview(ctx context.Context, input AdminReviewInput) error {
	app, err := s.repo.FindRealnameApplication(input.ApplicationID)
	if err != nil {
		return err
	}
	if app == nil {
		return ErrRealnameNotFound
	}
	// 仅允许 paid / verifying / failed / rejected 状态进入人工审核
	switch app.Status {
	case model.RealnameAppStatusPaid, model.RealnameAppStatusVerifying,
		model.RealnameAppStatusFailed, model.RealnameAppStatusRejected:
	default:
		return ErrRealnameInvalidStatus
	}

	realName := crypto.DecryptOrPlaintext(app.RealName, s.encryptionKey)
	now := time.Now()
	reason := strings.TrimSpace(input.Reason)

	return s.repo.Transaction(func(tx *gorm.DB) error {
		if input.Approved {
			if err := tx.Model(app).Updates(map[string]interface{}{
				"status":       model.RealnameAppStatusVerified,
				"reject_reason": "",
				"reviewed_by":   input.ReviewerID,
				"reviewed_at":   &now,
				"verified_at":   &now,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
			maskedName := model.MaskRealName(realName)
			return s.repo.UpdateUserRealnameStatus(tx, app.UserID, model.RealnameStatusVerified, maskedName, &now)
		}
		if err := tx.Model(app).Updates(map[string]interface{}{
			"status":        model.RealnameAppStatusRejected,
			"reject_reason": reason,
			"reviewed_by":   input.ReviewerID,
			"reviewed_at":   &now,
			"updated_at":    now,
		}).Error; err != nil {
			return err
		}
		return s.repo.UpdateUserRealnameStatus(tx, app.UserID, model.RealnameStatusRejected, "", nil)
	})
}

// RetryVerification 重试失败的第三方 API 调用。
func (s *RealnameService) RetryVerification(ctx context.Context, applicationID uint) error {
	return s.StartVerification(ctx, applicationID)
}

// --- 输入校验 ---

func isValidIDCard(id string) bool {
	if len(id) != 18 {
		return false
	}
	// 简单校验：前 17 位数字 + 末位数字或 X
	for i := 0; i < 17; i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	last := id[17]
	if !((last >= '0' && last <= '9') || last == 'X' || last == 'x') {
		return false
	}
	return true
}

func isValidRealName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 30 {
		return false
	}
	for _, r := range name {
		// 允许中文、英文字母、点、中间点（·）
		if r >= 0x4e00 && r <= 0x9fff {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if r == '.' || r == '·' || r == ' ' {
			continue
		}
		return false
	}
	return true
}

// --- 错误定义 ---

var (
	ErrRealnameDisabled      = errors.New("realname authentication is disabled")
	ErrRealnameInvalidInput  = errors.New("invalid realname input")
	ErrRealnameInvalidIDCard = errors.New("invalid id card number")
	ErrRealnameInvalidName   = errors.New("invalid real name")
	ErrRealnamePendingExists = errors.New("a pending realname application already exists")
	ErrRealnameNotFound      = errors.New("realname application not found")
	ErrRealnameInvalidStatus = errors.New("realname application status is not reviewable")
	ErrRealnamePayURLFailed  = errors.New("failed to build payment url")
)
