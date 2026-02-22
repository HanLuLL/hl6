package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"hl6-server/internal/config"
	"hl6-server/internal/model"
	"hl6-server/internal/router"
)

func main() {
	godotenv.Load("../.env")

	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	migrateCreditsToInt(db)

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
	); err != nil {
		log.Fatal("failed to migrate:", err)
	}

	log.Println("Database migrated successfully")
	seedDefaults(db)

	r := router.Setup(cfg, db)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("failed to start server:", err)
	}
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
