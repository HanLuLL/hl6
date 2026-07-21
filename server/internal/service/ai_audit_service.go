package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/crypto"
)

// DefaultAuditSystemPrompt 默认审查系统提示词。
const DefaultAuditSystemPrompt = `你是一个专业的内容安全审查助手，负责审查网站内容是否违反平台规则。

你需要审查的内容类型包括：
- 网页文本内容（HTML 渲染后的纯文本）
- 网页标题
- URL 指向的目标内容
- DNS 记录指向的内容

请识别以下类型的违规内容：
1. 暴力与威胁：包含暴力行为、恐吓、威胁他人安全的内容
2. 色情与露骨内容：包含色情、淫秽或不当性内容
3. 仇恨言论：基于种族、宗教、性别、性取向等歧视或攻击性言论
4. 敏感政治内容：煽动颠覆、分裂国家、危害国家安全的内容
5. 恶意代码：包含恶意脚本、钓鱼页面、木马分发等
6. 诈骗与欺诈：虚假宣传、网络诈骗、传销等
7. 违法信息：赌博、毒品、武器交易等违法内容
8. 侵权内容：盗版软件、侵犯知识产权等

你必须以 JSON 格式返回审查结果，格式如下：
{
  "judgment": "clean 或 violation",
  "violation_types": ["类型1", "类型2"],
  "confidence": 0.0 到 1.0 的置信度,
  "suggested_action": "observe 或 site 或 user",
  "reason": "判断理由的简要说明"
}

判定标准：
- judgment 为 "clean" 表示内容安全无违规
- judgment 为 "violation" 表示检测到违规内容
- suggested_action 处置建议：
  - observe: 仅记录，不执行操作（轻微违规或不确定时）
  - site: 释放/封禁该子域名（中等违规）
  - user: 封禁该用户的所有子域名（严重违规）
- confidence: 0-1 之间的置信度分数
- 当 confidence 低于 0.6 时，建议使用 observe 行动

重要：只返回 JSON，不要返回其他内容。`

// DefaultAuditUserPrompt 默认用户提示词模板。
// 支持变量：{{fqdn}}、{{title}}、{{content}}、{{url}}
const DefaultAuditUserPrompt = `请审查以下网站内容：

域名：{{fqdn}}
URL：{{url}}
标题：{{title}}

内容摘要：
{{content}}

请判断以上内容是否违规，并以 JSON 格式返回审查结果。`

// AIAuditService AI 审查服务，协调 LLM 调用、审查记录和自动处置。
type AIAuditService struct {
	repo          *repository.Repository
	llm           *AILLMService
	auditSvc      *AuditService
	auditLog      *AuditLogService
	encryptionKey []byte
}

// NewAIAuditService 创建 AI 审查服务。
func NewAIAuditService(
	repo *repository.Repository,
	llm *AILLMService,
	auditSvc *AuditService,
	auditLog *AuditLogService,
	encryptionKey []byte,
) *AIAuditService {
	return &AIAuditService{
		repo:          repo,
		llm:           llm,
		auditSvc:      auditSvc,
		auditLog:      auditLog,
		encryptionKey: encryptionKey,
	}
}

// AIAuditInput AI 审查输入。
type AIAuditInput struct {
	SubdomainID uint
	ScanID      *uint
	FQDN        string
	Title       string
	Content     string // 网页内容（截断后）
	URL         string
}

// ReviewSubdomain 对子域名内容执行 AI 审查。
func (s *AIAuditService) ReviewSubdomain(ctx context.Context, input AIAuditInput) (*model.AuditAIReview, error) {
	// 1. 获取默认的 AI 模型配置
	modelConfig, err := s.repo.GetDefaultAIModelConfig()
	if err != nil || modelConfig == nil {
		slog.Warn("AI audit: no default model config", "err", err)
		return nil, fmt.Errorf("no default AI model config: %w", err)
	}
	if !modelConfig.IsEnabled {
		return nil, fmt.Errorf("default AI model config is disabled")
	}

	// 2. 获取启用的提示词模板（优先级最高）
	promptTemplate, err := s.repo.GetHighestPriorityPromptTemplate()
	if err != nil {
		slog.Warn("AI audit: no prompt template, using default", "err", err)
	}

	systemPrompt := DefaultAuditSystemPrompt
	userPromptTemplate := DefaultAuditUserPrompt
	var promptTemplateID *uint

	if promptTemplate != nil && promptTemplate.IsEnabled {
		systemPrompt = promptTemplate.SystemPrompt
		userPromptTemplate = promptTemplate.UserPrompt
		promptTemplateID = &promptTemplate.ID
	}

	// 3. 渲染用户提示词
	userPrompt := renderAuditPrompt(userPromptTemplate, map[string]string{
		"{{fqdn}}":   input.FQDN,
		"{{title}}":  input.Title,
		"{{content}}": truncateContent(input.Content, 8000), // 限制内容长度
		"{{url}}":    input.URL,
	})

	// 4. 截断输入内容用于存储
	inputSummary := truncateContent(input.Content, 2000)

	// 5. 调用 LLM
	result := s.llm.SendChatRequest(ctx, modelConfig, systemPrompt, userPrompt)

	review := &model.AuditAIReview{
		SubdomainID:       input.SubdomainID,
		ScanID:            input.ScanID,
		FQDN:              input.FQDN,
		ModelConfigID:     modelConfig.ID,
		PromptTemplateID:  promptTemplateID,
		InputContent:      inputSummary,
		AdminReviewStatus: model.AdminReviewPending,
	}

	if result.Error != nil {
		review.AIJudgment = model.AIJudgmentError
		review.AIResponse = truncateContent(result.Error.Error(), 1000)
		slog.Error("AI audit: LLM request failed", "fqdn", input.FQDN, "err", result.Error)
	} else {
		review.AIResponse = result.Response
		review.TokensUsed = result.TokensUsed

		judgment, violationTypes, confidence, suggestedAction, _ := ParseAIJudgment(result.Response)
		review.AIJudgment = judgment
		review.ViolationTypes = model.StringSlice(violationTypes)
		review.AIConfidence = confidence
		review.AISuggestedAction = suggestedAction

		slog.Info("AI audit: review completed",
			"fqdn", input.FQDN,
			"judgment", judgment,
			"confidence", confidence,
			"suggested_action", suggestedAction,
			"tokens", result.TokensUsed,
			"duration", result.Duration,
		)
	}

	// 6. 保存审查记录
	if err := s.repo.CreateAuditAIReview(review); err != nil {
		slog.Error("AI audit: failed to save review", "fqdn", input.FQDN, "err", err)
		return nil, err
	}

	// 7. 如果 AI 判定为违规，根据置信度决定是否自动处置
	if review.AIJudgment == model.AIJudgmentViolation {
		s.handleViolation(ctx, input, review)
	}

	return review, nil
}

// handleViolation 处理 AI 判定的违规内容。
func (s *AIAuditService) handleViolation(ctx context.Context, input AIAuditInput, review *model.AuditAIReview) {
	confidence := review.AIConfidence
	action := review.AISuggestedAction

	slog.Warn("AI audit: violation detected",
		"fqdn", input.FQDN,
		"confidence", confidence,
		"suggested_action", action,
		"violation_types", review.ViolationTypes,
	)

	// 低置信度（<0.6）仅记录，不自动处置
	if confidence < 0.6 {
		review.FinalAction = "observe_low_confidence"
		s.updateReviewAction(review)
		return
	}

	// 根据处置建议执行
	switch action {
	case "user":
		// 高置信度 + 严重违规 → 自动封禁用户所有子域名
		if confidence >= 0.8 {
			review.FinalAction = model.AuditActionUser
			s.autoSuspendUser(ctx, input, review)
		} else {
			// 中等置信度 → 仅封禁该子域名，等待管理员二次审核
			review.FinalAction = model.AuditActionSite
			s.autoSuspendSubdomain(ctx, input, review)
		}
	case "site":
		if confidence >= 0.7 {
			review.FinalAction = model.AuditActionSite
			s.autoSuspendSubdomain(ctx, input, review)
		} else {
			review.FinalAction = "observe_pending_review"
			s.updateReviewAction(review)
		}
	default:
		// observe 或其他 → 仅记录
		review.FinalAction = "observe"
		s.updateReviewAction(review)
	}
}

func (s *AIAuditService) autoSuspendUser(ctx context.Context, input AIAuditInput, review *model.AuditAIReview) {
	sub, err := s.repo.FindSubdomain(input.SubdomainID)
	if err != nil {
		slog.Error("AI audit: subdomain not found for user suspend", "id", input.SubdomainID, "err", err)
		return
	}

	// 构造一个虚拟 AuditRule 用于调用已有的封禁逻辑
	violationDesc := strings.Join(review.ViolationTypes, ", ")
	rule := &model.AuditRule{
		Name:   fmt.Sprintf("AI审查自动封禁: %s", violationDesc),
		Action: model.AuditActionUser,
	}

	s.auditSvc.suspendUserSubdomains(ctx, sub.UserID, rule, review.AIResponse)

	// 记录审计日志
	_ = s.auditLog.RecordUser(sub.UserID, "ai_audit_auto_suspend_user", "subdomain", sub.ID, map[string]interface{}{
		"fqdn":             input.FQDN,
		"ai_review_id":     review.ID,
		"violation_types":  review.ViolationTypes,
		"confidence":       review.AIConfidence,
		"suggested_action": review.AISuggestedAction,
	})

	s.updateReviewAction(review)
}

func (s *AIAuditService) autoSuspendSubdomain(ctx context.Context, input AIAuditInput, review *model.AuditAIReview) {
	sub, err := s.repo.FindSubdomain(input.SubdomainID)
	if err != nil {
		slog.Error("AI audit: subdomain not found for site suspend", "id", input.SubdomainID, "err", err)
		return
	}

	violationDesc := strings.Join(review.ViolationTypes, ", ")
	rule := &model.AuditRule{
		Name:   fmt.Sprintf("AI审查自动封禁: %s", violationDesc),
		Action: model.AuditActionSite,
	}

	s.auditSvc.releaseSubdomainViaRule(ctx, sub, rule, review.AIResponse)

	_ = s.auditLog.RecordUser(sub.UserID, "ai_audit_auto_suspend_site", "subdomain", sub.ID, map[string]interface{}{
		"fqdn":             input.FQDN,
		"ai_review_id":     review.ID,
		"violation_types":  review.ViolationTypes,
		"confidence":       review.AIConfidence,
	})

	s.updateReviewAction(review)
}

func (s *AIAuditService) updateReviewAction(review *model.AuditAIReview) {
	if err := s.repo.UpdateAuditAIReviewFinalAction(review.ID, review.FinalAction); err != nil {
		slog.Error("AI audit: failed to update final action", "review_id", review.ID, "err", err)
	}
}

// renderAuditPrompt 渲染提示词模板中的变量。
func renderAuditPrompt(template string, vars map[string]string) string {
	result := template
	for k, v := range vars {
		result = strings.ReplaceAll(result, k, v)
	}
	return result
}

// truncateContent 截断内容到指定字符数。
func truncateContent(content string, maxChars int) string {
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content
	}
	return string(runes[:maxChars]) + "\n...[内容已截断]"
}

// 提示词模板种子数据

// DefaultPromptTemplates 返回系统默认的提示词模板列表。
func DefaultPromptTemplates(createdBy uint) []model.AuditPromptTemplate {
	return []model.AuditPromptTemplate{
		{
			Name:         "通用内容安全审查",
			IsDefault:    true,
			IsEnabled:    true,
			SortOrder:    0,
			SystemPrompt: DefaultAuditSystemPrompt,
			UserPrompt:   DefaultAuditUserPrompt,
			Description:  "默认审查模板，覆盖暴力、色情、仇恨言论、敏感政治、恶意代码等违规类型",
			CreatedBy:    createdBy,
		},
		{
			Name:      "严格审查模式",
			IsDefault: false,
			IsEnabled: true,
			SortOrder: 1,
			SystemPrompt: `你是一个严格的内容安全审查助手。你的任务是以最严格的标准审查网站内容。

任何可能违反法律或平台规则的内容都应被标记为违规。宁可误报，不可漏报。

你需要识别的违规类型与通用模板相同，但判定标准更严格：
- confidence 低于 0.4 才使用 observe
- 任何疑似违规内容都应标记为 violation
- suggested_action 应偏向 site 或 user

返回 JSON 格式：
{
  "judgment": "clean 或 violation",
  "violation_types": ["类型"],
  "confidence": 0.0-1.0,
  "suggested_action": "observe 或 site 或 user",
  "reason": "判断理由"
}`,
			UserPrompt:  DefaultAuditUserPrompt,
			Description: "严格审查模式，降低漏报率，适用于高风险域名",
			CreatedBy:   createdBy,
		},
		{
			Name:      "代码与安全审查",
			IsDefault: false,
			IsEnabled: true,
			SortOrder: 2,
			SystemPrompt: `你是一个专业的网络安全审查助手，专注于检测网站中的恶意代码和安全威胁。

你需要识别以下类型的安全威胁：
1. 恶意脚本：XSS 攻击代码、JavaScript 挖矿脚本
2. 钓鱼页面：仿冒登录页面、窃取用户凭证
3. 恶意重定向：强制跳转到恶意站点
4. 恶意软件分发：引导用户下载木马或病毒
5. 后门与 WebShell：远程代码执行后门
6. CDN 滥用：利用 CDN 分发恶意内容

返回 JSON 格式：
{
  "judgment": "clean 或 violation",
  "violation_types": ["类型"],
  "confidence": 0.0-1.0,
  "suggested_action": "observe 或 site 或 user",
  "reason": "判断理由"
}`,
			UserPrompt: `请审查以下网站的代码和安全性：

域名：{{fqdn}}
URL：{{url}}
标题：{{title}}

页面内容/代码：
{{content}}

请判断该网站是否存在安全威胁，以 JSON 格式返回结果。`,
			Description: "专注于代码安全审查，检测恶意脚本、钓鱼页面等",
			CreatedBy:   createdBy,
		},
	}
}

// GetEncryptionKey 辅助函数：获取加密密钥的十六进制解码结果。
func GetEncryptionKey(hexKey string) ([]byte, error) {
	if hexKey == "" {
		return nil, nil
	}
	return decodeHexKey(hexKey), nil
}

func decodeHexKey(s string) []byte {
	if len(s) != 64 {
		return nil
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b := hexByte(s[i*2], s[i*2+1])
		key[i] = b
	}
	return key
}

func hexByte(h, l byte) byte {
	return hexDigit(h)<<4 | hexDigit(l)
}

func hexDigit(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// CreateDefaultPromptTemplates 创建默认提示词模板（仅在表为空时执行）。
func (s *AIAuditService) CreateDefaultPromptTemplates(createdBy uint) error {
	existing, err := s.repo.CountPromptTemplates()
	if err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	templates := DefaultPromptTemplates(createdBy)
	for i := range templates {
		if err := s.repo.CreatePromptTemplate(&templates[i]); err != nil {
			slog.Error("AI audit: failed to create default prompt template", "name", templates[i].Name, "err", err)
		}
	}
	return nil
}

// DefaultAuditRules 返回系统默认的内容审查规则列表。
// 这些规则覆盖最常见的违规场景，安全可靠，可作为新部署的基线。
// 管理员可在后台「内容审查 - 规则」Tab 中编辑或禁用。
func DefaultAuditRules(createdBy uint) []model.AuditRule {
	return []model.AuditRule{
		// 1. 长期不可达回收：双次确认不可达后释放子域名，回收占用资源
		{
			Name:        "长期不可达回收",
			Enabled:     true,
			Description: "子域名连续两次抓取均不可达时释放（回收长期未使用的占用）",
			Targets:     model.StringSlice{model.AuditTargetBody},
			MatchType:   model.AuditMatchUnreachable,
			Keywords:    model.StringSlice{},
			Pattern:     "",
			Action:      model.AuditActionSite,
			BanNotifyContent: "你的子域名 {{fqdn}} 因长期不可达已被释放。" +
				"如需继续使用，请重新认领并配置可正常访问的 DNS 记录。",
			ExemptEnabled:        true,
			ExemptRecheckMinutes: 1440, // 24 小时宽限期
			ExemptNotifyContent: "你的子域名 {{fqdn}} 当前不可达。" +
				"请在 {{recheck_minutes}} 分钟内恢复访问，否则将被释放。",
			CreatedBy: createdBy,
		},
		// 2. 5xx 服务端错误告警：仅观察，不处置（可能是临时故障）
		{
			Name:        "5xx 服务端错误告警",
			Enabled:     true,
			Description: "页面返回 5xx 状态码时记录为违规，仅观察不处置（可能是临时故障）",
			Targets:     model.StringSlice{model.AuditTargetStatusCode},
			MatchType:   model.AuditMatchStatusEq,
			Keywords:    model.StringSlice{},
			Pattern:     "502",
			Action:      model.AuditActionObserve,
			ExemptEnabled:        false,
			ExemptRecheckMinutes: 0,
			CreatedBy: createdBy,
		},
		// 3. 敏感政治内容：命中即释放子域名
		{
			Name:        "敏感政治内容拦截",
			Enabled:     true,
			Description: "命中煽动颠覆、分裂国家等敏感政治关键词时释放子域名",
			Targets:     model.StringSlice{model.AuditTargetBody, model.AuditTargetTitle},
			MatchType:   model.AuditMatchKeyword,
			Keywords: model.StringSlice{
				"煽动颠覆国家政权",
				"推翻社会主义制度",
				"分裂国家",
				"颠覆国家政权",
			},
			KeywordLogic: model.AuditKeywordLogicAny,
			Pattern:      "",
			Action:       model.AuditActionSite,
			BanNotifyContent: "你的子域名 {{fqdn}} 因命中敏感政治内容关键词已被释放。" +
				"如认为误判，请提交申诉。",
			ExemptEnabled:        false,
			ExemptRecheckMinutes: 0,
			CreatedBy: createdBy,
		},
		// 4. 恶意脚本检测：命中即封禁用户所有子域名
		{
			Name:        "恶意脚本检测",
			Enabled:     true,
			Description: "检测 XSS、挖矿、WebShell 等恶意脚本特征，命中即封禁用户所有子域名",
			Targets:     model.StringSlice{model.AuditTargetBody},
			MatchType:   model.AuditMatchRegex,
			Keywords:    model.StringSlice{},
			Pattern:     "(eval\\(atob\\(|document\\.write\\(unescape\\(|coinhive|cryptonight|<script[^>]*>\\s*eval\\(|base64,PHNjcmlwd)",
			CaseSensitive: false,
			Action:        model.AuditActionUser,
			BanNotifyContent: "你的子域名 {{fqdn}} 检测到恶意脚本，已封禁账户下所有子域名。" +
				"如认为误判，请立即提交申诉并配合审查。",
			ExemptEnabled:        false,
			ExemptRecheckMinutes: 0,
			CreatedBy: createdBy,
		},
		// 5. 钓鱼登录页面检测：命中即释放子域名
		{
			Name:        "钓鱼登录页面检测",
			Enabled:     true,
			Description: "检测仿冒登录表单特征，命中即释放子域名",
			Targets:     model.StringSlice{model.AuditTargetBody},
			MatchType:   model.AuditMatchKeyword,
			Keywords: model.StringSlice{
				"password",
				"passwd",
				"密码",
			},
			KeywordLogic: model.AuditKeywordLogicAll,
			Pattern:      "",
			Action:       model.AuditActionSite,
			BanNotifyContent: "你的子域名 {{fqdn}} 检测到疑似钓鱼登录页面，已被释放。" +
				"如认为误判，请提交申诉。",
			ExemptEnabled:        false,
			ExemptRecheckMinutes: 0,
			CreatedBy: createdBy,
		},
	}
}

// CreateDefaultAuditRules 创建默认内容审查规则（仅在表为空时执行）。
func (s *AIAuditService) CreateDefaultAuditRules(createdBy uint) error {
	existing, err := s.repo.CountAuditRules()
	if err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	rules := DefaultAuditRules(createdBy)
	for i := range rules {
		if err := s.repo.CreateAuditRule(&rules[i]); err != nil {
			slog.Error("AI audit: failed to create default audit rule", "name", rules[i].Name, "err", err)
		}
	}
	return nil
}

// MaskAPIKey 对 API Key 进行脱敏处理（仅显示后4位）。
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	decrypted := key
	// 如果 key 看起来是加密的（base64），返回掩码
	if len(decrypted) > 8 {
		return "********" + decrypted[len(decrypted)-4:]
	}
	return "********"
}

// EnsureAIAuditServiceCompatible 确保 crypto 包被引用（防止未使用导入）。
var _ = crypto.DecryptOrPlaintext

// Ensure json 包被引用。
var _ = json.Marshal