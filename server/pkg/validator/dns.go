package validator

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type ValidationError struct {
	Message string
	Key     string
	Params  map[string]string
}

func (e *ValidationError) Error() string { return e.Message }

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
		return &ValidationError{
			Message: "invalid subdomain name: must contain only lowercase letters, numbers, and hyphens",
			Key:     "error.invalidSubdomainName",
		}
	}
	if reservedNames[name] {
		return &ValidationError{
			Message: fmt.Sprintf("subdomain name '%s' is reserved", name),
			Key:     "error.reservedSubdomain",
			Params:  map[string]string{"name": name},
		}
	}
	return nil
}

func ValidateDNSRecord(recordType, content string) error {
	switch strings.ToUpper(recordType) {
	case "A":
		ip := net.ParseIP(content)
		if ip == nil || ip.To4() == nil {
			return &ValidationError{
				Message: fmt.Sprintf("invalid IPv4 address: %s", content),
				Key:     "error.invalidIPv4",
				Params:  map[string]string{"value": content},
			}
		}
	case "AAAA":
		ip := net.ParseIP(content)
		if ip == nil || ip.To4() != nil {
			return &ValidationError{
				Message: fmt.Sprintf("invalid IPv6 address: %s", content),
				Key:     "error.invalidIPv6",
				Params:  map[string]string{"value": content},
			}
		}
	case "CNAME":
		if !isValidHostname(content) {
			return &ValidationError{
				Message: fmt.Sprintf("invalid CNAME target: %s", content),
				Key:     "error.invalidCNAME",
				Params:  map[string]string{"value": content},
			}
		}
	case "TXT":
		if strings.TrimSpace(content) == "" {
			return &ValidationError{
				Message: "TXT record content cannot be empty",
				Key:     "error.invalidTXT",
			}
		}
		if len(content) > 2048 {
			return &ValidationError{
				Message: fmt.Sprintf("TXT record content too long: %d characters (max 2048)", len(content)),
				Key:     "error.txtTooLong",
			}
		}
	default:
		return &ValidationError{
			Message: fmt.Sprintf("unsupported record type: %s", recordType),
			Key:     "error.unsupportedRecordType",
			Params:  map[string]string{"type": recordType},
		}
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
