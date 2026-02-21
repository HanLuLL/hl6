package config

import (
	"os"
	"strings"
)

type Config struct {
	Port             string
	DatabaseURL      string
	LogtoEndpoint    string
	LogtoAPIResource string
	CloudflareToken  string
	AdminEmails      []string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("SERVER_PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://hl6:hl6dev@localhost:5432/hl6?sslmode=disable"),
		LogtoEndpoint:    getEnv("LOGTO_ENDPOINT", ""),
		LogtoAPIResource: getEnv("LOGTO_API_RESOURCE", ""),
		CloudflareToken:  getEnv("CLOUDFLARE_API_TOKEN", ""),
		AdminEmails:      parseList(getEnv("ADMIN_EMAILS", "")),
	}
}

func (c *Config) IsAdminEmail(email string) bool {
	for _, e := range c.AdminEmails {
		if e == email {
			return true
		}
	}
	return false
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			result = append(result, v)
		}
	}
	return result
}
