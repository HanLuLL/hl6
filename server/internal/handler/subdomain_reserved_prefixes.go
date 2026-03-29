package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"hl6-server/internal/repository"
)

const configKeyReservedSubdomainPrefixes = "reserved_subdomain_prefixes"

var reservedSubdomainPrefixPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func loadReservedSubdomainPrefixes(repo *repository.Repository) ([]string, error) {
	raw, err := repo.GetSystemConfig(configKeyReservedSubdomainPrefixes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []string{}, nil
		}
		return nil, err
	}
	return parseReservedSubdomainPrefixes(raw), nil
}

func saveReservedSubdomainPrefixes(repo *repository.Repository, prefixes []string) error {
	encoded, err := json.Marshal(prefixes)
	if err != nil {
		return err
	}
	return repo.SetSystemConfig(configKeyReservedSubdomainPrefixes, string(encoded))
}

func parseReservedSubdomainPrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	var prefixes []string
	if err := json.Unmarshal([]byte(raw), &prefixes); err == nil {
		normalized, normalizeErr := normalizeReservedSubdomainPrefixes(prefixes)
		if normalizeErr == nil {
			return normalized
		}
		return []string{}
	}

	lines := strings.Split(raw, "\n")
	normalized, err := normalizeReservedSubdomainPrefixes(lines)
	if err != nil {
		return []string{}
	}
	return normalized
}

func normalizeReservedSubdomainPrefixes(prefixes []string) ([]string, error) {
	seen := make(map[string]struct{}, len(prefixes))
	normalized := make([]string, 0, len(prefixes))

	for _, prefix := range prefixes {
		value := strings.ToLower(strings.TrimSpace(prefix))
		if value == "" {
			continue
		}
		if strings.Contains(value, ".") {
			return nil, fmt.Errorf("reserved prefix %q cannot contain dots", value)
		}
		if value != "@" && value != "*" && !reservedSubdomainPrefixPattern.MatchString(value) {
			return nil, fmt.Errorf("invalid reserved prefix %q", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized, nil
}

func isReservedSubdomainPrefix(name string, reservedPrefixes []string) bool {
	value := strings.ToLower(strings.TrimSpace(name))
	for _, prefix := range reservedPrefixes {
		if value == prefix {
			return true
		}
	}
	return false
}
