package audit

import (
	"errors"
	"strconv"
	"strings"
)

// InaccessibleHTTPStatusCodes HTTP 层视为不可访问的状态码（不含 404）。
var InaccessibleHTTPStatusCodes = map[int]bool{
	403: true,
	500: true,
	502: true,
	503: true,
	504: true,
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
		return InaccessibleHTTPStatusCodes[fr.StatusCode]
	default:
		return false
	}
}

// IsSiteUnreachable 双协议均不可访问时返回 true。
func IsSiteUnreachable(dual DualFetchResult) bool {
	return IsChannelInaccessible(dual.HTTPS.FetchResult) && IsChannelInaccessible(dual.HTTP.FetchResult)
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
