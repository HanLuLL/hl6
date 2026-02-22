package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	LogtoEndpoint   string
	LogtoAppID      string
	LogtoAppSecret  string
	SessionSecret   string
	FrontendURL     string
	AllowedOrigins  []string
	AdminEmails     []string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("SERVER_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://hl6:hl6dev@localhost:5432/hl6?sslmode=disable"),
		LogtoEndpoint:   getEnv("LOGTO_ENDPOINT", ""),
		LogtoAppID:      getEnv("LOGTO_APP_ID", ""),
		LogtoAppSecret:  getEnv("LOGTO_APP_SECRET", ""),
		SessionSecret:   getEnv("SESSION_SECRET", ""),
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:5173"),
		AllowedOrigins:  parseList(getEnv("ALLOWED_ORIGINS", "")),
		AdminEmails:     parseListLower(getEnv("ADMIN_EMAILS", "")),
	}
}

func (c *Config) IsAdminEmail(email string) bool {
	lower := strings.ToLower(email)
	for _, e := range c.AdminEmails {
		if e == lower {
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

func parseListLower(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			result = append(result, strings.ToLower(v))
		}
	}
	return result
}
