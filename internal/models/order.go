package models

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	OrderID   string    `gorm:"uniqueIndex;size:40" json:"orderId"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"userId"`
	User      User      `gorm:"foreignKey:UserID"`
	FrameID   uuid.UUID `gorm:"type:uuid" json:"frameId"`
	Frame     Frame     `gorm:"foreignKey:FrameID"`
	SizeID    uuid.UUID `gorm:"type:uuid" json:"sizeId"`
	Size      FrameSize `gorm:"foreignKey:SizeID"`
	ImageURL  string    `gorm:"size:600" json:"imageUrl"`
	Status    string    `gorm:"size:40;default:'Pending'" json:"status"`
	Notes     string    `gorm:"size:400" json:"notes"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderResponse struct {
	ID      uuid.UUID `json:"id"`
	OrderID string    `json:"orderId"`
	User    struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"user"`
	Frame struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"frame"`
	Size struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Price int       `json:"price"`
	} `json:"size"`
	// Frame     string    `json:"frame"`
	// Size      string    `json:"size"`
	Price     int       `json:"price"`
	ImageURL  string    `json:"imageUrl"`
	Status    string    `json:"status"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
