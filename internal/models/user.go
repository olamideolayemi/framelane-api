package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Email     string    `gorm:"uniqueIndex;size:255"`
	Password  string    `json:"-"` // hashed
	Name      string    `gorm:"size:120"`
	Phone     string    `gorm:"size:40"`
    Address   string    `gorm:"size:400"`
	IsAdmin   bool      `gorm:"default:false"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Hook to set UUID before create
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
