package validator

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var subdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

var reservedNames = map[string]bool{
	"www": true, "mail": true, "ftp": true, "ns1": true, "ns2": true,
	"smtp": true, "pop": true, "imap": true, "admin": true, "api": true,
	"mx": true, "dns": true, "ns": true, "cpanel": true, "webmail": true,
	"localhost": true, "autoconfig": true, "autodiscover": true,
}

func ValidateSubdomainName(name string) error {
	name = strings.ToLower(name)
	if !subdomainRegex.MatchString(name) {
		return fmt.Errorf("invalid subdomain name: must contain only lowercase letters, numbers, and hyphens")
	}
	if reservedNames[name] {
		return fmt.Errorf("subdomain name '%s' is reserved", name)
	}
	return nil
}

func ValidateDNSRecord(recordType, content string) error {
	switch strings.ToUpper(recordType) {
	case "A":
		ip := net.ParseIP(content)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid IPv4 address: %s", content)
		}
	case "AAAA":
		ip := net.ParseIP(content)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("invalid IPv6 address: %s", content)
		}
	case "CNAME":
		if !isValidHostname(content) {
			return fmt.Errorf("invalid CNAME target: %s", content)
		}
	default:
		return fmt.Errorf("unsupported record type: %s", recordType)
	}
	return nil
}

var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}\.?$`)

func isValidHostname(host string) bool {
	if len(host) > 253 {
		return false
	}
	return hostnameRegex.MatchString(host)
}
