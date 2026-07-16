package config

import (
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	SessionSecret         string
	AuthPasswordPepperID  string
	AuthPasswordPepper    string
	AuthPreviousPepperID  string
	AuthPreviousPepper    string
	BootstrapSMTPHost     string
	BootstrapSMTPPort     string
	BootstrapSMTPUsername string
	BootstrapSMTPPassword string
	BootstrapSMTPFromName string
	BootstrapSMTPFromAddr string
	BootstrapSMTPUseTLS   bool
	BootstrapSMTPEnabled  bool
	MaintenanceDataDir    string
	AppURL                string
	BackendURLs           []string
	FrontendURLs          []string
	BackendURL            string
	FrontendURL           string
	BackendURLEnvSet      bool
	FrontendURLEnvSet     bool
	AllowedOrigins        []string
	EncryptionKey         []byte
	DNSBatchThreshold     int
	RedisAddr             string
	AuditScanInterval     time.Duration
	AuditScanWorkerCount  int
	AuditScanTimeout      time.Duration
}

func Load() *Config {
	sharedURL, err := NormalizePublicURL(getEnv("APP_URL", ""))
	if err != nil {
		log.Fatalf("invalid APP_URL: %v", err)
	}
	frontendURLs, err := ParsePublicURLList(getEnv("FRONTEND_URL", ""))
	if err != nil {
		log.Fatalf("invalid FRONTEND_URL: %v", err)
	}
	backendURLs, err := ParsePublicURLList(getEnv("BACKEND_URL", ""))
	if err != nil {
		log.Fatalf("invalid BACKEND_URL: %v", err)
	}
	if len(frontendURLs) == 0 && sharedURL != "" {
		frontendURLs = []string{sharedURL}
	}
	if len(backendURLs) == 0 && sharedURL != "" {
		backendURLs = []string{sharedURL}
	}
	effectiveFrontendURL := firstOrEmpty(frontendURLs)
	effectiveBackendURL := firstOrEmpty(backendURLs)
	databaseURL := expandEnvRefs(getEnv("DATABASE_URL", "postgres://hl6:hl6dev@localhost:5433/hl6?sslmode=disable"))

	cfg := &Config{
		Port:                  getEnv("SERVER_PORT", "8081"),
		DatabaseURL:           databaseURL,
		SessionSecret:         getEnv("SESSION_SECRET", ""),
		AuthPasswordPepperID:  getEnv("AUTH_PASSWORD_PEPPER_ID", "v1"),
		AuthPasswordPepper:    getEnv("AUTH_PASSWORD_PEPPER", ""),
		AuthPreviousPepperID:  getEnv("AUTH_PREVIOUS_PASSWORD_PEPPER_ID", ""),
		AuthPreviousPepper:    getEnv("AUTH_PREVIOUS_PASSWORD_PEPPER", ""),
		BootstrapSMTPHost:     strings.TrimSpace(getEnv("SMTP_BOOTSTRAP_HOST", "")),
		BootstrapSMTPPort:     strconv.Itoa(getEnvInt("SMTP_BOOTSTRAP_PORT", 587)),
		BootstrapSMTPUsername: strings.TrimSpace(getEnv("SMTP_BOOTSTRAP_USERNAME", "")),
		BootstrapSMTPPassword: getEnv("SMTP_BOOTSTRAP_PASSWORD", ""),
		BootstrapSMTPFromName: strings.TrimSpace(getEnv("SMTP_BOOTSTRAP_FROM_NAME", "HL6")),
		BootstrapSMTPFromAddr: strings.TrimSpace(getEnv("SMTP_BOOTSTRAP_FROM_ADDR", "")),
		BootstrapSMTPUseTLS:   getEnvBool("SMTP_BOOTSTRAP_USE_TLS", true),
		BootstrapSMTPEnabled:  getEnvBool("SMTP_BOOTSTRAP_ENABLED", false),
		MaintenanceDataDir:    getEnv("MAINTENANCE_DATA_DIR", "./data/maintenance"),
		AppURL:                sharedURL,
		BackendURLs:           backendURLs,
		FrontendURLs:          frontendURLs,
		BackendURL:            effectiveBackendURL,
		FrontendURL:           effectiveFrontendURL,
		BackendURLEnvSet:      len(backendURLs) > 0,
		FrontendURLEnvSet:     len(frontendURLs) > 0,
		AllowedOrigins:        parseList(getEnv("ALLOWED_ORIGINS", "")),
		DNSBatchThreshold:     getEnvInt("DNS_BATCH_ASYNC_THRESHOLD", getEnvInt("DNS_BATCH_THRESHOLD", 200)),
		RedisAddr:             strings.TrimSpace(getEnv("REDIS_ADDR", "")),
		AuditScanInterval:     getEnvDuration("AUDIT_SCAN_INTERVAL", 30*time.Minute),
		AuditScanWorkerCount:  getEnvInt("AUDIT_SCAN_WORKER_COUNT", 2),
		AuditScanTimeout:      getEnvDuration("AUDIT_SCAN_TIMEOUT", 15*time.Second),
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func expandEnvRefs(value string) string {
	return os.Expand(value, func(key string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		// Keep unresolved placeholders untouched for easier troubleshooting.
		return "${" + key + "}"
	})
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

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid %s=%q, fallback to %d", key, raw, fallback)
		return fallback
	}
	return v
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid %s=%q, fallback to %s", key, raw, fallback)
		return fallback
	}
	return d
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("invalid %s=%q, fallback to %t", key, raw, fallback)
		return fallback
	}
	return value
}
