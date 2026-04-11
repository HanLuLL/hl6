package service

import (
	"fmt"
	"strings"
)

func normalizeFQDN(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(name), ".")
}

func ensureFQDN(name string) string {
	trimmed := normalizeFQDN(name)
	if trimmed == "" {
		return ""
	}
	return trimmed + "."
}

func relativeRecordName(fqdn, zoneName string) (string, error) {
	record := normalizeFQDN(fqdn)
	zone := normalizeFQDN(zoneName)
	if record == "" || zone == "" {
		return "", fmt.Errorf("invalid fqdn(%q) or zone(%q)", fqdn, zoneName)
	}

	if strings.EqualFold(record, zone) {
		return "@", nil
	}

	lowerRecord := strings.ToLower(record)
	lowerZone := strings.ToLower(zone)
	suffix := "." + lowerZone
	if !strings.HasSuffix(lowerRecord, suffix) {
		return "", fmt.Errorf("record %q does not belong to zone %q", fqdn, zoneName)
	}

	relative := strings.TrimSuffix(record[:len(record)-len(suffix)], ".")
	if relative == "" {
		return "@", nil
	}
	return relative, nil
}

func containsStringFold(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
