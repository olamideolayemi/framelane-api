package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Address is an entry in a user's address book. Orders reference one of these.
type Address struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;index" json:"userId"`
	Label      string         `gorm:"size:80" json:"label"`
	Recipient  string         `gorm:"size:120" json:"recipient"`
	Phone      string         `gorm:"size:40" json:"phone"`
	Line1      string         `gorm:"size:255" json:"line1"`
	Line2      string         `gorm:"size:255" json:"line2"`
	City       string         `gorm:"size:120" json:"city"`
	State      string         `gorm:"size:120" json:"state"`
	PostalCode string         `gorm:"size:40" json:"postalCode"`
	Country    string         `gorm:"size:80;default:'Nigeria'" json:"country"`
	IsDefault  bool           `gorm:"default:false" json:"isDefault"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
