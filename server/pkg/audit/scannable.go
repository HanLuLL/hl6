package audit

import "hl6-server/internal/model"

// ScannableRecordTypes 内容审查可抓取 HTTP 内容的 DNS 记录类型。
var ScannableRecordTypes = []string{"A", "AAAA", "CNAME"}

var scannableRecordTypeSet = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true,
}

// IsScannableRecordType 判断 DNS 记录类型是否可扫描。
func IsScannableRecordType(t string) bool {
	return scannableRecordTypeSet[t]
}

// SubdomainHasScannableActiveDNS 判断子域是否存在至少一条可扫描且 active 的 DNS 记录。
func SubdomainHasScannableActiveDNS(records []model.DNSRecord) bool {
	for _, rec := range records {
		if rec.Status == model.DNSRecordStatusActive && IsScannableRecordType(rec.Type) {
			return true
		}
	}
	return false
}
