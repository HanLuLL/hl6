package config

import "testing"

func TestLoadReadsSMTPBootstrapConfiguration(t *testing.T) {
	t.Setenv("APP_URL", "")
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("BACKEND_URL", "")
	t.Setenv("ENCRYPTION_KEY", "")
	t.Setenv("SMTP_BOOTSTRAP_HOST", "smtp.example.test")
	t.Setenv("SMTP_BOOTSTRAP_PORT", "465")
	t.Setenv("SMTP_BOOTSTRAP_USERNAME", "mailer@example.test")
	t.Setenv("SMTP_BOOTSTRAP_PASSWORD", "test-password")
	t.Setenv("SMTP_BOOTSTRAP_FROM_NAME", "HL6 Mail")
	t.Setenv("SMTP_BOOTSTRAP_FROM_ADDR", "noreply@example.test")
	t.Setenv("SMTP_BOOTSTRAP_USE_TLS", "true")
	t.Setenv("SMTP_BOOTSTRAP_ENABLED", "true")

	cfg := Load()
	if cfg.BootstrapSMTPHost != "smtp.example.test" || cfg.BootstrapSMTPPort != "465" {
		t.Fatalf("unexpected SMTP host/port: %#v", cfg)
	}
	if cfg.BootstrapSMTPUsername != "mailer@example.test" || cfg.BootstrapSMTPPassword != "test-password" {
		t.Fatalf("unexpected SMTP credentials: %#v", cfg)
	}
	if cfg.BootstrapSMTPFromName != "HL6 Mail" || cfg.BootstrapSMTPFromAddr != "noreply@example.test" {
		t.Fatalf("unexpected SMTP sender: %#v", cfg)
	}
	if !cfg.BootstrapSMTPUseTLS || !cfg.BootstrapSMTPEnabled {
		t.Fatalf("unexpected SMTP flags: %#v", cfg)
	}
}
