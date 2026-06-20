package service

import (
	"regexp"
	"strconv"
	"strings"

	"hl6-server/internal/apperr"
	"hl6-server/internal/model"
)

const (
	auditKeywordMaxCount      = 50
	auditKeywordMaxLen        = 128
	auditRegexPatternMaxLen   = 512
)

// AuditRuleValidationError 携带可下发给客户端的 i18n message key。
type AuditRuleValidationError struct {
	Key string
}

func (e *AuditRuleValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Key
}

var auditAllowedTargets = map[string]bool{
	model.AuditTargetBody:       true,
	model.AuditTargetTitle:      true,
	model.AuditTargetFinalURL:   true,
	model.AuditTargetStatusCode: true,
}

// ValidateAuditRule 校验审核规则字段与组合合法性。
func ValidateAuditRule(rule *model.AuditRule) error {
	if rule == nil {
		return &AuditRuleValidationError{Key: apperr.KeyInvalidRequestBody}
	}
	if strings.TrimSpace(rule.Name) == "" {
		return &AuditRuleValidationError{Key: apperr.KeyInvalidRequestBody}
	}

	if len(rule.Targets) == 0 {
		return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleTargets}
	}
	for _, t := range rule.Targets {
		if !auditAllowedTargets[t] {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleTargets}
		}
	}

	switch rule.MatchType {
	case model.AuditMatchKeyword, model.AuditMatchRegex, model.AuditMatchStatusEq:
	default:
		return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleMatchType}
	}

	hasStatusCode := false
	for _, t := range rule.Targets {
		if t == model.AuditTargetStatusCode {
			hasStatusCode = true
			break
		}
	}

	switch rule.MatchType {
	case model.AuditMatchStatusEq:
		if !hasStatusCode || len(rule.Targets) != 1 {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleCombination}
		}
		pattern := strings.TrimSpace(rule.Pattern)
		code, err := strconv.Atoi(pattern)
		if err != nil || code < 100 || code > 599 {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRulePattern}
		}
	case model.AuditMatchKeyword:
		if hasStatusCode {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleCombination}
		}
		if len(rule.Keywords) == 0 {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleKeywords}
		}
		if len(rule.Keywords) > auditKeywordMaxCount {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleKeywords}
		}
		for _, kw := range rule.Keywords {
			kw = strings.TrimSpace(kw)
			if kw == "" || len(kw) > auditKeywordMaxLen {
				return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleKeywords}
			}
		}
		if rule.KeywordLogic != model.AuditKeywordLogicAny && rule.KeywordLogic != model.AuditKeywordLogicAll {
			rule.KeywordLogic = model.AuditKeywordLogicAny
		}
	case model.AuditMatchRegex:
		if hasStatusCode {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleCombination}
		}
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" || len(pattern) > auditRegexPatternMaxLen {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRulePattern}
		}
		flag := ""
		if !rule.CaseSensitive {
			flag = "(?i)"
		}
		if _, err := regexp.Compile(flag + pattern); err != nil {
			return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRulePattern}
		}
	}

	switch rule.Action {
	case model.AuditActionObserve, model.AuditActionSite, model.AuditActionUser:
	default:
		return &AuditRuleValidationError{Key: apperr.KeyInvalidAuditRuleAction}
	}

	if rule.ScopeDomainIDs == nil {
		rule.ScopeDomainIDs = model.UintSlice{}
	}
	if rule.Keywords == nil {
		rule.Keywords = model.StringSlice{}
	}
	if rule.Targets == nil {
		rule.Targets = model.StringSlice{}
	}

	return nil
}

// ValidateAuditRuleScope 校验 scope_domain_ids 中的根域名均存在。
func ValidateAuditRuleScope(rule *model.AuditRule, domainExists func(uint) bool) error {
	if err := ValidateAuditRule(rule); err != nil {
		return err
	}
	if len(rule.ScopeDomainIDs) == 0 || domainExists == nil {
		return nil
	}
	for _, id := range rule.ScopeDomainIDs {
		if !domainExists(id) {
			return &AuditRuleValidationError{Key: apperr.KeyDomainNotFoundOrInactive}
		}
	}
	return nil
}
