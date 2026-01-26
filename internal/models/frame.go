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
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"size:80;uniqueIndex" json:"name"` // e.g., "Wooden Frame"
	Description string    `gorm:"size:500" json:"description"`     // Description of the frame type
	Status      string    `gorm:"size:40;default:'available'" json:"status"`
	ImageURL    string    `gorm:"size:255" json:"image_url"` // URL to the frame type image
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// FrameResponse can be used in APIs to safely return frame info
type FrameResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ImageURL    string    `json:"image_url"`
}
