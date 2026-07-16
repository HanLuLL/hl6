package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"hl6-server/internal/auth"
	"hl6-server/internal/config"
	"hl6-server/internal/ctxutil"
	"hl6-server/internal/migration"
	"hl6-server/internal/model"
	"hl6-server/internal/repository"
	"hl6-server/internal/service"
)

func newEmailAuthHandlerForTest(t *testing.T) (*repository.Repository, *EmailAuthHandler) {
	t.Helper()

	dsn := os.Getenv("HL6_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HL6_TEST_DATABASE_URL is required for PostgreSQL handler tests")
	}

	prefix := "email_auth_test_" + uuid.NewString()[:8] + "_"
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
		&model.SystemConfig{},
		&model.EmailLog{},
	); err != nil {
		t.Fatalf("migrate handler test models: %v", err)
	}
	if err := migration.InstallAuthSchema(db); err != nil {
		t.Fatalf("install auth schema: %v", err)
	}
	if err := db.Create(&model.UserGroup{Name: "Default", IsDefault: true}).Error; err != nil {
		t.Fatalf("create default group: %v", err)
	}
	if err := db.Create(&model.SystemConfig{Key: "auth.local.enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("enable local auth: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Migrator().DropTable(
			&model.AuthSecurityEvent{},
			&model.AuthToken{},
			&model.UserCredential{},
			&model.DatabaseRestoreJob{},
			&model.DatabaseBackup{},
			&model.EmailLog{},
			&model.SystemConfig{},
			&model.AuditLog{},
			&model.UserReferral{},
			&model.CreditBalance{},
			&model.User{},
			&model.UserGroup{},
		)
	})

	repo := repository.New(db)
	cfg := &config.Config{
		SessionSecret:        "test-session-secret-with-enough-entropy",
		FrontendURL:          "https://example.test",
		FrontendURLs:         []string{"https://example.test"},
		FrontendURLEnvSet:    true,
		AuthPasswordPepperID: "test-v1",
		AuthPasswordPepper:   "test-password-pepper",
	}
	return repo, NewEmailAuthHandler(repo, service.NewEmailService(repo, nil), cfg)
}

func TestForgotPasswordDoesNotExposeAccountExistence(t *testing.T) {
	repo, authHandler := newEmailAuthHandlerForTest(t)
	var defaultGroup model.UserGroup
	if err := repo.GetDB().Where("is_default = ?", true).First(&defaultGroup).Error; err != nil {
		t.Fatal(err)
	}
	knownUser := model.User{
		Email:   "known@example.com",
		Name:    "known",
		Role:    "user",
		GroupID: &defaultGroup.ID,
	}
	if err := repo.GetDB().Create(&knownUser).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := repo.GetDB().Create(&model.UserCredential{
		UserID:          knownUser.ID,
		EmailNormalized: "known@example.com",
		PasswordHash:    "$argon2id$test",
		EmailVerifiedAt: &now,
		PasswordSetAt:   &now,
		SessionVersion:  1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/api/v1/auth/password/forgot", authHandler.ForgotPassword)

	known := postEmailAuthJSON(router, "/api/v1/auth/password/forgot", `{"email":"known@example.com"}`)
	unknown := postEmailAuthJSON(router, "/api/v1/auth/password/forgot", `{"email":"unknown@example.com"}`)
	if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted {
		t.Fatalf("account enumeration response: known=%d unknown=%d", known.Code, unknown.Code)
	}
}

func TestPasswordCompletionConsumesToken(t *testing.T) {
	repo, authHandler := newEmailAuthHandlerForTest(t)
	var defaultGroup model.UserGroup
	if err := repo.GetDB().Where("is_default = ?", true).First(&defaultGroup).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{
		Email:   "existing@example.com",
		Name:    "existing",
		Role:    "user",
		GroupID: &defaultGroup.ID,
	}
	if err := repo.GetDB().Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	credential := model.UserCredential{
		UserID:               user.ID,
		EmailNormalized:      "existing@example.com",
		SessionVersion:       1,
		ActivationRequiredAt: &now,
	}
	if err := repo.GetDB().Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	raw, err := auth.NewRawToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAuthToken(&model.AuthToken{
		Purpose:         model.AuthTokenPurposeAccountActivation,
		UserID:          &user.ID,
		EmailNormalized: credential.EmailNormalized,
		TokenHash:       auth.HashToken(raw),
		ExpiresAt:       time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/api/v1/auth/password/complete", authHandler.CompletePassword)
	body := `{"token":"` + raw + `","password":"correct horse battery staple"}`
	first := postEmailAuthJSON(router, "/api/v1/auth/password/complete", body)
	second := postEmailAuthJSON(router, "/api/v1/auth/password/complete", body)
	if first.Code != http.StatusOK || second.Code != http.StatusBadRequest {
		t.Fatalf("got %d then %d", first.Code, second.Code)
	}
}

func TestLogoutInvalidatesIssuedSessions(t *testing.T) {
	repo, authHandler := newEmailAuthHandlerForTest(t)
	var defaultGroup model.UserGroup
	if err := repo.GetDB().Where("is_default = ?", true).First(&defaultGroup).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{Email: "logout@example.com", Name: "logout", Role: "user", GroupID: &defaultGroup.ID}
	if err := repo.GetDB().Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := repo.GetDB().Create(&model.UserCredential{
		UserID:          user.ID,
		EmailNormalized: "logout@example.com",
		PasswordHash:    "$argon2id$test",
		EmailVerifiedAt: &now,
		PasswordSetAt:   &now,
		SessionVersion:  1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/api/v1/auth/logout", func(c *gin.Context) {
		ctxutil.SetUser(c, &user)
		authHandler.Logout(c)
	})
	response := postEmailAuthJSON(router, "/api/v1/auth/logout", `{}`)
	if response.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
	}
	credential, err := repo.FindCredentialByUserID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if credential.SessionVersion != 2 {
		t.Fatalf("got session version %d, want 2", credential.SessionVersion)
	}
}

func postEmailAuthJSON(router http.Handler, path string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
