package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/olamideolayemi/framelane-api/internal/payments"
	"gorm.io/gorm"
)

type PaymentsHandler struct {
	Stripe *payments.Stripe
	DB     *gorm.DB
	Email  *email.Sender
}

type intentDTO struct {
	OrderID string `json:"order_id" binding:"required"`
}

func (h *PaymentsHandler) CreateIntent(c *gin.Context) {
	var in intentDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		respondError(c, http.StatusBadRequest, "order_id is required", nil)
		return
	}
	if h.DB == nil {
		respondError(c, http.StatusInternalServerError, "database is not configured", nil)
		return
	}

	var order models.Order
	if err := h.DB.Preload("Size").Preload("User").Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
		return
	}
	if isTerminalOrderStatus(order.Status) {
		respondError(c, http.StatusConflict, "order is not payable", nil)
		return
	}

	total := int64(order.Size.Price)
	if total <= 0 {
		respondError(c, http.StatusBadRequest, "invalid order amount", nil)
		return
	}

	currency := strings.ToLower(strings.TrimSpace(os.Getenv("CURRENCY")))
	if currency == "" {
		currency = "ngn"
	}

	stripeClient := h.Stripe
	if stripeClient == nil {
		secret := strings.TrimSpace(os.Getenv("STRIPE_SECRET"))
		if secret == "" {
			respondError(c, http.StatusInternalServerError, "stripe is not configured", nil)
			return
		}
		stripeClient = payments.NewStripe(secret)
	}

	pi, err := stripeClient.CreateIntent(c, total, currency, strings.TrimSpace(order.User.Email), map[string]string{"orderId": order.OrderID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create payment intent", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"client_secret": pi.ClientSecret}, nil)
}
