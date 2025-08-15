package handlers

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/payments"
	"gorm.io/gorm"
)

type PaymentsHandler struct{ Stripe *payments.Stripe; DB *gorm.DB }

type intentDTO struct {
  OrderID string `json:"order_id" binding:"required"`
}

func (h *PaymentsHandler) CreateIntent(c *gin.Context) {
  var in intentDTO
  if err := c.ShouldBindJSON(&in); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }

  // lookup order total from DB by in.OrderID
  // TODO: Replace with actual lookup of order total from DB using in.OrderID
  total := int64(10000) // Example: 10000 kobo (₦100.00)

  pi, err := h.Stripe.CreateIntent(c, total, os.Getenv("CURRENCY"), os.Getenv("Email") , map[string]string{"orderId": in.OrderID})
  if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }

  c.JSON(200, gin.H{"client_secret": pi.ClientSecret})
}
