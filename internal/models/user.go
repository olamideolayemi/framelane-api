package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"uniqueIndex;size:255"`
	Password  string    `json:"-"` // hashed
	Name      string    `gorm:"size:120"`
	IsAdmin   bool      `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
