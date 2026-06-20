package audit

import (
	"errors"
	"strconv"
	"strings"
)

// IsHTTPStatusInaccessible 判定 HTTP 响应状态码是否视为不可访问。
// 5xx 一律不可访问；4xx 默认不可访问，401/404/429 除外。
func IsHTTPStatusInaccessible(code int) bool {
	if code >= 500 {
		return true
	}
	if code < 400 || code >= 500 {
		return false
	}
	switch code {
	case 401, 404, 429:
		return false
	default:
		return true
	}
}

// IsChannelInaccessible 判定单协议通道是否不可访问。
func IsChannelInaccessible(fr FetchResult) bool {
	switch fr.Status {
	case FetchStatusUnreachable:
		return true
	case FetchStatusError:
		if errors.Is(fr.Error, ErrPrivateIP) {
			return false
		}
		return true
	case FetchStatusClean:
		return IsHTTPStatusInaccessible(fr.StatusCode)
	default:
		return false
	}
}

// IsSiteUnreachable 双协议均不可访问时返回 true。
func IsSiteUnreachable(dual DualFetchResult) bool {
	return IsChannelInaccessible(dual.HTTPS.FetchResult) && IsChannelInaccessible(dual.HTTP.FetchResult)
}

// ConfirmedUnreachable 两次双通道探测均不可访问时返回 true。
// 任一次可访问则不算不可达；仅完成一次且不可达时返回 false（尚未确认）。
func ConfirmedUnreachable(first DualFetchResult, second *DualFetchResult) bool {
	if !IsSiteUnreachable(first) {
		return false
	}
	if second == nil {
		return false
	}
	return IsSiteUnreachable(*second)
}

// HasPrivateIPError 任一通道因 SSRF 拦截失败。
func HasPrivateIPError(dual DualFetchResult) bool {
	return errors.Is(dual.HTTPS.Error, ErrPrivateIP) || errors.Is(dual.HTTP.Error, ErrPrivateIP)
}

// UnreachableChannelSummary 生成无法访问证据摘要。
func UnreachableChannelSummary(dual DualFetchResult) string {
	parts := make([]string, 0, 2)
	if ch := channelFailureLabel("https", dual.HTTPS); ch != "" {
		parts = append(parts, ch)
	}
	if ch := channelFailureLabel("http", dual.HTTP); ch != "" {
		parts = append(parts, ch)
	}
	return strings.Join(parts, "; ")
}

func channelFailureLabel(scheme string, ch ChannelResult) string {
	fr := ch.FetchResult
	if !IsChannelInaccessible(fr) {
		return ""
	}
	switch fr.Status {
	case FetchStatusClean:
		return scheme + ": HTTP " + strconv.Itoa(fr.StatusCode)
	case FetchStatusUnreachable, FetchStatusError:
		msg := strings.TrimSpace(fr.ErrorMessage)
		if msg == "" {
			msg = fr.Status
		}
		if len(msg) > 80 {
			msg = msg[:80] + "…"
		}
		return scheme + ": " + msg
	default:
		return scheme + ": " + fr.Status
	}
}
