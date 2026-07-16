package repository

import (
	"context"
	"errors"
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

func newPostgresTestRepository(t *testing.T) *Repository {
	t.Helper()

	dsn := os.Getenv("HL6_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HL6_TEST_DATABASE_URL is required for PostgreSQL repository tests")
	}

	prefix := "auth_test_" + uuid.NewString()[:8] + "_"
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
		&model.UserReferral{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("migrate supporting models: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserCredential{},
		&model.AuthToken{},
		&model.AuthSecurityEvent{},
		&model.DatabaseBackup{},
		&model.DatabaseRestoreJob{},
	); err != nil {
		t.Fatalf("migrate auth test models: %v", err)
	}
	if err := db.Create(&model.UserGroup{Name: "Default", IsDefault: true}).Error; err != nil {
		t.Fatalf("create default group: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Migrator().DropTable(
			&model.AuthSecurityEvent{},
			&model.AuthToken{},
			&model.UserCredential{},
			&model.DatabaseRestoreJob{},
			&model.DatabaseBackup{},
			&model.AuditLog{},
			&model.UserReferral{},
			&model.CreditBalance{},
			&model.User{},
			&model.UserGroup{},
		)
	})

	return New(db)
}

func TestCreateUserWithCredentialLinksOriginalUser(t *testing.T) {
	repo := newPostgresTestRepository(t)

	user, credential, err := repo.CreateUserWithCredential(context.Background(), NewUserInput{
		Email:           "new@example.com",
		EmailNormalized: "new@example.com",
		Name:            "new",
		PasswordHash:    "$argon2id$v=19$m=65536,t=3,p=4,pepper=v1$c2FsdA$aGFzaA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 || credential.UserID != user.ID {
		t.Fatalf("credential is not linked to the created user: user=%d credential=%d", user.ID, credential.UserID)
	}
	if credential.EmailNormalized != "new@example.com" || credential.SessionVersion == 0 {
		t.Fatalf("unexpected credential: %#v", credential)
	}
	if _, err := repo.GetCreditBalance(user.ID); err != nil {
		t.Fatalf("credit balance was not created: %v", err)
	}
}

func TestConsumeAuthTokenAllowsOnlyOneSuccessfulConsumer(t *testing.T) {
	repo := newPostgresTestRepository(t)
	now := time.Now()
	token := model.AuthToken{
		Purpose:         model.AuthTokenPurposePasswordReset,
		EmailNormalized: "existing@example.com",
		TokenHash:       strings.Repeat("a", 64),
		ExpiresAt:       now.Add(time.Hour),
	}
	if err := repo.CreateAuthToken(&token); err != nil {
		t.Fatal(err)
	}

	first, err := repo.ConsumeAuthToken(context.Background(), token.TokenHash, token.Purpose)
	if err != nil {
		t.Fatalf("consume first token use: %v", err)
	}
	if first.ConsumedAt == nil {
		t.Fatal("consumed token did not record consumed_at")
	}
	if _, err := repo.ConsumeAuthToken(context.Background(), token.TokenHash, token.Purpose); err == nil {
		t.Fatal("consumed token was accepted twice")
	}
}

func TestIncrementSessionVersionInvalidatesPriorSessions(t *testing.T) {
	repo := newPostgresTestRepository(t)
	user, credential, err := repo.CreateUserWithCredential(context.Background(), NewUserInput{
		Email:           "version@example.com",
		EmailNormalized: "version@example.com",
		Name:            "version",
		PasswordHash:    "$argon2id$v=19$m=65536,t=3,p=4,pepper=v1$c2FsdA$aGFzaA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.IncrementSessionVersion(user.ID); err != nil {
		t.Fatal(err)
	}

	reloaded, err := repo.FindCredentialByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.SessionVersion != credential.SessionVersion+1 {
		t.Fatalf("got session version %d, want %d", reloaded.SessionVersion, credential.SessionVersion+1)
	}
}

func TestCountRecentAuthSecurityEventsCountsEmailAndIPIndependently(t *testing.T) {
	repo := newPostgresTestRepository(t)
	createdAt := time.Now().UTC()
	for _, email := range []string{"target@example.com", "other@example.com"} {
		if err := repo.CreateAuthSecurityEvent(&model.AuthSecurityEvent{
			Action:          "login",
			Outcome:         model.AuthSecurityOutcomeFailure,
			EmailNormalized: email,
			IPHash:          "shared-proxy-ip",
			CreatedAt:       createdAt,
		}); err != nil {
			t.Fatal(err)
		}
	}

	emailCount, err := repo.CountRecentAuthSecurityEventsForEmail("login", "target@example.com", createdAt.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if emailCount != 1 {
		t.Fatalf("got %d events for target email, want 1", emailCount)
	}
	ipCount, err := repo.CountRecentAuthSecurityEventsForIP("login", "shared-proxy-ip", createdAt.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if ipCount != 2 {
		t.Fatalf("got %d events for shared IP, want 2", ipCount)
	}
}

func TestUpdateCredentialPasswordInvalidatesOutstandingRecoveryTokens(t *testing.T) {
	repo := newPostgresTestRepository(t)
	user, _, err := repo.CreateUserWithCredential(context.Background(), NewUserInput{
		Email:               "recover@example.com",
		EmailNormalized:     "recover@example.com",
		Name:                "recover",
		PasswordHash:        "$argon2id$v=19$m=65536,t=3,p=4,pepper=v1$c2FsdA$aGFzaA",
		PasswordHashVersion: "argon2id",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tokenHash := range []string{strings.Repeat("b", 64), strings.Repeat("c", 64)} {
		if err := repo.CreateAuthToken(&model.AuthToken{
			Purpose:         model.AuthTokenPurposePasswordReset,
			UserID:          &user.ID,
			EmailNormalized: "recover@example.com",
			TokenHash:       tokenHash,
			ExpiresAt:       time.Now().UTC().Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.UpdateCredentialPassword(context.Background(), user.ID, "new-password-hash", "argon2id"); err != nil {
		t.Fatal(err)
	}
	var activeTokens int64
	if err := repo.GetDB().Model(&model.AuthToken{}).
		Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", user.ID, model.AuthTokenPurposePasswordReset).
		Count(&activeTokens).Error; err != nil {
		t.Fatal(err)
	}
	if activeTokens != 0 {
		t.Fatalf("got %d active reset tokens after password change, want 0", activeTokens)
	}
}

func TestUpdateCredentialPasswordFromAuthTokenRejectsConsumedSiblingToken(t *testing.T) {
	repo := newPostgresTestRepository(t)
	user, credential, err := repo.CreateUserWithCredential(context.Background(), NewUserInput{
		Email:               "race@example.com",
		EmailNormalized:     "race@example.com",
		Name:                "race",
		PasswordHash:        "$argon2id$v=19$m=65536,t=3,p=4,pepper=v1$c2FsdA$aGFzaA",
		PasswordHashVersion: "argon2id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.GetDB().Model(&model.UserCredential{}).Where("id = ?", credential.ID).Update("password_set_at", time.Now().UTC().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}

	for _, tokenHash := range []string{strings.Repeat("d", 64), strings.Repeat("e", 64)} {
		if err := repo.CreateAuthToken(&model.AuthToken{
			Purpose:         model.AuthTokenPurposePasswordReset,
			UserID:          &user.ID,
			EmailNormalized: "race@example.com",
			TokenHash:       tokenHash,
			ExpiresAt:       time.Now().UTC().Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}

	first, err := repo.ConsumeAuthToken(context.Background(), strings.Repeat("d", 64), model.AuthTokenPurposePasswordReset)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.ConsumeAuthToken(context.Background(), strings.Repeat("e", 64), model.AuthTokenPurposePasswordReset)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repo.UpdateCredentialPasswordFromAuthToken(context.Background(), first, "first-password-hash", "argon2id"); err != nil {
		t.Fatalf("complete first token: %v", err)
	}
	if _, err := repo.UpdateCredentialPasswordFromAuthToken(context.Background(), second, "second-password-hash", "argon2id"); !errors.Is(err, ErrAuthTokenUnavailable) {
		t.Fatalf("second consumed token error = %v, want ErrAuthTokenUnavailable", err)
	}

	reloaded, err := repo.FindCredentialByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.PasswordHash != "first-password-hash" {
		t.Fatalf("password hash was overwritten by sibling token: %q", reloaded.PasswordHash)
	}
}
