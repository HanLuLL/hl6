package auth

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

const (
	DomainPolicyUnrestricted = "unrestricted"
	DomainPolicyAllowlist    = "allowlist"
	DomainPolicyBlocklist    = "blocklist"
)

var (
	ErrInvalidEmail        = errors.New("invalid email address")
	ErrInvalidDomainPolicy = errors.New("invalid email domain policy")
	ErrRegistrationDenied  = errors.New("email domain is not allowed to register")
)

type DomainPolicy struct {
	Mode    string
	Domains []string
}

// NormalizeEmail creates the case-insensitive identity key used for sign-in.
// The original user-entered value remains available separately for display.
func NormalizeEmail(raw string) (string, error) {
	email := strings.TrimSpace(raw)
	if email == "" || len(email) > 320 || !utf8.ValidString(email) || strings.Count(email, "@") != 1 {
		return "", ErrInvalidEmail
	}

	local, domain, _ := strings.Cut(email, "@")
	if !validLocalPart(local) {
		return "", ErrInvalidEmail
	}

	normalizedDomain, err := normalizeDomain(domain)
	if err != nil {
		return "", ErrInvalidEmail
	}

	return strings.ToLower(local) + "@" + normalizedDomain, nil
}

func ValidateRegistrationDomain(email string, policy DomainPolicy) error {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return err
	}

	_, domain, _ := strings.Cut(normalizedEmail, "@")
	mode := strings.TrimSpace(policy.Mode)
	if mode == "" {
		mode = DomainPolicyUnrestricted
	}

	normalizedDomains := make(map[string]struct{}, len(policy.Domains))
	for _, configuredDomain := range policy.Domains {
		normalizedDomain, err := normalizeDomain(configuredDomain)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidDomainPolicy, err)
		}
		normalizedDomains[normalizedDomain] = struct{}{}
	}

	_, matched := normalizedDomains[domain]
	switch mode {
	case DomainPolicyUnrestricted:
		return nil
	case DomainPolicyAllowlist:
		if matched {
			return nil
		}
		return ErrRegistrationDenied
	case DomainPolicyBlocklist:
		if matched {
			return ErrRegistrationDenied
		}
		return nil
	default:
		return ErrInvalidDomainPolicy
	}
}

// QQAvatarURL returns the official QQ image endpoint only for numeric qq.com
// mailboxes. HTTPS is required to avoid browser mixed-content blocking.
func QQAvatarURL(email string) string {
	normalized, err := NormalizeEmail(email)
	if err != nil {
		return ""
	}

	local, domain, _ := strings.Cut(normalized, "@")
	if domain != "qq.com" || len(local) < 5 || len(local) > 12 {
		return ""
	}
	for _, char := range local {
		if char < '0' || char > '9' {
			return ""
		}
	}

	return "https://q.qlogo.cn/headimg_dl?dst_uin=" + url.QueryEscape(local) + "&spec=640&img_type=jpg"
}

func validLocalPart(local string) bool {
	if local == "" || len(local) > 64 || strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return false
	}
	for _, char := range local {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			continue
		}
		if strings.ContainsRune(".!#$%&'*+-/=?^_`{|}~", char) {
			continue
		}
		return false
	}
	return true
}

func normalizeDomain(raw string) (string, error) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.HasSuffix(raw, ".") {
		return "", ErrInvalidEmail
	}

	domain := raw
	if domain == "" || len(domain) > 253 || strings.ContainsAny(domain, "@/\\") || strings.Contains(domain, "*") {
		return "", ErrInvalidEmail
	}

	asciiDomain, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return "", err
	}
	asciiDomain = strings.ToLower(asciiDomain)
	if !strings.Contains(asciiDomain, ".") {
		return "", ErrInvalidEmail
	}

	for _, label := range strings.Split(asciiDomain, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", ErrInvalidEmail
		}
		for _, char := range label {
			if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
				continue
			}
			return "", ErrInvalidEmail
		}
	}

	return asciiDomain, nil
}
