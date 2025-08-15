package handlers

import (
  "io"
  "os"

  "github.com/gin-gonic/gin"
  "github.com/stripe/stripe-go/v74/webhook"
)

func (h *PaymentsHandler) Webhook(c *gin.Context) {
  payload, _ := io.ReadAll(c.Request.Body)
  event, err := webhook.ConstructEvent(payload, c.Request.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
  if err != nil { c.JSON(400, gin.H{"error":"invalid signature"}); return }

  switch event.Type {
  case "payment_intent.succeeded":
      // get metadata.orderId, set order status=paid, send confirmation email
  }
  c.JSON(200, gin.H{"ok": true})
}
