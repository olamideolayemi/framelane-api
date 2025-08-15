package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type FrameSize struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name      string         `gorm:"unique;not null" json:"name"`
	Price     int            `gorm:"not null" json:"price"`
	Status    string         `gorm:"default:'available'" json:"status"` // "available" or "out_of_stock"
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Frame struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:80;uniqueIndex" json:"name"` // e.g., "Wooden Frame"
	Status    string    `gorm:"size:40;default:'available'" json:"status"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FrameResponse can be used in APIs to safely return frame info
type FrameResponse struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Status string    `json:"status"`
}
