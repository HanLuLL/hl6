package config

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizePublicURL validates and normalizes public base URL values.
// Empty input is allowed and returns empty output.
func NormalizePublicURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("must be an absolute URL with scheme and host")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("user info is not allowed")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("query and fragment are not allowed")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("path is not allowed")
	}

	return fmt.Sprintf("%s://%s", scheme, strings.ToLower(parsed.Host)), nil
}

// ParsePublicURLList parses a comma/newline/semicolon separated URL list.
// Empty input is allowed and returns nil.
func ParsePublicURLList(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	fields := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';'
	})

	result := make([]string, 0, len(fields))
	seen := make(map[string]bool, len(fields))
	for _, field := range fields {
		urlText := strings.TrimSpace(field)
		if urlText == "" {
			continue
		}
		normalized, err := NormalizePublicURL(urlText)
		if err != nil {
			return nil, err
		}
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}
	return result, nil
}
