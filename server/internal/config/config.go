package config

import (
	"encoding/hex"
	"log"
	"os"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	OIDCIssuer      string
	OIDCClientID    string
	OIDCClientSecret string
	SessionSecret   string
	BackendURL      string
	FrontendURL     string
	AllowedOrigins  []string
	AdminEmails     []string
	EncryptionKey   []byte
}

func Load() *Config {
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:5173")
	cfg := &Config{
		Port:            getEnv("SERVER_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://hl6:hl6dev@localhost:5432/hl6?sslmode=disable"),
		OIDCIssuer:      getEnv("OIDC_ISSUER", ""),
		OIDCClientID:    getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),
		SessionSecret:   getEnv("SESSION_SECRET", ""),
		BackendURL:      getEnv("BACKEND_URL", frontendURL),
		FrontendURL:     frontendURL,
		AllowedOrigins:  parseList(getEnv("ALLOWED_ORIGINS", "")),
		AdminEmails:     parseListLower(getEnv("ADMIN_EMAILS", "")),
	}

	if keyHex := getEnv("ENCRYPTION_KEY", ""); keyHex != "" {
		key, err := hex.DecodeString(keyHex)
		if err != nil || len(key) != 32 {
			log.Fatal("ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
		}
		cfg.EncryptionKey = key
	}

	return cfg
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
