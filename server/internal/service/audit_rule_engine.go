package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"hl6-server/internal/helpers"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/pkg/audit"
)

const auditMatchTargetMaxBytes = 2 * 1024 * 1024

// MatchedRule 记录单条规则命中结果。
type MatchedRule struct {
	Rule    *model.AuditRule
	Snippet string
}

// AuditRuleEngine 基于管理员配置的规则，对抓取到的网页内容进行匹配。
type AuditRuleEngine struct {
	repo  *repository.Repository
	regex sync.Map // key: "id:updatedAtUnix:pattern" -> *regexp.Regexp
}

func NewAuditRuleEngine(repo *repository.Repository) *AuditRuleEngine {
	return &AuditRuleEngine{repo: repo}
}

// MatchAll 加载所有已启用规则，对双次探测结果返回全部命中。
func (e *AuditRuleEngine) MatchAll(ctx context.Context, domainID uint, probe audit.DualFetchProbeResult) ([]MatchedRule, error) {
	rules, err := e.repo.ListEnabledAuditRules()
	if err != nil {
		return nil, err
	}
	return e.MatchAllProbeWithRules(domainID, probe, rules), nil
}

// MatchAllWithRules 对给定规则列表执行单通道匹配（兼容旧调用）。
func (e *AuditRuleEngine) MatchAllWithRules(domainID uint, fr audit.FetchResult, rules []model.AuditRule) []MatchedRule {
	dual := audit.DualFetchResult{
		HTTPS: audit.ChannelResult{Scheme: "https", FetchResult: fr},
	}
	return e.MatchAllDualWithRules(domainID, dual, rules)
}

// MatchAllDualWithRules 双通道规则匹配（兼容旧调用，不做双次不可达确认）。
func (e *AuditRuleEngine) MatchAllDualWithRules(domainID uint, dual audit.DualFetchResult, rules []model.AuditRule) []MatchedRule {
	return e.MatchAllProbeWithRules(domainID, audit.DualFetchProbeResult{Primary: dual, First: dual}, rules)
}

// MatchAllProbeWithRules 双次探测结果规则匹配（试跑草稿规则用）。
func (e *AuditRuleEngine) MatchAllProbeWithRules(domainID uint, probe audit.DualFetchProbeResult, rules []model.AuditRule) []MatchedRule {
	if probe.SkipRuleMatch {
		return nil
	}
	dual := probe.Primary
	var matches []MatchedRule
	for i := range rules {
		rule := &rules[i]
		if !rule.Enabled && rule.ID != 0 {
			continue
		}
		if !ruleInScope(rule, domainID) {
			continue
		}
		if rule.MatchType == model.AuditMatchUnreachable {
			if probe.ConfirmedUnreachable() {
				summaryDual := probe.First
				if probe.Second != nil {
					summaryDual = *probe.Second
				}
				matches = append(matches, MatchedRule{
					Rule:    rule,
					Snippet: helpers.TruncateRunes(audit.UnreachableChannelSummary(summaryDual), 200),
				})
			}
			continue
		}
		for _, ch := range []audit.ChannelResult{dual.HTTPS, dual.HTTP} {
			if ch.Scheme == "" {
				continue
			}
			snippet, matched := e.matchRule(rule, ch.FetchResult)
			if !matched {
				continue
			}
			prefixed := fmt.Sprintf("[%s] %s", ch.Scheme, snippet)
			matches = append(matches, MatchedRule{
				Rule:    rule,
				Snippet: helpers.TruncateRunes(prefixed, 200),
			})
		}
	}
	return matches
}

// PickPrimaryMatch 从命中列表取主命中（action 最高；同档取 created_at 最早）。
func PickPrimaryMatch(matches []MatchedRule) *MatchedRule {
	if len(matches) == 0 {
		return nil
	}
	best := &matches[0]
	for i := 1; i < len(matches); i++ {
		candidate := &matches[i]
		if betterAuditMatch(candidate, best) {
			best = candidate
		}
	}
	return best
}

func ruleInScope(rule *model.AuditRule, domainID uint) bool {
	if len(rule.ScopeDomainIDs) == 0 {
		return true
	}
	for _, id := range rule.ScopeDomainIDs {
		if id == domainID {
			return true
		}
	}
	return false
}

func auditRuleActionRank(action string) int {
	switch action {
	case model.AuditActionUser:
		return 4
	case model.AuditActionSite:
		return 3
	case model.AuditActionDeleteDNS:
		return 2
	case model.AuditActionObserve:
		return 1
	default:
		return 0
	}
}

func betterAuditMatch(candidate, current *MatchedRule) bool {
	cRank := auditRuleActionRank(candidate.Rule.Action)
	curRank := auditRuleActionRank(current.Rule.Action)
	if cRank != curRank {
		return cRank > curRank
	}
	return candidate.Rule.CreatedAt.Before(current.Rule.CreatedAt)
}

func (e *AuditRuleEngine) matchRule(rule *model.AuditRule, fr audit.FetchResult) (string, bool) {
	switch rule.MatchType {
	case model.AuditMatchStatusEq:
		if fr.Status != audit.FetchStatusClean {
			return "", false
		}
		return e.matchStatusEq(rule, fr.StatusCode)
	case model.AuditMatchKeyword:
		return e.matchKeywordMultiTarget(rule, fr)
	case model.AuditMatchRegex:
		return e.matchRegexMultiTarget(rule, fr)
	}
	return "", false
}

func (e *AuditRuleEngine) matchKeywordMultiTarget(rule *model.AuditRule, fr audit.FetchResult) (string, bool) {
	keywords := make([]string, 0, len(rule.Keywords))
	for _, kw := range rule.Keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			keywords = append(keywords, kw)
		}
	}
	if len(keywords) == 0 {
		return "", false
	}

	if rule.KeywordLogic == model.AuditKeywordLogicAll {
		return e.matchKeywordAll(rule, fr, keywords)
	}
	return e.matchKeywordAny(rule, fr, keywords)
}

func (e *AuditRuleEngine) matchKeywordAny(rule *model.AuditRule, fr audit.FetchResult, keywords []string) (string, bool) {
	for _, targetName := range rule.Targets {
		target := e.getTarget(targetName, fr)
		if target == "" {
			continue
		}
		if len(target) > auditMatchTargetMaxBytes {
			target = target[:auditMatchTargetMaxBytes]
		}
		search := target
		if !rule.CaseSensitive {
			search = strings.ToLower(search)
		}
		for _, kw := range keywords {
			needle := kw
			if !rule.CaseSensitive {
				needle = strings.ToLower(kw)
			}
			if idx := strings.Index(search, needle); idx >= 0 {
				return helpers.SnippetAroundByteIndex(target, idx, len(needle), 50, 100), true
			}
		}
	}
	return "", false
}

func (e *AuditRuleEngine) matchKeywordAll(rule *model.AuditRule, fr audit.FetchResult, keywords []string) (string, bool) {
	type targetText struct {
		raw     string
		search  string
	}
	targets := make([]targetText, 0, len(rule.Targets))
	for _, targetName := range rule.Targets {
		raw := e.getTarget(targetName, fr)
		if raw == "" {
			continue
		}
		if len(raw) > auditMatchTargetMaxBytes {
			raw = raw[:auditMatchTargetMaxBytes]
		}
		search := raw
		if !rule.CaseSensitive {
			search = strings.ToLower(search)
		}
		targets = append(targets, targetText{raw: raw, search: search})
	}
	if len(targets) == 0 {
		return "", false
	}

	var firstSnippet string
	for _, kw := range keywords {
		needle := kw
		if !rule.CaseSensitive {
			needle = strings.ToLower(kw)
		}
		found := false
		for _, t := range targets {
			if idx := strings.Index(t.search, needle); idx >= 0 {
				found = true
				if firstSnippet == "" {
					firstSnippet = helpers.SnippetAroundByteIndex(t.raw, idx, len(needle), 50, 100)
				}
				break
			}
		}
		if !found {
			return "", false
		}
	}
	return firstSnippet, true
}

func (e *AuditRuleEngine) matchRegexMultiTarget(rule *model.AuditRule, fr audit.FetchResult) (string, bool) {
	re, err := e.compiledRegex(rule)
	if err != nil || re == nil {
		return "", false
	}
	for _, targetName := range rule.Targets {
		target := e.getTarget(targetName, fr)
		if target == "" {
			continue
		}
		if len(target) > auditMatchTargetMaxBytes {
			target = target[:auditMatchTargetMaxBytes]
		}
		loc := re.FindStringIndex(target)
		if loc != nil {
			return helpers.SnippetAroundByteIndex(target, loc[0], loc[1]-loc[0], 50, 100), true
		}
	}
	return "", false
}

func (e *AuditRuleEngine) getTarget(target string, fr audit.FetchResult) string {
	switch target {
	case model.AuditTargetBody:
		return fr.Body
	case model.AuditTargetTitle:
		return fr.Title
	case model.AuditTargetFinalURL:
		return fr.FinalURL
	default:
		return ""
	}
}

func (e *AuditRuleEngine) compiledRegex(rule *model.AuditRule) (*regexp.Regexp, error) {
	cacheKey := fmt.Sprintf("%d:%d:%s", rule.ID, rule.UpdatedAt.Unix(), rule.Pattern)
	if rule.ID == 0 {
		cacheKey = fmt.Sprintf("draft:%s:%v", rule.Pattern, rule.CaseSensitive)
	}
	if v, ok := e.regex.Load(cacheKey); ok {
		return v.(*regexp.Regexp), nil
	}
	flag := ""
	if !rule.CaseSensitive {
		flag = "(?i)"
	}
	re, err := regexp.Compile(flag + rule.Pattern)
	if err != nil {
		return nil, err
	}
	e.regex.Store(cacheKey, re)
	return re, nil
}

func (e *AuditRuleEngine) matchStatusEq(rule *model.AuditRule, statusCode int) (string, bool) {
	expected, err := strconv.Atoi(strings.TrimSpace(rule.Pattern))
	if err != nil {
		return "", false
	}
	if statusCode == expected {
		return strconv.Itoa(statusCode), true
	}
	return "", false
}

// ToMatchedRuleHits 将引擎命中转为存储/响应结构。
func ToMatchedRuleHits(matches []MatchedRule) model.MatchedRulesSlice {
	out := make(model.MatchedRulesSlice, 0, len(matches))
	for _, m := range matches {
		out = append(out, model.MatchedRuleHit{
			RuleID:   m.Rule.ID,
			RuleName: m.Rule.Name,
			Action:   m.Rule.Action,
			Snippet:  m.Snippet,
		})
	}
	return out
}
