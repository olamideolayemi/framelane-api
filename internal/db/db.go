package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

func Connect(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { log.Fatal(err) }
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		log.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Order{}, &models.Frame{}, &models.FrameSize{}); err != nil {
		log.Fatal(err)
	}
	// Mark legacy users (created before email verification) as verified.
	if err := db.Exec(`
		UPDATE users
		SET email_verified = true
		WHERE email_verified = false
		  AND email_verify_sent_at IS NULL
		  AND COALESCE(email_verify_token_hash, '') = ''
	`).Error; err != nil {
		log.Fatal(err)
	}
	return db
}
