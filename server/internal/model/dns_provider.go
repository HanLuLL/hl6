package model

import "strings"

const (
	DNSProviderCloudflare = "cloudflare"
	DNSProviderDNSPod     = "dnspod"
	DNSProviderAliDNS     = "aliyun_dns"
	DNSProviderHuaweiDNS  = "huawei_cloud_dns"
)

var supportedDNSProviders = map[string]struct{}{
	DNSProviderCloudflare: {},
	DNSProviderDNSPod:     {},
	DNSProviderAliDNS:     {},
	DNSProviderHuaweiDNS:  {},
}

func NormalizeProvider(provider string) string {
	return strings.TrimSpace(strings.ToLower(provider))
}

func IsSupportedProvider(provider string) bool {
	_, ok := supportedDNSProviders[NormalizeProvider(provider)]
	return ok
}
