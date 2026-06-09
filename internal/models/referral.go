package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ReferralStatusPending  = "pending"
	ReferralStatusCredited = "credited"
)

// Referral tracks a referrer→referred relationship. RewardAmount is credited
// to the referrer's User.WalletBalance once the referred user's first paid
// order is recorded.
type Referral struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ReferrerUserID uuid.UUID `gorm:"type:uuid;index" json:"referrerUserId"`
	ReferredUserID uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"referredUserId"`
	Code           string    `gorm:"size:40" json:"code"`
	RewardAmount   int       `json:"rewardAmount"`
	Status         string    `gorm:"size:20;default:'pending'" json:"status"`
	CreditedAt     *time.Time `json:"creditedAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
