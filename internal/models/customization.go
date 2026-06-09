package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Glass represents a glass option a customer can choose for their frame.
// PriceModifier is added to the line total (NGN, integer).
type Glass struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name          string         `gorm:"size:80;uniqueIndex" json:"name"`
	Description   string         `gorm:"size:500" json:"description"`
	PriceModifier int            `gorm:"default:0" json:"priceModifier"`
	Status        string         `gorm:"size:40;default:'available'" json:"status"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Lamination represents a lamination/finish coating option.
type Lamination struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name          string         `gorm:"size:80;uniqueIndex" json:"name"`
	Description   string         `gorm:"size:500" json:"description"`
	PriceModifier int            `gorm:"default:0" json:"priceModifier"`
	Status        string         `gorm:"size:40;default:'available'" json:"status"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// FrameFinish is a color/material variant tied to a specific Frame.
// e.g. Frame "Modern" might have finishes: Gold, Black, Walnut, White.
type FrameFinish struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	FrameID       uuid.UUID      `gorm:"type:uuid;index" json:"frameId"`
	Name          string         `gorm:"size:80" json:"name"`
	HexColor      string         `gorm:"size:16" json:"hexColor"`
	ImageURL      string         `gorm:"size:600" json:"imageUrl"`
	PriceModifier int            `gorm:"default:0" json:"priceModifier"`
	Status        string         `gorm:"size:40;default:'available'" json:"status"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
