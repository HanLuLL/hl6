package audit

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const NotifyTemplateMaxRunes = 1024

// NotifyTemplateVars 审核通知模板变量。
type NotifyTemplateVars struct {
	FQDN           string
	RuleName       string
	MatchedSnippet string
	Action         string
	RecheckMinutes int
	RecheckAt      time.Time
}

var notifyTemplateKeys = []string{
	"fqdn",
	"rule_name",
	"matched_snippet",
	"action",
	"recheck_minutes",
	"recheck_at",
}

// RenderNotifyTemplate 将 {{var}} 占位符替换为变量值；未知变量置空。
func RenderNotifyTemplate(tmpl string, vars NotifyTemplateVars) string {
	if tmpl == "" {
		return ""
	}
	values := map[string]string{
		"fqdn":            vars.FQDN,
		"rule_name":       vars.RuleName,
		"matched_snippet": vars.MatchedSnippet,
		"action":          vars.Action,
	}
	if vars.RecheckMinutes > 0 {
		values["recheck_minutes"] = strconv.Itoa(vars.RecheckMinutes)
	} else {
		values["recheck_minutes"] = ""
	}
	if !vars.RecheckAt.IsZero() {
		values["recheck_at"] = vars.RecheckAt.UTC().Format(time.RFC3339)
	} else {
		values["recheck_at"] = ""
	}

	out := tmpl
	for _, key := range notifyTemplateKeys {
		out = strings.ReplaceAll(out, "{{"+key+"}}", values[key])
	}
	return out
}

// ValidateNotifyTemplateContent 校验渲染后纯文本长度。
func ValidateNotifyTemplateContent(content string) bool {
	return utf8.RuneCountInString(content) <= NotifyTemplateMaxRunes
}
