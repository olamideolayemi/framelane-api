// Package pricing centralizes order-amount math so cart, order creation,
// promo validation, and webhook reconciliation never disagree.
package pricing

import (
	"strings"
	"time"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

// ItemUnitPrice returns the price of one configured frame (FrameSize base
// price plus any glass / lamination / finish modifiers). Quantity is applied
// by callers via LineTotal = UnitPrice * Quantity.
func ItemUnitPrice(size models.FrameSize, glass *models.Glass, lamination *models.Lamination, finish *models.FrameFinish) int {
	price := size.Price
	if glass != nil {
		price += glass.PriceModifier
	}
	if lamination != nil {
		price += lamination.PriceModifier
	}
	if finish != nil {
		price += finish.PriceModifier
	}
	if price < 0 {
		return 0
	}
	return price
}

// LineTotal multiplies unit price by quantity, clamping to >= 0.
func LineTotal(unitPrice, quantity int) int {
	if quantity < 1 {
		quantity = 1
	}
	if unitPrice < 0 {
		unitPrice = 0
	}
	return unitPrice * quantity
}

// ApplyPromo returns the discount amount (positive integer) the promo would
// apply to the given subtotal, plus a boolean indicating eligibility.
func ApplyPromo(p *models.PromoCode, subtotal int) (int, bool) {
	if p == nil {
		return 0, false
	}
	if strings.ToLower(p.Status) != "active" {
		return 0, false
	}
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return 0, false
	}
	if p.MaxRedemptions > 0 && p.RedeemedCount >= p.MaxRedemptions {
		return 0, false
	}
	if p.MinOrderAmount > 0 && subtotal < p.MinOrderAmount {
		return 0, false
	}
	var discount int
	switch strings.ToLower(p.DiscountType) {
	case models.PromoTypePercent:
		discount = subtotal * p.DiscountValue / 100
	case models.PromoTypeFixed:
		discount = p.DiscountValue
	default:
		return 0, false
	}
	if discount > subtotal {
		discount = subtotal
	}
	if discount < 0 {
		discount = 0
	}
	return discount, true
}
