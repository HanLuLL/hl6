package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/oidc"
	"hl6-server/internal/router"
)

const internalSessionSecretKey = "_internal_session_secret"

func main() {
	godotenv.Load("../.env")

	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// Rename logto_id column to external_id (idempotent)
	var colExists bool
	db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'logto_id')").Scan(&colExists)
	if colExists {
		if err := db.Exec("ALTER TABLE users RENAME COLUMN logto_id TO external_id").Error; err != nil {
			log.Fatal("failed to rename logto_id column:", err)
		}
		log.Println("Renamed column users.logto_id → external_id")
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.UserGroup{},
		&model.CloudflareAccount{},
		&model.Domain{},
		&model.DomainGroupAccess{},
		&model.Subdomain{},
		&model.DNSRecord{},
		&model.CreditBalance{},
		&model.CreditTransaction{},
		&model.AuditLog{},
		&model.SystemConfig{},
		&model.Notification{},
		&model.NotificationRead{},
		&model.NotificationImage{},
		&model.BrandingAsset{},
		&model.UserReferral{},
	); err != nil {
		log.Fatal("failed to migrate:", err)
	}

	if err := verifyRequiredTables(db, []interface{}{
		&model.User{},
		&model.UserGroup{},
		&model.SystemConfig{},
	}); err != nil {
		log.Fatal("database schema verification failed:", err)
	}

	log.Println("Database migrated successfully")

	// GIN index for JSONB target_ids queries
	db.Exec("CREATE INDEX IF NOT EXISTS idx_notifications_target_ids ON notifications USING GIN (target_ids)")

	migrateCreditsToInt(db)
	seedDefaults(db)
	bootstrapSessionSecret(db, cfg)

	// OIDC Discovery
	provider, err := oidc.Discover(context.Background(), cfg.OIDCIssuer)
	if err != nil {
		log.Fatal("OIDC discovery failed: ", err)
	}
	log.Printf("OIDC provider discovered: issuer=%s", provider.Issuer)

	r := router.Setup(cfg, db, provider)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("failed to start server:", err)
	}
}

func bootstrapSessionSecret(db *gorm.DB, cfg *config.Config) {
	seed := strings.TrimSpace(cfg.SessionSecret)
	secret, source, err := resolveSessionSecret(db, seed)
	if err != nil {
		log.Fatal("failed to bootstrap session secret:", err)
	}
	if seed != "" && source == "database" && seed != secret {
		log.Println("SESSION_SECRET env is ignored because database session secret already exists")
	}
	cfg.SessionSecret = secret
	log.Printf("Session secret loaded from %s", source)
}

func resolveSessionSecret(db *gorm.DB, seed string) (string, string, error) {
	existing, err := getSystemConfigValue(db, internalSessionSecretKey)
	if err == nil {
		if existing == "" {
			return "", "", errors.New("stored session secret is empty")
		}
		return existing, "database", nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", err
	}

	secret := seed
	source := "generated random secret"
	if secret == "" {
		generated, genErr := generateHexSecret(32)
		if genErr != nil {
			return "", "", genErr
		}
		secret = generated
	} else {
		source = "SESSION_SECRET env seed"
	}

	createErr := db.Create(&model.SystemConfig{
		Key:   internalSessionSecretKey,
		Value: secret,
	}).Error
	if createErr == nil {
		return secret, source, nil
	}

	// If another instance raced to create the row, read the existing value.
	existing, readErr := getSystemConfigValue(db, internalSessionSecretKey)
	if readErr != nil {
		return "", "", fmt.Errorf("failed to save session secret: %w (fallback read failed: %v)", createErr, readErr)
	}
	if existing == "" {
		return "", "", errors.New("stored session secret is empty")
	}
	return existing, "database", nil
}

func getSystemConfigValue(db *gorm.DB, key string) (string, error) {
	var cfg model.SystemConfig
	if err := db.Where("\"key\" = ?", key).First(&cfg).Error; err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.Value), nil
}

func generateHexSecret(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func verifyRequiredTables(db *gorm.DB, tables []interface{}) error {
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			return fmt.Errorf("missing table for %T", table)
		}
	}
	return nil
}

func seedDefaults(db *gorm.DB) {
	// 1. Ensure a default user group exists
	var groupCount int64
	db.Model(&model.UserGroup{}).Count(&groupCount)
	if groupCount == 0 {
		db.Create(&model.UserGroup{Name: "Default", IsDefault: true})
	}

	// 2. Assign all users without a group to the default group
	var defaultGroup model.UserGroup
	if err := db.Where("is_default = ?", true).First(&defaultGroup).Error; err == nil {
		db.Model(&model.User{}).Where("group_id IS NULL").Update("group_id", defaultGroup.ID)
	}

	// 3. Migrate Domain.CreditCost to DomainGroupAccess for default group
	if defaultGroup.ID > 0 {
		var domains []model.Domain
		db.Find(&domains)
		for _, d := range domains {
			var existing int64
			db.Model(&model.DomainGroupAccess{}).Where("domain_id = ?", d.ID).Count(&existing)
			if existing == 0 {
				db.Create(&model.DomainGroupAccess{
					DomainID:   d.ID,
					GroupID:    defaultGroup.ID,
					CreditCost: d.CreditCost,
				})
			}
		}
	}

	// 4. Ensure registration_bonus_credits config exists
	var cfg model.SystemConfig
	if db.Where("\"key\" = ?", "registration_bonus_credits").First(&cfg).Error != nil {
		db.Create(&model.SystemConfig{Key: "registration_bonus_credits", Value: "0"})
	}

	// 5. Seed referral config defaults
	referralConfigs := map[string]string{
		"referral_enabled":         "true",
		"referral_inviter_credits": "0",
		"referral_invitee_credits": "0",
	}
	for key, defaultVal := range referralConfigs {
		var rc model.SystemConfig
		if db.Where("\"key\" = ?", key).First(&rc).Error != nil {
			db.Create(&model.SystemConfig{Key: key, Value: defaultVal})
		}
	}

	// 6. Backfill referral_code for existing users
	var usersWithoutCode []model.User
	db.Where("referral_code = '' OR referral_code IS NULL").Find(&usersWithoutCode)
	for _, u := range usersWithoutCode {
		code := generateReferralCode()
		db.Model(&model.User{}).Where("id = ?", u.ID).Update("referral_code", code)
	}
}

func migrateCreditsToInt(db *gorm.DB) {
	// Check if migration already done
	var cfg model.SystemConfig
	if db.Where("\"key\" = ?", "credits_migrated_to_int").First(&cfg).Error == nil {
		return // already migrated
	}

	// Check if tables exist and columns are float type
	var colType string
	row := db.Raw("SELECT data_type FROM information_schema.columns WHERE table_name = 'credit_balances' AND column_name = 'balance'").Row()
	if row == nil {
		return // table doesn't exist yet
	}
	if err := row.Scan(&colType); err != nil {
		return // table doesn't exist yet
	}
	if colType == "bigint" {
		// Already int type, just mark as done
		db.Create(&model.SystemConfig{Key: "credits_migrated_to_int", Value: "1"})
		return
	}

	log.Println("Migrating credit columns from float to integer...")

	// Convert float values to int (multiply by 10)
	db.Exec("UPDATE credit_balances SET balance = ROUND(balance * 10)")
	db.Exec("ALTER TABLE credit_balances ALTER COLUMN balance TYPE bigint USING balance::bigint")

	db.Exec("UPDATE credit_transactions SET amount = ROUND(amount * 10), balance_after = ROUND(balance_after * 10)")
	db.Exec("ALTER TABLE credit_transactions ALTER COLUMN amount TYPE bigint USING amount::bigint")
	db.Exec("ALTER TABLE credit_transactions ALTER COLUMN balance_after TYPE bigint USING balance_after::bigint")

	db.Exec("UPDATE domain_group_accesses SET credit_cost = ROUND(credit_cost * 10)")
	db.Exec("ALTER TABLE domain_group_accesses ALTER COLUMN credit_cost TYPE bigint USING credit_cost::bigint")

	db.Exec("UPDATE domains SET credit_cost = ROUND(credit_cost * 10)")
	db.Exec("ALTER TABLE domains ALTER COLUMN credit_cost TYPE bigint USING credit_cost::bigint")

	db.Create(&model.SystemConfig{Key: "credits_migrated_to_int", Value: "1"})
	log.Println("Credit migration complete")
}

func generateReferralCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
