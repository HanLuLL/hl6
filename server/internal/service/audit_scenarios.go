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
		ID:           "keyword",
		NameKey:      "audit.scenarios.keyword.name",
		DescKey:      "audit.scenarios.keyword.desc",
		Targets:      []string{model.AuditTargetBody, model.AuditTargetTitle},
		MatchType:    model.AuditMatchKeyword,
		KeywordLogic: model.AuditKeywordLogicAny,
		Keywords:     []string{"demo", "test", "example", "示例", "演示"},
	},
	{
		ID:        "regex",
		NameKey:   "audit.scenarios.regex.name",
		DescKey:   "audit.scenarios.regex.desc",
		Targets:   []string{model.AuditTargetBody, model.AuditTargetTitle},
		MatchType: model.AuditMatchRegex,
		Pattern:   `(?i)hello\s+world|lorem\s+ipsum`,
	},
	{
		ID:        "status_eq",
		NameKey:   "audit.scenarios.status_eq.name",
		DescKey:   "audit.scenarios.status_eq.desc",
		Targets:   []string{model.AuditTargetStatusCode},
		MatchType: model.AuditMatchStatusEq,
		Pattern:   "502",
	},
	{
		ID:        "unreachable",
		NameKey:   "audit.scenarios.unreachable.name",
		DescKey:   "audit.scenarios.unreachable.desc",
		Targets:   []string{},
		MatchType: model.AuditMatchUnreachable,
	},
}

// ListAuditScenarios 返回内置场景模板列表。
func ListAuditScenarios() []AuditScenario {
	out := make([]AuditScenario, len(builtinAuditScenarios))
	copy(out, builtinAuditScenarios)
	return out
}
