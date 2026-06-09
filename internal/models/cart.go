package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CartItem is a server-persisted entry in a user's cart prior to checkout.
// Each row represents one configured frame the user intends to order.
type CartItem struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;index" json:"userId"`
	FrameID      uuid.UUID  `gorm:"type:uuid" json:"frameId"`
	Frame        Frame      `gorm:"foreignKey:FrameID" json:"frame,omitempty"`
	SizeID       uuid.UUID  `gorm:"type:uuid" json:"sizeId"`
	Size         FrameSize  `gorm:"foreignKey:SizeID" json:"size,omitempty"`
	GlassID      *uuid.UUID `gorm:"type:uuid" json:"glassId,omitempty"`
	Glass        *Glass     `gorm:"foreignKey:GlassID" json:"glass,omitempty"`
	LaminationID *uuid.UUID `gorm:"type:uuid" json:"laminationId,omitempty"`
	Lamination   *Lamination `gorm:"foreignKey:LaminationID" json:"lamination,omitempty"`
	FinishID     *uuid.UUID  `gorm:"type:uuid" json:"finishId,omitempty"`
	Finish       *FrameFinish `gorm:"foreignKey:FinishID" json:"finish,omitempty"`
	ImageURL     string      `gorm:"size:600" json:"imageUrl"`
	PreviewURL   string      `gorm:"size:600" json:"previewUrl"`
	GiftMessage  string      `gorm:"size:500" json:"giftMessage"`
	Quantity     int         `gorm:"default:1" json:"quantity"`
	UnitPrice    int         `json:"unitPrice"`
	LineTotal    int         `json:"lineTotal"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
