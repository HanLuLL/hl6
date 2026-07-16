package migration

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"hl6-server/internal/model"
)

func newAuthCutoverTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("HL6_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HL6_TEST_DATABASE_URL is required for PostgreSQL migration tests")
	}

	prefix := "auth_cutover_test_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "") + "_"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: prefix},
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{},
		&model.User{},
		&model.CreditBalance{},
		&model.Domain{},
		&model.SystemConfig{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate cutover fixtures: %v", err)
	}
	if err := InstallAuthSchema(db); err != nil {
		t.Fatalf("install auth schema: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Migrator().DropTable(
			&model.AuthSecurityEvent{},
			&model.AuthToken{},
			&model.UserCredential{},
			&model.DatabaseRestoreJob{},
			&model.DatabaseBackup{},
			&model.AuditLog{},
			&model.SystemConfig{},
			&model.Domain{},
			&model.CreditBalance{},
			&model.User{},
			&model.UserGroup{},
		)
	})
	return db
}

func TestPreflightReportsDuplicateNormalizedEmails(t *testing.T) {
	db := newAuthCutoverTestDB(t)
	group := model.UserGroup{Name: "Default", IsDefault: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"Person@example.com", "person@EXAMPLE.com"} {
		if err := db.Create(&model.User{Email: email, Name: email, Role: "user", GroupID: &group.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}

	report, err := AuthPreflight(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(report.Blockers, PreflightBlockerDuplicateNormalizedEmail) {
		t.Fatalf("duplicate-normalized-email blocker missing: %#v", report.Blockers)
	}
}

func TestCutoverRequiresConfirmation(t *testing.T) {
	db := newAuthCutoverTestDB(t)
	if _, err := AuthCutover(context.Background(), db, AuthCutoverOptions{}); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("got %v, want ErrConfirmationRequired", err)
	}
}

func TestCutoverCreatesActivationCredentialsWithoutChangingUserRecords(t *testing.T) {
	db := newAuthCutoverTestDB(t)
	group := model.UserGroup{Name: "Default", IsDefault: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{
		Email:     "123456@qq.com",
		Name:      "Custom name",
		AvatarURL: "https://cdn.example.test/custom.webp",
		Role:      "admin",
		GroupID:   &group.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CreditBalance{UserID: user.ID, Balance: 123}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Domain{Name: "example.test", Provider: "cloudflare", ProviderZoneID: "zone", CreditCost: 1}).Error; err != nil {
		t.Fatal(err)
	}

	usersTable := db.NamingStrategy.TableName("User")
	if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN external_id text`, quotedIdentifier(usersTable))).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(fmt.Sprintf(`CREATE TABLE %s (id bigserial primary key)`, quotedIdentifier(db.NamingStrategy.TableName("NativeAuthCode")))).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(fmt.Sprintf(`CREATE TABLE %s (id bigserial primary key)`, quotedIdentifier(db.NamingStrategy.TableName("NativeAuthRequest")))).Error; err != nil {
		t.Fatal(err)
	}

	archivePath, checksum := writeVerifiedBackupArchive(t)
	backup := model.DatabaseBackup{
		CreatedByUserID: user.ID,
		Filename:        "pre-cutover.dump.zip",
		ChecksumSHA256:  checksum,
		StoragePath:     archivePath,
		Status:          model.DatabaseBackupStatusReady,
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"frontend_urls":             "https://hl6.example.test",
		"smtp_enabled":              "true",
		"smtp_host":                 "smtp.example.test",
		"smtp_from_addr":            "noreply@example.test",
		"email.smtp.last_tested_at": time.Now().UTC().Format(time.RFC3339),
		"oidc_issuer":               "https://issuer.example.test",
		"auth.local.enabled":        "false",
	} {
		if err := db.Create(&model.SystemConfig{Key: key, Value: value}).Error; err != nil {
			t.Fatal(err)
		}
	}

	report, err := AuthCutover(context.Background(), db, AuthCutoverOptions{Confirmed: true, BackupID: backup.ID})
	if err != nil {
		t.Fatal(err)
	}
	if report.ActivationCredentialsCreated != 1 || report.BackupID != backup.ID {
		t.Fatalf("unexpected cutover report: %#v", report)
	}

	var credential model.UserCredential
	if err := db.Where("user_id = ?", user.ID).First(&credential).Error; err != nil {
		t.Fatal(err)
	}
	if credential.EmailNormalized != "123456@qq.com" || credential.ActivationRequiredAt == nil || credential.PasswordSetAt != nil {
		t.Fatalf("unexpected migrated credential: %#v", credential)
	}

	var reloaded model.User
	if err := db.First(&reloaded, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.ID != user.ID || reloaded.Name != user.Name || reloaded.AvatarURL != user.AvatarURL {
		t.Fatalf("user profile changed during cutover: %#v", reloaded)
	}
	var balance model.CreditBalance
	if err := db.Where("user_id = ?", user.ID).First(&balance).Error; err != nil || balance.Balance != 123 {
		t.Fatalf("user balance changed during cutover: balance=%#v err=%v", balance, err)
	}
	if externalIDExists, err := hasColumn(db, usersTable, "external_id"); err != nil || externalIDExists {
		t.Fatal("external_id column still exists after cutover")
	}
	if db.Migrator().HasTable(db.NamingStrategy.TableName("NativeAuthCode")) || db.Migrator().HasTable(db.NamingStrategy.TableName("NativeAuthRequest")) {
		t.Fatal("legacy native authentication tables still exist after cutover")
	}
	var oidcConfig model.SystemConfig
	if err := db.Where("\"key\" = ?", "oidc_issuer").First(&oidcConfig).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("OIDC configuration remains after cutover: %v", err)
	}
	var enabledConfig model.SystemConfig
	if err := db.Where("\"key\" = ?", "auth.local.enabled").First(&enabledConfig).Error; err != nil || enabledConfig.Value != "true" {
		t.Fatalf("local authentication was not enabled: %#v err=%v", enabledConfig, err)
	}
}

func TestCutoverAcceptsConfiguredFrontendURL(t *testing.T) {
	db := newAuthCutoverTestDB(t)
	group := model.UserGroup{Name: "Default", IsDefault: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{
		Email:   "legacy@example.com",
		Name:    "Legacy user",
		Role:    "admin",
		GroupID: &group.ID,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	archivePath, checksum := writeVerifiedBackupArchive(t)
	backup := model.DatabaseBackup{
		CreatedByUserID: user.ID,
		Filename:        "pre-cutover.dump.zip",
		ChecksumSHA256:  checksum,
		StoragePath:     archivePath,
		Status:          model.DatabaseBackupStatusReady,
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"smtp_enabled":              "true",
		"smtp_host":                 "smtp.example.test",
		"smtp_from_addr":            "noreply@example.test",
		"email.smtp.last_tested_at": time.Now().UTC().Format(time.RFC3339),
		"auth.local.enabled":        "false",
	} {
		if err := db.Create(&model.SystemConfig{Key: key, Value: value}).Error; err != nil {
			t.Fatal(err)
		}
	}

	_, err := AuthCutover(context.Background(), db, AuthCutoverOptions{
		Confirmed:          true,
		BackupID:           backup.ID,
		PublicFrontendURLs: []string{"https://hl6.example.test"},
	})
	if err != nil {
		t.Fatalf("cutover rejected the configured public frontend URL: %v", err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func writeVerifiedBackupArchive(t *testing.T) (string, string) {
	t.Helper()
	dump := []byte("PGDMPtest")
	dumpHash := sha256.Sum256(dump)
	manifest := struct {
		SchemaVersion int               `json:"schema_version"`
		Files         map[string]string `json:"files"`
	}{
		SchemaVersion: 1,
		Files: map[string]string{
			"database.dump": hex.EncodeToString(dumpHash[:]),
		},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestHash := sha256.Sum256(manifestBytes)
	checksums := fmt.Sprintf(
		"%s  database.dump\n%s  manifest.json\n",
		hex.EncodeToString(dumpHash[:]),
		hex.EncodeToString(manifestHash[:]),
	)

	archivePath := t.TempDir() + "/pre-cutover.dump.zip"
	file, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range []struct {
		name     string
		contents []byte
	}{
		{name: "database.dump", contents: dump},
		{name: "manifest.json", contents: manifestBytes},
		{name: "SHA256SUMS", contents: []byte(checksums)},
	} {
		output, createErr := writer.Create(entry.name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := output.Write(entry.contents); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	archiveHash := sha256.Sum256(archiveBytes)
	return archivePath, hex.EncodeToString(archiveHash[:])
}
