package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"hl6-server/internal/auth"
	"hl6-server/internal/config"
	"hl6-server/internal/migration"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

const activationRecoveryTTL = 30 * time.Minute

func main() {
	_ = godotenv.Load("../.env", ".env")
	if err := run(os.Args[1:]); err != nil {
		log.Printf("hl6-admin: %v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return usageError()
	}

	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	if err := migration.InstallAuthSchema(db); err != nil {
		return fmt.Errorf("install auth schema: %w", err)
	}

	switch args[0] {
	case "auth":
		switch args[1] {
		case "preflight":
			return runPreflight(db, cfg)
		case "cutover":
			return runCutover(db, cfg, args[2:])
		case "issue-activation":
			return runIssueActivation(db, cfg, args[2:])
		default:
			return usageError()
		}
	case "maintenance":
		if args[1] == "export" {
			return runMaintenanceExport(db, cfg, args[2:])
		}
		return usageError()
	case "mail":
		if args[1] == "test" {
			return runSMTPTest(db, cfg, args[2:])
		}
		return usageError()
	default:
		return usageError()
	}
}

func runPreflight(db *gorm.DB, cfg *config.Config) error {
	report, err := migration.AuthPreflightWithOptions(context.Background(), db, migration.AuthPreflightOptions{
		PublicFrontendURLs: configuredFrontendURLs(cfg),
	})
	if err != nil {
		return err
	}
	if err := printJSON(report); err != nil {
		return err
	}
	if len(report.Blockers) > 0 {
		return fmt.Errorf("preflight blocked: %s", strings.Join(report.Blockers, ", "))
	}
	return nil
}

func runMaintenanceExport(db *gorm.DB, cfg *config.Config, args []string) error {
	flags := flag.NewFlagSet("maintenance export", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	operatorID := flags.Uint("created-by-user-id", 0, "existing administrator user ID recorded as the export operator")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *operatorID == 0 {
		return errors.New("maintenance export requires --created-by-user-id")
	}

	repo := repository.New(db)
	operator, err := repo.FindUserByID(uint(*operatorID))
	if err != nil {
		return errors.New("the export operator does not exist")
	}
	if operator.Role != "admin" && (operator.Group == nil || !operator.Group.IsAdmin) {
		return errors.New("the export operator must be an administrator")
	}

	maintenance := service.NewDatabaseMaintenanceService(db, repo, cfg, service.NewDatabaseMaintenanceGate())
	backup, err := maintenance.CreateBackup(context.Background(), operator.ID)
	if err != nil {
		return fmt.Errorf("create database export: %w", err)
	}
	return printJSON(backup)
}

func runSMTPTest(db *gorm.DB, cfg *config.Config, args []string) error {
	flags := flag.NewFlagSet("mail test", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	recipient := flags.String("recipient", "", "email address that receives the SMTP verification message")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return usageError()
	}
	normalizedRecipient, err := auth.NormalizeEmail(*recipient)
	if err != nil {
		return errors.New("a valid --recipient is required")
	}

	repo := repository.New(db)
	siteName := "HL6"
	if configuredName, getErr := repo.GetSystemConfig("brand_name"); getErr == nil && strings.TrimSpace(configuredName) != "" {
		siteName = strings.TrimSpace(configuredName)
	}
	emailSvc := service.NewEmailService(repo, cfg.EncryptionKey)
	if err := emailSvc.SendTestEmail(normalizedRecipient, siteName); err != nil {
		return fmt.Errorf("send SMTP test: %w", err)
	}
	testedAt := time.Now().UTC()
	if err := repo.SetSystemConfig("email.smtp.last_tested_at", testedAt.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("record SMTP test: %w", err)
	}
	return printJSON(map[string]string{
		"recipient": normalizedRecipient,
		"tested_at": testedAt.Format(time.RFC3339),
	})
}

func runCutover(db *gorm.DB, cfg *config.Config, args []string) error {
	flags := flag.NewFlagSet("auth cutover", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	confirmed := flags.Bool("confirm", false, "confirm irreversible local-authentication cutover")
	backupID := flags.Uint("backup-id", 0, "verified database backup ID created by HL6")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return usageError()
	}

	report, err := migration.AuthCutover(context.Background(), db, migration.AuthCutoverOptions{
		Confirmed:          *confirmed,
		BackupID:           *backupID,
		PublicFrontendURLs: configuredFrontendURLs(cfg),
	})
	if err != nil {
		return err
	}
	return printJSON(report)
}

func runIssueActivation(db *gorm.DB, cfg *config.Config, args []string) error {
	flags := flag.NewFlagSet("auth issue-activation", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	email := flags.String("email", "", "existing account email address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return usageError()
	}
	normalizedEmail, err := auth.NormalizeEmail(*email)
	if err != nil {
		return errors.New("a valid --email is required")
	}

	repo := repository.New(db)
	localAuthEnabled, err := repo.GetSystemConfig("auth.local.enabled")
	if err != nil || localAuthEnabled != "true" {
		return errors.New("local authentication is not enabled; run a successful auth cutover first")
	}
	credential, err := repo.FindCredentialByEmail(normalizedEmail)
	if err != nil {
		return errors.New("no activation-required account matches --email")
	}
	if credential.ActivationRequiredAt == nil || credential.PasswordSetAt != nil {
		return errors.New("the account is already active")
	}

	frontendURL, err := resolveFrontendURL(cfg, repo)
	if err != nil {
		return err
	}
	rawToken, err := auth.NewRawToken()
	if err != nil {
		return err
	}
	token := &model.AuthToken{
		Purpose:         model.AuthTokenPurposeAccountActivation,
		UserID:          &credential.UserID,
		EmailNormalized: credential.EmailNormalized,
		TokenHash:       auth.HashToken(rawToken),
		ExpiresAt:       time.Now().UTC().Add(activationRecoveryTTL),
	}
	if err := repo.CreateAuthToken(token); err != nil {
		return fmt.Errorf("create activation token: %w", err)
	}
	details, _ := json.Marshal(map[string]string{"issued_by": "console_recovery"})
	if err := repo.CreateAuditLog(&model.AuditLog{
		UserID:     credential.UserID,
		Action:     "auth_issue_activation_recovery",
		Resource:   "authentication",
		ResourceID: credential.UserID,
		Details:    details,
	}); err != nil {
		return fmt.Errorf("write activation recovery audit log: %w", err)
	}

	link := strings.TrimRight(frontendURL, "/") + "/set-password?token=" + rawToken
	// This value is intentionally only emitted to the operator's protected
	// deployment console. It is never written to the database or application log.
	fmt.Printf("Deliver this one-time activation link only through a secure channel. It expires at %s:\n%s\n", token.ExpiresAt.Format(time.RFC3339), link)
	return nil
}

func resolveFrontendURL(cfg *config.Config, repo *repository.Repository) (string, error) {
	for _, candidate := range configuredFrontendURLs(cfg) {
		if strings.HasPrefix(candidate, "https://") {
			return strings.TrimRight(candidate, "/"), nil
		}
	}
	configs, err := repo.GetSystemConfigsByKeys([]string{"frontend_urls", "frontend_url"})
	if err != nil {
		return "", err
	}
	for _, raw := range []string{configs["frontend_urls"], configs["frontend_url"]} {
		urls, parseErr := config.ParsePublicURLList(raw)
		if parseErr != nil {
			continue
		}
		for _, candidate := range urls {
			if strings.HasPrefix(candidate, "https://") {
				return strings.TrimRight(candidate, "/"), nil
			}
		}
	}
	return "", errors.New("a public HTTPS FRONTEND_URL or frontend_urls setting is required")
}

func configuredFrontendURLs(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	return append([]string(nil), cfg.FrontendURLs...)
}

func printJSON(value interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func usageError() error {
	return errors.New("usage: hl6-admin auth preflight | auth cutover --confirm --backup-id ID | auth issue-activation --email address@example.com | maintenance export --created-by-user-id ID | mail test --recipient address@example.com")
}
