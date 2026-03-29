package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"hl6-server/internal/repository"
)

const (
	configKeyReservedSubdomainPrefixes = "reserved_subdomain_prefixes"
	configKeySubdomainMinLength        = "subdomain_min_length"
	configKeySubdomainMaxLength        = "subdomain_max_length"
	defaultSubdomainMinLength          = 1
	defaultSubdomainMaxLength          = 63
	minAllowedSubdomainLength          = 1
	maxAllowedSubdomainLength          = 63
)

var reservedSubdomainPrefixPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type subdomainLengthSettings struct {
	MinLength int `json:"min_length"`
	MaxLength int `json:"max_length"`
}

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

func defaultLengthSettings() subdomainLengthSettings {
	return subdomainLengthSettings{
		MinLength: defaultSubdomainMinLength,
		MaxLength: defaultSubdomainMaxLength,
	}
}

func validateLengthSettings(minLength, maxLength int) error {
	if minLength < minAllowedSubdomainLength || maxLength > maxAllowedSubdomainLength || minLength > maxLength {
		return fmt.Errorf("invalid subdomain length settings: min=%d max=%d", minLength, maxLength)
	}
	return nil
}

func loadSubdomainLengthSettings(repo *repository.Repository) (subdomainLengthSettings, error) {
	configs, err := repo.GetSystemConfigsByKeys([]string{
		configKeySubdomainMinLength,
		configKeySubdomainMaxLength,
	})
	if err != nil {
		return defaultLengthSettings(), err
	}

	settings := defaultLengthSettings()
	if rawMin, ok := configs[configKeySubdomainMinLength]; ok {
		if parsed, parseErr := strconv.Atoi(strings.TrimSpace(rawMin)); parseErr == nil {
			settings.MinLength = parsed
		}
	}
	if rawMax, ok := configs[configKeySubdomainMaxLength]; ok {
		if parsed, parseErr := strconv.Atoi(strings.TrimSpace(rawMax)); parseErr == nil {
			settings.MaxLength = parsed
		}
	}

	if err := validateLengthSettings(settings.MinLength, settings.MaxLength); err != nil {
		return defaultLengthSettings(), nil
	}
	return settings, nil
}

func saveSubdomainLengthSettings(repo *repository.Repository, settings subdomainLengthSettings) error {
	if err := validateLengthSettings(settings.MinLength, settings.MaxLength); err != nil {
		return err
	}
	if err := repo.SetSystemConfig(configKeySubdomainMinLength, strconv.Itoa(settings.MinLength)); err != nil {
		return err
	}
	return repo.SetSystemConfig(configKeySubdomainMaxLength, strconv.Itoa(settings.MaxLength))
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
