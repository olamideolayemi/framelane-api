package models

import (
	"time"

	"github.com/google/uuid"
)

// Order is the head row for a customer purchase.
// Items live in OrderItem; legacy single-item columns are kept as nullable
// so previously-created orders still load.
type Order struct {
	ID      uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()" json:"id"`
	OrderID string    `gorm:"uniqueIndex;size:40" json:"orderId"`
	UserID  uuid.UUID `gorm:"type:uuid;index" json:"userId"`
	User    User      `gorm:"foreignKey:UserID"`

	// Items / customization
	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items"`

	// Legacy fields kept for backwards compatibility with rows that predate items.
	FrameID  *uuid.UUID `gorm:"type:uuid" json:"frameId,omitempty"`
	Frame    *Frame     `gorm:"foreignKey:FrameID" json:"frame,omitempty"`
	SizeID   *uuid.UUID `gorm:"type:uuid" json:"sizeId,omitempty"`
	Size     *FrameSize `gorm:"foreignKey:SizeID" json:"size,omitempty"`
	ImageURL string     `gorm:"size:600" json:"imageUrl,omitempty"`

	// Address
	AddressID       *uuid.UUID `gorm:"type:uuid" json:"addressId,omitempty"`
	Address         *Address   `gorm:"foreignKey:AddressID" json:"address,omitempty"`
	ShippingSnapshot string    `gorm:"type:text" json:"shippingSnapshot,omitempty"` // JSON copy of the address at time of purchase

	// Money (NGN, integer)
	Subtotal              int    `json:"subtotal"`
	Discount              int    `json:"discount"`
	Shipping              int    `json:"shipping"`
	Total                 int    `json:"total"`
	Currency              string `gorm:"size:8;default:'NGN'" json:"currency"`
	ReferralCreditApplied int    `json:"referralCreditApplied"`

	// Payment
	PaymentProvider string `gorm:"size:40" json:"paymentProvider"`
	PaymentRef      string `gorm:"size:120;index" json:"paymentRef"`
	PaidAt          *time.Time `json:"paidAt"`

	// Promo
	PromoCodeID *uuid.UUID `gorm:"type:uuid" json:"promoCodeId,omitempty"`
	PromoCode   *PromoCode `gorm:"foreignKey:PromoCodeID" json:"promoCode,omitempty"`

	// Fulfillment
	Status              string     `gorm:"size:40;default:'Pending';index" json:"status"`
	Notes               string     `gorm:"size:400" json:"notes"`
	GiftMessage         string     `gorm:"size:500" json:"giftMessage"`
	TrackingCarrier     string     `gorm:"size:80" json:"trackingCarrier"`
	TrackingNumber      string     `gorm:"size:120" json:"trackingNumber"`
	EstimatedDeliveryAt *time.Time `json:"estimatedDeliveryAt"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OrderItem captures one configured frame within an order. Prices are
// snapshotted (UnitPrice/LineTotal) so historical orders remain stable even
// if catalog prices change later.
type OrderItem struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	OrderID      uuid.UUID  `gorm:"type:uuid;index" json:"orderId"`
	FrameID      uuid.UUID  `gorm:"type:uuid" json:"frameId"`
	Frame        Frame      `gorm:"foreignKey:FrameID" json:"frame,omitempty"`
	SizeID       uuid.UUID  `gorm:"type:uuid" json:"sizeId"`
	Size         FrameSize  `gorm:"foreignKey:SizeID" json:"size,omitempty"`
	GlassID      *uuid.UUID `gorm:"type:uuid" json:"glassId,omitempty"`
	Glass        *Glass     `gorm:"foreignKey:GlassID" json:"glass,omitempty"`
	LaminationID *uuid.UUID `gorm:"type:uuid" json:"laminationId,omitempty"`
	Lamination   *Lamination `gorm:"foreignKey:LaminationID" json:"lamination,omitempty"`
	FinishID     *uuid.UUID `gorm:"type:uuid" json:"finishId,omitempty"`
	Finish       *FrameFinish `gorm:"foreignKey:FinishID" json:"finish,omitempty"`
	ImageURL     string     `gorm:"size:600" json:"imageUrl"`
	PreviewURL   string     `gorm:"size:600" json:"previewUrl"`
	Quantity     int        `gorm:"default:1" json:"quantity"`
	UnitPrice    int        `json:"unitPrice"`
	LineTotal    int        `json:"lineTotal"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

const (
	OrderEventCreated     = "created"
	OrderEventPaid        = "paid"
	OrderEventInProduction = "in_production"
	OrderEventShipped     = "shipped"
	OrderEventDelivered   = "delivered"
	OrderEventCancelled   = "cancelled"
	OrderEventNote        = "note"
	OrderEventStatusChange = "status_change"
)

// OrderEvent is an append-only audit log entry for an order.
// Payload is a free-form JSON string (we keep it as text for portability).
type OrderEvent struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	OrderID     uuid.UUID  `gorm:"type:uuid;index" json:"orderId"`
	Type        string     `gorm:"size:40;index" json:"type"`
	ActorUserID *uuid.UUID `gorm:"type:uuid" json:"actorUserId,omitempty"`
	Message     string     `gorm:"size:500" json:"message"`
	Payload     string     `gorm:"type:text" json:"payload"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// OrderResponse is the canonical wire shape returned by list/track/get
// endpoints. Items is preferred over the legacy single-frame fields.
type OrderResponse struct {
	ID      uuid.UUID `json:"id"`
	OrderID string    `json:"orderId"`
	User    struct {
		ID      uuid.UUID `json:"id"`
		Name    string    `json:"name"`
		Phone   string    `json:"phone"`
		Email   string    `json:"email"`
		Address string    `json:"address"`
	} `json:"user"`

	Items []OrderItemResponse `json:"items"`

	// Legacy single-item convenience fields. Populated from the first item
	// when items is non-empty, or from the old order columns otherwise.
	Frame struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"frame"`
	Size struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Price int       `json:"price"`
	} `json:"size"`
	Price    int    `json:"price"`
	ImageURL string `json:"imageUrl"`

	Subtotal              int    `json:"subtotal"`
	Discount              int    `json:"discount"`
	Shipping              int    `json:"shipping"`
	Total                 int    `json:"total"`
	Currency              string `json:"currency"`
	ReferralCreditApplied int    `json:"referralCreditApplied"`

	Status              string     `json:"status"`
	Notes               string     `json:"notes"`
	GiftMessage         string     `json:"giftMessage"`
	TrackingCarrier     string     `json:"trackingCarrier"`
	TrackingNumber      string     `json:"trackingNumber"`
	EstimatedDeliveryAt *time.Time `json:"estimatedDeliveryAt"`
	PaidAt              *time.Time `json:"paidAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type OrderItemResponse struct {
	ID    uuid.UUID `json:"id"`
	Frame struct {
		ID       uuid.UUID `json:"id"`
		Name     string    `json:"name"`
		ImageURL string    `json:"imageUrl"`
	} `json:"frame"`
	Size struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Price int       `json:"price"`
	} `json:"size"`
	Glass *struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"glass,omitempty"`
	Lamination *struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"lamination,omitempty"`
	Finish *struct {
		ID       uuid.UUID `json:"id"`
		Name     string    `json:"name"`
		HexColor string    `json:"hexColor"`
	} `json:"finish,omitempty"`
	ImageURL   string `json:"imageUrl"`
	PreviewURL string `json:"previewUrl"`
	Quantity   int    `json:"quantity"`
	UnitPrice  int    `json:"unitPrice"`
	LineTotal  int    `json:"lineTotal"`
}
