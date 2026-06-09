package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusHidden   = "hidden"
)

// Review is a customer rating/comment tied to a specific OrderItem (and
// transitively a Frame). Frontend lists Approved reviews on frame tiles.
type Review struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;index" json:"userId"`
	User        User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OrderItemID uuid.UUID  `gorm:"type:uuid;index" json:"orderItemId"`
	FrameID     uuid.UUID  `gorm:"type:uuid;index" json:"frameId"`
	Rating      int        `json:"rating"`
	Body        string     `gorm:"size:1000" json:"body"`
	PhotoURL    string     `gorm:"size:600" json:"photoUrl"`
	Status      string     `gorm:"size:20;default:'pending'" json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
