package model

import "strings"

const (
	DNSProviderCloudflare   = "cloudflare"
	DNSProviderDNSPod       = "dnspod"
	DNSProviderAliDNS       = "aliyun_dns"
	DNSProviderHuaweiDNS    = "huawei_cloud_dns"
	DNSProviderDNSCom       = "dns_com"
	DNSProviderDNSLA        = "dnsla"
	DNSProviderWestCN       = "westcn_dns"
	DNSProviderBaiduDNS     = "baidu_cloud_dns"
	DNSProviderAWSRoute53   = "aws_route53"
	DNSProviderGoogleDNS    = "google_cloud_dns"
)

var supportedDNSProviders = map[string]struct{}{
	DNSProviderCloudflare: {},
	DNSProviderDNSPod:     {},
	DNSProviderAliDNS:     {},
	DNSProviderHuaweiDNS:  {},
	DNSProviderDNSCom:     {},
	DNSProviderDNSLA:      {},
	DNSProviderWestCN:     {},
	DNSProviderBaiduDNS:   {},
	DNSProviderAWSRoute53: {},
	DNSProviderGoogleDNS:  {},
}

func NormalizeProvider(provider string) string {
	return strings.TrimSpace(strings.ToLower(provider))
}

func IsSupportedProvider(provider string) bool {
	_, ok := supportedDNSProviders[NormalizeProvider(provider)]
	return ok
}
