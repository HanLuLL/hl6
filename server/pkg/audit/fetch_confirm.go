package audit

import "context"

// DualFetchProbeResult 双次探测结果：首次不可达时会自动重试一次。
type DualFetchProbeResult struct {
	// Primary 用于扫描记录与规则匹配的主结果（优先采用可访问的那次）。
	Primary DualFetchResult
	First   DualFetchResult
	Second  *DualFetchResult
	// SkipRuleMatch 为 true 时表示首次不可达但二次可访问，不触发任何规则匹配。
	SkipRuleMatch bool
}

// FetchDualConfirmed 执行双次可达性探测。
// 首次可访问时直接返回；首次不可达（非内网 IP 拦截）时再探测一次。
func (f *SafeFetcher) FetchDualConfirmed(ctx context.Context, fqdn string) DualFetchProbeResult {
	first := f.FetchDual(ctx, fqdn)
	if !IsSiteUnreachable(first) || HasPrivateIPError(first) {
		return DualFetchProbeResult{Primary: first, First: first}
	}

	second := f.FetchDual(ctx, fqdn)
	if !IsSiteUnreachable(second) {
		return DualFetchProbeResult{
			Primary:       second,
			First:         first,
			Second:        &second,
			SkipRuleMatch: true,
		}
	}
	return DualFetchProbeResult{Primary: first, First: first, Second: &second}
}

// ConfirmedUnreachable 两次探测均不可达时返回 true。
func (p DualFetchProbeResult) ConfirmedUnreachable() bool {
	return ConfirmedUnreachable(p.First, p.Second)
}
