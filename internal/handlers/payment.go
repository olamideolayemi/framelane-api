package handlers

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/olamideolayemi/framelane-api/internal/payments"
	"github.com/olamideolayemi/framelane-api/ws"
	"gorm.io/gorm"
)

// PaymentsHandler handles Paystack transaction initialization and webhook
// confirmation. Stripe support is parked behind a build tag in stripe_*.go
// for potential reuse on international expansion.
type PaymentsHandler struct {
	Paystack *payments.Paystack
	DB       *gorm.DB
	Email    *email.Sender
	Hub      *ws.Hub
}

// POST /v1/payments/paystack/initialize
//
// Body: { "orderId": "FL-..." }
// Returns Paystack access_code, reference, and authorization_url. The
// frontend opens Paystack's inline checkout with the access_code.
func (h *PaymentsHandler) Initialize(c *gin.Context) {
	var in struct {
		OrderID  string `json:"orderId" binding:"required"`
		Callback string `json:"callbackUrl"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		respondError(c, http.StatusBadRequest, "orderId is required", nil)
		return
	}
	if h.DB == nil {
		respondError(c, http.StatusInternalServerError, "database is not configured", nil)
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
		respondError(c, http.StatusConflict, "order is not payable", nil)
		return
	}
	if order.Total <= 0 {
		respondError(c, http.StatusBadRequest, "invalid order amount", nil)
		return
	}

	client := h.Paystack
	if client == nil {
		secret := strings.TrimSpace(os.Getenv("PAYSTACK_SECRET"))
		if secret == "" {
			respondError(c, http.StatusInternalServerError, "paystack is not configured", nil)
			return
		}
		client = payments.NewPaystack(secret)
	}

	currency := strings.ToUpper(envDefault("CURRENCY", "NGN"))
	resp, err := client.Initialize(payments.InitializeRequest{
		Email:    order.User.Email,
		Amount:   order.Total * 100, // kobo
		Currency: currency,
		Callback: strings.TrimSpace(in.Callback),
		Metadata: map[string]string{"orderId": order.OrderID, "userId": order.UserID.String()},
	})
	if err != nil {
		respondError(c, http.StatusBadGateway, "failed to initialize payment", err.Error())
		return
	}

	// Save the reference so the webhook + verify flow can correlate.
	_ = h.DB.Model(&order).Updates(map[string]any{
		"payment_provider": "paystack",
		"payment_ref":      resp.Data.Reference,
	}).Error

	respondSuccess(c, http.StatusOK, gin.H{
		"reference":        resp.Data.Reference,
		"accessCode":       resp.Data.AccessCode,
		"authorizationUrl": resp.Data.AuthorizationURL,
		"publicKey":        os.Getenv("PAYSTACK_PUBLIC"),
	}, nil)
}

// POST /v1/payments/paystack/verify
//
// Optional belt-and-braces endpoint for the frontend to confirm a transaction
// completed (in addition to the webhook).
func (h *PaymentsHandler) Verify(c *gin.Context) {
	ref := strings.TrimSpace(c.Query("reference"))
	if ref == "" {
		respondError(c, http.StatusBadRequest, "reference is required", nil)
		return
	}
	client := h.Paystack
	if client == nil {
		client = payments.NewPaystack(os.Getenv("PAYSTACK_SECRET"))
	}
	v, err := client.Verify(ref)
	if err != nil {
		respondError(c, http.StatusBadGateway, "failed to verify payment", err.Error())
		return
	}
	if strings.ToLower(v.Data.Status) != "success" {
		respondSuccess(c, http.StatusOK, gin.H{"status": v.Data.Status}, nil)
		return
	}
	if err := h.markOrderPaid(v.Data.Reference, v.Data.Metadata, v.Data.Amount); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to mark order paid", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"status": "success"}, nil)
}

// POST /v1/payments/paystack/webhook
//
// Paystack signs the raw request body using HMAC-SHA512 with the SECRET key.
// Header: x-paystack-signature.
func (h *PaymentsHandler) Webhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	secret := strings.TrimSpace(os.Getenv("PAYSTACK_SECRET"))
	if secret == "" {
		respondError(c, http.StatusInternalServerError, "paystack secret not configured", nil)
		return
	}
	sig := c.GetHeader("x-paystack-signature")
	if sig == "" || !verifyPaystackSignature(secret, body, sig) {
		respondError(c, http.StatusBadRequest, "invalid signature", nil)
		return
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string            `json:"reference"`
			Status    string            `json:"status"`
			Amount    int               `json:"amount"` // kobo
			Currency  string            `json:"currency"`
			Metadata  map[string]string `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(c, http.StatusBadRequest, "invalid event payload", err.Error())
		return
	}

	switch event.Event {
	case "charge.success":
		if err := h.markOrderPaid(event.Data.Reference, event.Data.Metadata, event.Data.Amount); err != nil {
			log.Printf("paystack webhook: %v", err)
		}
	}
	respondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
}

func verifyPaystackSignature(secret string, body []byte, sig string) bool {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// markOrderPaid flips the order to Processing, emits an event, broadcasts WS,
// sends the status email, credits any referral reward, and is idempotent
// (re-running it on an already-paid order is a no-op).
func (h *PaymentsHandler) markOrderPaid(reference string, metadata map[string]string, amountKobo int) error {
	orderID := strings.TrimSpace(metadata["orderId"])
	if orderID == "" {
		orderID = strings.TrimSpace(metadata["order_id"])
	}

	var order models.Order
	q := h.DB.Preload("User")
	if orderID != "" {
		q = q.Where("order_id = ?", orderID)
	} else {
		q = q.Where("payment_ref = ?", reference)
	}
	if err := q.First(&order).Error; err != nil {
		return fmt.Errorf("order lookup: %w", err)
	}
	if isTerminalOrderStatus(order.Status) || strings.EqualFold(order.Status, OrderStatusProcessing) {
		return nil // idempotent
	}
	if amountKobo > 0 && amountKobo != order.Total*100 {
		// Amount mismatch is suspicious; log but still record payment to avoid
		// double-charging the customer. Admin should reconcile manually.
		log.Printf("paystack amount mismatch: order=%s expected=%d got=%d", order.OrderID, order.Total*100, amountKobo)
	}

	now := time.Now()
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{
			"status":           OrderStatusProcessing,
			"payment_ref":      reference,
			"payment_provider": "paystack",
			"paid_at":          &now,
		}).Error; err != nil {
			return err
		}
		ev := models.OrderEvent{
			ID: uuid.New(), OrderID: order.ID, Type: models.OrderEventPaid,
			Message: "payment received", Payload: fmt.Sprintf(`{"reference":%q,"amount":%d}`, reference, amountKobo),
		}
		if err := tx.Create(&ev).Error; err != nil {
			return err
		}
		return creditReferralIfFirstPaid(tx, order.UserID)
	})
	if err != nil {
		return err
	}

	if h.Email != nil && order.User.Email != "" {
		data := map[string]string{
			"CustomerName": order.User.Name,
			"OrderID":      order.OrderID,
			"NewStatus":    OrderStatusProcessing,
			"OrderLink":    fmt.Sprintf("https://framelane.com/track/%s", order.OrderID),
			"Year":         strconv.Itoa(time.Now().Year()),
		}
		_ = SendOrderStatusUpdate(h.Email, order.User.Email, data)
	}
	if h.Hub != nil {
		msg, _ := json.Marshal(map[string]any{
			"event": "order.paid", "orderId": order.OrderID, "id": order.ID,
		})
		h.Hub.Broadcast(msg)
	}
	return nil
}

// creditReferralIfFirstPaid credits the referrer's wallet on the referred
// user's first paid order. No-op if user wasn't referred or already credited.
func creditReferralIfFirstPaid(tx *gorm.DB, referredUserID uuid.UUID) error {
	var r models.Referral
	if err := tx.Where("referred_user_id = ? AND status = ?", referredUserID, models.ReferralStatusPending).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if r.RewardAmount <= 0 {
		return nil
	}
	now := time.Now()
	if err := tx.Model(&models.Referral{}).Where("id = ?", r.ID).Updates(map[string]any{
		"status":      models.ReferralStatusCredited,
		"credited_at": &now,
	}).Error; err != nil {
		return err
	}
	// Credit referrer's wallet.
	if err := tx.Model(&models.User{}).Where("id = ?", r.ReferrerUserID).
		Update("wallet_balance", gorm.Expr("wallet_balance + ?", r.RewardAmount)).Error; err != nil {
		return err
	}
	return nil
}
