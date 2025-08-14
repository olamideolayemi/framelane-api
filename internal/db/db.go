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
	if err := db.AutoMigrate(&models.User{}, &models.Order{}); err != nil {
		log.Fatal(err)
	}
	return db
}
