package service

import "hl6-server/internal/model"

// AuditScenario 内置场景模板（只预填检测条件，不预填 action）。
type AuditScenario struct {
	ID           string   `json:"id"`
	NameKey      string   `json:"name_key"`
	DescKey      string   `json:"desc_key"`
	Targets      []string `json:"targets"`
	MatchType    string   `json:"match_type"`
	Keywords     []string `json:"keywords,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	KeywordLogic string   `json:"keyword_logic,omitempty"`
}

var builtinAuditScenarios = []AuditScenario{
	{
		ID:           "gambling",
		NameKey:      "audit.scenarios.gambling.name",
		DescKey:      "audit.scenarios.gambling.desc",
		Targets:      []string{model.AuditTargetBody, model.AuditTargetTitle},
		MatchType:    model.AuditMatchKeyword,
		KeywordLogic: model.AuditKeywordLogicAny,
		Keywords:     []string{"casino", "博彩", "赌博", "bet365", "slot machine", "poker"},
	},
	{
		ID:        "adult",
		NameKey:   "audit.scenarios.adult.name",
		DescKey:   "audit.scenarios.adult.desc",
		Targets:   []string{model.AuditTargetBody, model.AuditTargetTitle},
		MatchType: model.AuditMatchRegex,
		Pattern:   `(?i)(porn|xxx|adult\s+content|色情|成人视频)`,
	},
	{
		ID:           "phishing",
		NameKey:      "audit.scenarios.phishing.name",
		DescKey:      "audit.scenarios.phishing.desc",
		Targets:      []string{model.AuditTargetTitle, model.AuditTargetFinalURL},
		MatchType:    model.AuditMatchKeyword,
		KeywordLogic: model.AuditKeywordLogicAny,
		Keywords:     []string{"verify account", "login", "验证码", "account suspended", "security alert"},
	},
	{
		ID:        "malicious_redirect",
		NameKey:   "audit.scenarios.malicious_redirect.name",
		DescKey:   "audit.scenarios.malicious_redirect.desc",
		Targets:   []string{model.AuditTargetFinalURL},
		MatchType: model.AuditMatchRegex,
		Pattern:   `(?i)https?://(?!.*\bexample\.com\b)`,
	},
	{
		ID:           "placeholder",
		NameKey:      "audit.scenarios.placeholder.name",
		DescKey:      "audit.scenarios.placeholder.desc",
		Targets:      []string{model.AuditTargetBody},
		MatchType:    model.AuditMatchKeyword,
		KeywordLogic: model.AuditKeywordLogicAny,
		Keywords:     []string{"Under Construction", "默认页", "nginx welcome", "It works!", "Coming Soon"},
	},
	{
		ID:        "http_error",
		NameKey:   "audit.scenarios.http_error.name",
		DescKey:   "audit.scenarios.http_error.desc",
		Targets:   []string{model.AuditTargetStatusCode},
		MatchType: model.AuditMatchStatusEq,
		Pattern:   "502",
	},
}

// ListAuditScenarios 返回内置场景模板列表。
func ListAuditScenarios() []AuditScenario {
	out := make([]AuditScenario, len(builtinAuditScenarios))
	copy(out, builtinAuditScenarios)
	return out
}
