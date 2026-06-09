package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PromoTypePercent = "percent"
	PromoTypeFixed   = "fixed"
)

// PromoCode is a discount coupon. DiscountValue is interpreted by DiscountType:
//   - percent: 1..100 (10 = 10% off)
//   - fixed: amount in NGN to subtract from the subtotal
type PromoCode struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Code            string     `gorm:"size:40;uniqueIndex" json:"code"`
	DiscountType    string     `gorm:"size:20" json:"discountType"`
	DiscountValue   int        `json:"discountValue"`
	MaxRedemptions  int        `gorm:"default:0" json:"maxRedemptions"`
	RedeemedCount   int        `gorm:"default:0" json:"redeemedCount"`
	MinOrderAmount  int        `gorm:"default:0" json:"minOrderAmount"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	Status          string     `gorm:"size:40;default:'active'" json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}
