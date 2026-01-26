package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
	"gorm.io/gorm"
)

func (h *PaymentsHandler) Webhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}

	secret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if secret == "" {
		respondError(c, http.StatusInternalServerError, "stripe webhook secret not configured", nil)
		return
	}
	if h.DB == nil {
		respondError(c, http.StatusInternalServerError, "database is not configured", nil)
		return
	}

	event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), secret)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid signature", err.Error())
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			respondError(c, http.StatusBadRequest, "invalid event payload", err.Error())
			return
		}

		orderID := strings.TrimSpace(pi.Metadata["orderId"])
		if orderID == "" {
			orderID = strings.TrimSpace(pi.Metadata["order_id"])
		}
		if orderID == "" {
			respondError(c, http.StatusBadRequest, "missing orderId metadata", nil)
			return
		}

		var order models.Order
		if err := h.DB.Preload("User").Where("order_id = ?", orderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusNotFound, "order not found", nil)
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
			return
		}
		if isTerminalOrderStatus(order.Status) {
			respondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
			return
		}

		if err := h.DB.Model(&order).Update("status", OrderStatusProcessing).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "failed to update order status", err.Error())
			return
		}

		if h.Email != nil && order.User.Email != "" {
			data := map[string]string{
				"CustomerName": order.User.Name,
				"OrderID":      order.OrderID,
				"NewStatus":    OrderStatusProcessing,
				"OrderLink":    fmt.Sprintf("https://framelane.com/track/%s", order.OrderID),
				"Year":         fmt.Sprintf("%d", time.Now().Year()),
			}
			_ = SendOrderStatusUpdate(h.Email, order.User.Email, data)
		}
	}

	respondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
}
