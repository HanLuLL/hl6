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

	if err := db.AutoMigrate(
		&model.User{},
		&model.Domain{},
		&model.Subdomain{},
		&model.DNSRecord{},
		&model.CreditBalance{},
		&model.CreditTransaction{},
		&model.AuditLog{},
	); err != nil {
		log.Fatal("failed to migrate:", err)
	}

	log.Println("Database migrated successfully")

	r := router.Setup(cfg, db)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
