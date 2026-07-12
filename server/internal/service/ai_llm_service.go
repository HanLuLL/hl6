package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"hl6-server/internal/model"
	"hl6-server/pkg/crypto"
)

// ---- OpenAI API 兼容的 LLM 调用服务 ----

// LLMChatRequest OpenAI Chat Completion 请求格式。
type LLMChatRequest struct {
	Model       string          `json:"model"`
	Messages    []LLMChatMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

// LLMChatMessage 对话消息。
type LLMChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMChatResponse OpenAI Chat Completion 响应格式。
type LLMChatResponse struct {
	ID      string        `json:"id"`
	Choices []LLMChoice   `json:"choices"`
	Usage   LLMUsage      `json:"usage"`
	Error   *LLMAPIError  `json:"error,omitempty"`
}

// LLMChoice 选项。
type LLMChoice struct {
	Index   int           `json:"index"`
	Message LLMChatMessage `json:"message"`
	FinishReason string    `json:"finish_reason"`
}

// LLMUsage Token 用量。
type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMAPIError API 错误。
type LLMAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// LLMSendResult LLM 调用结果。
type LLMSendResult struct {
	Response    string
	TokensUsed  int
	ModelUsed   string
	Duration    time.Duration
	Error       error
}

// AILLMService 封装与 AI LLM 的交互，支持 OpenAI API 兼容格式。
type AILLMService struct {
	encryptionKey []byte
	httpClient    *http.Client
	// 限流：per model config ID 的令牌桶
	rateLimiters sync.Map // map[uint]*rateLimiter
}

type rateLimiter struct {
	mu       sync.Mutex
	tokens   int
	maxRPM   int
	lastFill time.Time
}

// NewAILLMService 创建 LLM 服务实例。
func NewAILLMService(encryptionKey []byte) *AILLMService {
	return &AILLMService{
		encryptionKey: encryptionKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // AI 调用可能较慢
		},
	}
}

// SendChatRequest 向 LLM 发送 Chat Completion 请求。
// config: AI 模型配置（含加密的 API Key）
// systemPrompt: 系统提示词
// userPrompt: 用户消息
func (s *AILLMService) SendChatRequest(ctx context.Context, config *model.AIModelConfig, systemPrompt, userPrompt string) LLMSendResult {
	start := time.Now()

	// 限流检查
	if !s.tryAcquire(config.ID, config.RateLimitRPM) {
		return LLMSendResult{
			Error:    fmt.Errorf("rate limit exceeded for model config %d (%d RPM)", config.ID, config.RateLimitRPM),
			Duration: time.Since(start),
		}
	}

	// 解密 API Key
	apiKey := crypto.DecryptOrPlaintext(config.APIKey, s.encryptionKey)

	// 构建请求
	reqBody := LLMChatRequest{
		Model: config.ModelName,
		Messages: []LLMChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return LLMSendResult{Error: fmt.Errorf("marshal request: %w", err), Duration: time.Since(start)}
	}

	baseURL := strings.TrimRight(config.APIBaseURL, "/")
	url := baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return LLMSendResult{Error: fmt.Errorf("create request: %w", err), Duration: time.Since(start)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求（含重试）
	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = s.httpClient.Do(req)
		if err != nil {
			if attempt < 2 {
				slog.Warn("AI LLM request failed, retrying", "attempt", attempt+1, "err", err)
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return LLMSendResult{Error: fmt.Errorf("send request after retries: %w", err), Duration: time.Since(start)}
		}
		break
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LLMSendResult{Error: fmt.Errorf("read response: %w", err), Duration: time.Since(start)}
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("AI LLM API error", "status", resp.StatusCode, "body", string(body))
		return LLMSendResult{
			Error:    fmt.Errorf("API returned status %d: %s", resp.StatusCode, truncateString(string(body), 200)),
			Duration: time.Since(start),
		}
	}

	var chatResp LLMChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return LLMSendResult{Error: fmt.Errorf("unmarshal response: %w", err), Duration: time.Since(start)}
	}

	if chatResp.Error != nil {
		return LLMSendResult{
			Error:    fmt.Errorf("API error: %s", chatResp.Error.Message),
			Duration: time.Since(start),
		}
	}

	if len(chatResp.Choices) == 0 {
		return LLMSendResult{Error: fmt.Errorf("no choices in response"), Duration: time.Since(start)}
	}

	return LLMSendResult{
		Response:   chatResp.Choices[0].Message.Content,
		TokensUsed: chatResp.Usage.TotalTokens,
		ModelUsed:  chatResp.Model,
		Duration:   time.Since(start),
	}
}

// tryAcquire 令牌桶限流检查。
func (s *AILLMService) tryAcquire(configID uint, maxRPM int) bool {
	if maxRPM <= 0 {
		return true
	}
	val, _ := s.rateLimiters.LoadOrStore(configID, &rateLimiter{maxRPM: maxRPM, tokens: maxRPM, lastFill: time.Now()})
	rl := val.(*rateLimiter)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastFill)
	tokensToAdd := int(elapsed.Minutes()) * rl.maxRPM
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxRPM {
			rl.tokens = rl.maxRPM
		}
		rl.lastFill = now
	}

	if rl.tokens <= 0 {
		return false
	}
	rl.tokens--
	return true
}

// ParseAIJudgment 从 AI 响应中解析审查判定结果。
// AI 应返回 JSON 格式：{"judgment":"clean/violation","violation_types":[...],"confidence":0.95,"suggested_action":"observe/site/user","reason":"..."}
func ParseAIJudgment(response string) (judgment string, violationTypes []string, confidence float64, suggestedAction string, reason string) {
	// 尝试从响应中提取 JSON
	response = strings.TrimSpace(response)

	// 尝试直接解析
	var result struct {
		Judgment        string   `json:"judgment"`
		ViolationTypes  []string `json:"violation_types"`
		Confidence      float64  `json:"confidence"`
		SuggestedAction string   `json:"suggested_action"`
		Reason          string   `json:"reason"`
	}

	// 尝试提取 JSON 块（可能被 markdown 代码块包裹）
	jsonStr := extractJSONBlock(response)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		slog.Warn("AI audit: failed to parse AI response as JSON", "err", err, "response_preview", truncateString(response, 200))
		// 无法解析时默认标记为 error
		return model.AIJudgmentError, nil, 0, "", truncateString(response, 500)
	}

	judgment = strings.ToLower(strings.TrimSpace(result.Judgment))
	switch judgment {
	case "clean", "safe", "pass", "ok":
		judgment = model.AIJudgmentClean
	case "violation", "unsafe", "fail", "violated":
		judgment = model.AIJudgmentViolation
	default:
		judgment = model.AIJudgmentError
	}

	return judgment, result.ViolationTypes, result.Confidence, result.SuggestedAction, result.Reason
}

// extractJSONBlock 从可能包含 markdown 代码块的文本中提取 JSON。
func extractJSONBlock(s string) string {
	// 尝试提取 ```json ... ``` 块
	if idx := strings.Index(s, "```json"); idx >= 0 {
		start := idx + 7
		if end := strings.Index(s[start:], "```"); end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// 尝试提取 ``` ... ``` 块
	if idx := strings.Index(s, "```"); idx >= 0 {
		start := idx + 3
		if end := strings.Index(s[start:], "```"); end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// 尝试找到第一个 { 和最后一个 }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
