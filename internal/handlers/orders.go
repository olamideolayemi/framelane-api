package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"os"

	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/olamideolayemi/framelane-api/internal/pricing"
	"github.com/olamideolayemi/framelane-api/ws"
	"gorm.io/gorm"
)

type OrdersHandler struct {
	DB    *gorm.DB
	Email *email.Sender
	Hub   *ws.Hub
}

func randID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil // 12 hex chars
}

// POST /v1/orders
//
// Two modes:
//  1. fromCart=true (default): build the order from the user's current cart_items.
//  2. items=[...]: caller passes items inline (used for single-frame quick orders).
//
// Always requires addressId. Computes totals server-side, optionally applies a
// promo code, optionally applies wallet credit. Emits an OrderEvent("created").
func (h *OrdersHandler) Create(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusUnauthorized, "user not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch user", err.Error())
		return
	}
	if !user.IsActive {
		respondError(c, http.StatusForbidden, "account is suspended", nil)
		return
	}

	var in struct {
		AddressID     string          `json:"addressId" binding:"required"`
		PromoCode     string          `json:"promoCode"`
		UseWallet     bool            `json:"useWallet"`
		FromCart      *bool           `json:"fromCart"`
		Items         []cartItemInput `json:"items"`
		Notes         string          `json:"notes"`
		GiftMessage   string          `json:"giftMessage"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.AddressID = strings.TrimSpace(in.AddressID)
	in.PromoCode = strings.TrimSpace(in.PromoCode)
	in.Notes = strings.TrimSpace(in.Notes)
	in.GiftMessage = strings.TrimSpace(in.GiftMessage)

	addressID, err := uuid.Parse(in.AddressID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid addressId", nil)
		return
	}
	var address models.Address
	if err := h.DB.First(&address, "id = ? AND user_id = ?", addressID, uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "address not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch address", err.Error())
		return
	}

	// Build the slice of OrderItems (not yet persisted).
	cartHandler := &CartHandler{DB: h.DB}
	var orderItems []models.OrderItem
	fromCart := true
	if in.FromCart != nil {
		fromCart = *in.FromCart
	}

	if fromCart {
		cartItems, err := cartHandler.listForUser(uid)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to load cart", err.Error())
			return
		}
		if len(cartItems) == 0 {
			respondError(c, http.StatusBadRequest, "cart is empty", nil)
			return
		}
		for _, ci := range cartItems {
			orderItems = append(orderItems, models.OrderItem{
				ID:           uuid.New(),
				FrameID:      ci.FrameID,
				SizeID:       ci.SizeID,
				GlassID:      ci.GlassID,
				LaminationID: ci.LaminationID,
				FinishID:     ci.FinishID,
				ImageURL:     ci.ImageURL,
				PreviewURL:   ci.PreviewURL,
				Quantity:     ci.Quantity,
				UnitPrice:    ci.UnitPrice,
				LineTotal:    ci.LineTotal,
			})
		}
	} else {
		if len(in.Items) == 0 {
			respondError(c, http.StatusBadRequest, "items is required", nil)
			return
		}
		for _, item := range in.Items {
			resolved, status, msg := cartHandler.resolveItem(item)
			if status != 0 {
				respondError(c, status, msg, nil)
				return
			}
			orderItems = append(orderItems, models.OrderItem{
				ID:           uuid.New(),
				FrameID:      resolved.FrameID,
				SizeID:       resolved.SizeID,
				GlassID:      resolved.GlassID,
				LaminationID: resolved.LaminationID,
				FinishID:     resolved.FinishID,
				ImageURL:     resolved.ImageURL,
				PreviewURL:   resolved.PreviewURL,
				Quantity:     resolved.Quantity,
				UnitPrice:    resolved.UnitPrice,
				LineTotal:    resolved.LineTotal,
			})
		}
	}

	// Compute totals.
	subtotal := 0
	for _, it := range orderItems {
		subtotal += it.LineTotal
	}

	// Promo
	var promo *models.PromoCode
	discount := 0
	if in.PromoCode != "" {
		var p models.PromoCode
		if err := h.DB.Where("LOWER(code) = ?", strings.ToLower(in.PromoCode)).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusBadRequest, "invalid promo code", nil)
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to load promo", err.Error())
			return
		}
		d, eligible := pricing.ApplyPromo(&p, subtotal)
		if !eligible {
			respondError(c, http.StatusBadRequest, "promo code is not valid for this order", nil)
			return
		}
		promo = &p
		discount = d
	}

	// Referral / wallet credit
	creditApplied := 0
	if in.UseWallet && user.WalletBalance > 0 {
		cap := subtotal - discount
		if cap < 0 {
			cap = 0
		}
		if user.WalletBalance < cap {
			creditApplied = user.WalletBalance
		} else {
			creditApplied = cap
		}
	}

	shipping := 0 // TODO: derive from address.State; flat 0 for now
	total := subtotal - discount - creditApplied + shipping
	if total < 0 {
		total = 0
	}

	// Build the order.
	rid, err := randID()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate order id", err.Error())
		return
	}
	shipSnap, _ := json.Marshal(address)
	currency := strings.ToUpper(strings.TrimSpace(envDefault("CURRENCY", "NGN")))

	addrIDCopy := address.ID
	order := models.Order{
		ID:                    uuid.New(),
		OrderID:               strings.ToUpper("FL-" + rid),
		UserID:                uid,
		AddressID:             &addrIDCopy,
		ShippingSnapshot:      string(shipSnap),
		Subtotal:              subtotal,
		Discount:              discount,
		Shipping:              shipping,
		Total:                 total,
		Currency:              currency,
		ReferralCreditApplied: creditApplied,
		Status:                OrderStatusPending,
		Notes:                 in.Notes,
		GiftMessage:           in.GiftMessage,
	}
	if promo != nil {
		pid := promo.ID
		order.PromoCodeID = &pid
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		for i := range orderItems {
			orderItems[i].OrderID = order.ID
		}
		if err := tx.Create(&orderItems).Error; err != nil {
			return err
		}
		if creditApplied > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", uid).
				Update("wallet_balance", gorm.Expr("wallet_balance - ?", creditApplied)).Error; err != nil {
				return err
			}
		}
		if promo != nil {
			if err := tx.Model(&models.PromoCode{}).Where("id = ?", promo.ID).
				Update("redeemed_count", gorm.Expr("redeemed_count + 1")).Error; err != nil {
				return err
			}
		}
		// Empty the cart only if we ordered from it.
		if fromCart {
			if err := tx.Where("user_id = ?", uid).Delete(&models.CartItem{}).Error; err != nil {
				return err
			}
		}
		// Emit event.
		payload, _ := json.Marshal(map[string]any{
			"subtotal": subtotal, "discount": discount, "credit": creditApplied, "total": total,
		})
		actor := uid
		ev := models.OrderEvent{
			ID: uuid.New(), OrderID: order.ID, Type: models.OrderEventCreated,
			ActorUserID: &actor, Message: "order created", Payload: string(payload),
		}
		return tx.Create(&ev).Error
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create order", err.Error())
		return
	}

	// Reload with relations for response + email.
	full, _ := h.loadOrder(order.ID.String())

	if h.Email != nil && user.Email != "" {
		data := map[string]string{
			"CustomerName": user.Name,
			"OrderID":      order.OrderID,
			"Frame":        firstFrameName(full),
			"Size":         firstSizeName(full),
			"Price":        fmt.Sprintf("₦%d", subtotal),
			"ImageURL":     firstImageURL(full),
			"Address":      formatAddress(address),
			"Notes":        in.Notes,
			"Total":        fmt.Sprintf("₦%d", total),
			"Status":       OrderStatusPending,
			"Year":         strconv.Itoa(time.Now().Year()),
		}
		if err := SendOrderConfirmation(h.Email, user.Email, data); err != nil {
			log.Printf("Error sending order confirmation email: %v", err)
		}
	}
	h.broadcastOrder("order.created", &order)

	respondSuccess(c, http.StatusCreated, mapOrderResponse(full), nil)
}

// loadOrder fetches an order by id or orderId (FL-…) with all relations.
func (h *OrdersHandler) loadOrder(idOrCode string) (*models.Order, error) {
	id := strings.TrimSpace(idOrCode)
	q := h.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "phone", "email", "address")
		}).
		Preload("Address").
		Preload("Items").
		Preload("Items.Frame").
		Preload("Items.Size").
		Preload("Items.Glass").
		Preload("Items.Lamination").
		Preload("Items.Finish").
		Preload("Frame").
		Preload("Size").
		Preload("PromoCode")

	var order models.Order
	var err error
	if strings.HasPrefix(id, "FL-") {
		err = q.Where("order_id = ?", id).First(&order).Error
	} else {
		parsed, perr := uuid.Parse(id)
		if perr != nil {
			return nil, perr
		}
		err = q.Where("id = ?", parsed).First(&order).Error
	}
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func mapOrderResponse(o *models.Order) models.OrderResponse {
	if o == nil {
		return models.OrderResponse{}
	}
	resp := models.OrderResponse{
		ID:                    o.ID,
		OrderID:               o.OrderID,
		Subtotal:              o.Subtotal,
		Discount:              o.Discount,
		Shipping:              o.Shipping,
		Total:                 o.Total,
		Currency:              o.Currency,
		ReferralCreditApplied: o.ReferralCreditApplied,
		Status:                o.Status,
		Notes:                 o.Notes,
		GiftMessage:           o.GiftMessage,
		TrackingCarrier:       o.TrackingCarrier,
		TrackingNumber:        o.TrackingNumber,
		EstimatedDeliveryAt:   o.EstimatedDeliveryAt,
		PaidAt:                o.PaidAt,
		CreatedAt:             o.CreatedAt,
		UpdatedAt:             o.UpdatedAt,
	}
	resp.User.ID = o.User.ID
	resp.User.Name = o.User.Name
	resp.User.Phone = o.User.Phone
	resp.User.Email = o.User.Email
	resp.User.Address = o.User.Address

	for _, it := range o.Items {
		ir := models.OrderItemResponse{
			ID:         it.ID,
			ImageURL:   it.ImageURL,
			PreviewURL: it.PreviewURL,
			Quantity:   it.Quantity,
			UnitPrice:  it.UnitPrice,
			LineTotal:  it.LineTotal,
		}
		ir.Frame.ID = it.Frame.ID
		ir.Frame.Name = it.Frame.Name
		ir.Frame.ImageURL = it.Frame.ImageURL
		ir.Size.ID = it.Size.ID
		ir.Size.Name = it.Size.Name
		ir.Size.Price = it.Size.Price
		if it.Glass != nil {
			ir.Glass = &struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			}{ID: it.Glass.ID, Name: it.Glass.Name}
		}
		if it.Lamination != nil {
			ir.Lamination = &struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			}{ID: it.Lamination.ID, Name: it.Lamination.Name}
		}
		if it.Finish != nil {
			ir.Finish = &struct {
				ID       uuid.UUID `json:"id"`
				Name     string    `json:"name"`
				HexColor string    `json:"hexColor"`
			}{ID: it.Finish.ID, Name: it.Finish.Name, HexColor: it.Finish.HexColor}
		}
		resp.Items = append(resp.Items, ir)
	}

	// Legacy single-frame convenience fields.
	if len(resp.Items) > 0 {
		first := resp.Items[0]
		resp.Frame.ID = first.Frame.ID
		resp.Frame.Name = first.Frame.Name
		resp.Size.ID = first.Size.ID
		resp.Size.Name = first.Size.Name
		resp.Size.Price = first.Size.Price
		resp.Price = first.Size.Price
		resp.ImageURL = first.ImageURL
	} else {
		if o.Frame != nil {
			resp.Frame.ID = o.Frame.ID
			resp.Frame.Name = o.Frame.Name
		}
		if o.Size != nil {
			resp.Size.ID = o.Size.ID
			resp.Size.Name = o.Size.Name
			resp.Size.Price = o.Size.Price
			resp.Price = o.Size.Price
		}
		resp.ImageURL = o.ImageURL
	}
	return resp
}

func firstFrameName(o *models.Order) string {
	if o != nil && len(o.Items) > 0 {
		return o.Items[0].Frame.Name
	}
	if o != nil && o.Frame != nil {
		return o.Frame.Name
	}
	return ""
}
func firstSizeName(o *models.Order) string {
	if o != nil && len(o.Items) > 0 {
		return o.Items[0].Size.Name
	}
	if o != nil && o.Size != nil {
		return o.Size.Name
	}
	return ""
}
func firstImageURL(o *models.Order) string {
	if o != nil && len(o.Items) > 0 {
		return o.Items[0].ImageURL
	}
	if o != nil {
		return o.ImageURL
	}
	return ""
}
func formatAddress(a models.Address) string {
	parts := []string{a.Line1, a.Line2, a.City, a.State, a.Country}
	out := []string{}
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

// GET /v1/orders
func (h *OrdersHandler) ListMine(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	page, limit := parsePageLimit(c, 10)
	offset := (page - 1) * limit

	var total int64
	if err := h.DB.Model(&models.Order{}).Where("user_id = ?", uid).Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count orders", err.Error())
		return
	}

	var orders []models.Order
	if err := h.DB.
		Where("user_id = ?", uid).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "phone", "email", "address")
		}).
		Preload("Items").
		Preload("Items.Frame").Preload("Items.Size").
		Preload("Items.Glass").Preload("Items.Lamination").Preload("Items.Finish").
		Preload("Frame").Preload("Size").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&orders).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch orders", err.Error())
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	responses := make([]models.OrderResponse, len(orders))
	for i := range orders {
		responses[i] = mapOrderResponse(&orders[i])
	}
	respondSuccess(c, http.StatusOK, responses, gin.H{
		"page": page, "limit": limit, "total": total, "total_pages": totalPages,
	})
}

// GET /v1/orders/:id  (auth, owner-only)
func (h *OrdersHandler) GetMine(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	order, err := h.loadOrder(c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusBadRequest, "invalid order id", err.Error())
		return
	}
	if order.UserID != uid {
		respondError(c, http.StatusForbidden, "not your order", nil)
		return
	}
	respondSuccess(c, http.StatusOK, mapOrderResponse(order), nil)
}

// POST /v1/orders/:id/reorder  -> repopulates the cart with items from this order
func (h *OrdersHandler) Reorder(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	order, err := h.loadOrder(c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusBadRequest, "invalid order id", err.Error())
		return
	}
	if order.UserID != uid {
		respondError(c, http.StatusForbidden, "not your order", nil)
		return
	}
	if len(order.Items) == 0 {
		respondError(c, http.StatusBadRequest, "order has no items to reorder", nil)
		return
	}

	now := time.Now()
	var cartItems []models.CartItem
	for _, it := range order.Items {
		cartItems = append(cartItems, models.CartItem{
			ID: uuid.New(), UserID: uid,
			FrameID: it.FrameID, SizeID: it.SizeID,
			GlassID: it.GlassID, LaminationID: it.LaminationID, FinishID: it.FinishID,
			ImageURL: it.ImageURL, PreviewURL: it.PreviewURL,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, LineTotal: it.LineTotal,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	if err := h.DB.Create(&cartItems).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to repopulate cart", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"added": len(cartItems)}, nil)
}

// GET /v1/admin/orders
func (h *OrdersHandler) ListAll(c *gin.Context) {
	page, limit := parsePageLimit(c, 10)
	offset := (page - 1) * limit

	var total int64
	if err := h.DB.Model(&models.Order{}).Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count orders", err.Error())
		return
	}

	var orders []models.Order
	if err := h.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "phone", "email", "address")
		}).
		Preload("Items").
		Preload("Items.Frame").Preload("Items.Size").
		Preload("Items.Glass").Preload("Items.Lamination").Preload("Items.Finish").
		Preload("Frame").Preload("Size").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&orders).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch orders", err.Error())
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	responses := make([]models.OrderResponse, len(orders))
	for i := range orders {
		responses[i] = mapOrderResponse(&orders[i])
	}
	respondSuccess(c, http.StatusOK, responses, gin.H{
		"page": page, "limit": limit, "total": total, "total_pages": totalPages,
	})
}

// PATCH /v1/admin/orders/:id/status
func (h *OrdersHandler) UpdateStatus(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "order ID is required", nil)
		return
	}
	var in struct {
		Status         string `json:"status" binding:"required"`
		TrackingCarrier string `json:"trackingCarrier"`
		TrackingNumber  string `json:"trackingNumber"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	normalized, ok := normalizeOrderStatus(in.Status)
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid status", gin.H{"allowed": orderStatuses})
		return
	}

	order, err := h.loadOrder(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusBadRequest, "invalid order id", err.Error())
		return
	}

	updates := map[string]any{"status": normalized}
	if strings.TrimSpace(in.TrackingCarrier) != "" {
		updates["tracking_carrier"] = strings.TrimSpace(in.TrackingCarrier)
	}
	if strings.TrimSpace(in.TrackingNumber) != "" {
		updates["tracking_number"] = strings.TrimSpace(in.TrackingNumber)
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(updates).Error; err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]string{"to": normalized})
		ev := models.OrderEvent{
			ID: uuid.New(), OrderID: order.ID, Type: models.OrderEventStatusChange,
			Message: "status changed to " + normalized, Payload: string(payload),
		}
		return tx.Create(&ev).Error
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update status", err.Error())
		return
	}

	if h.Email != nil && order.User.Email != "" {
		data := map[string]string{
			"CustomerName": order.User.Name,
			"OrderID":      order.OrderID,
			"NewStatus":    normalized,
			"OrderLink":    fmt.Sprintf("https://framelane.com/track/%s", order.OrderID),
			"Year":         strconv.Itoa(time.Now().Year()),
		}
		_ = SendOrderStatusUpdate(h.Email, order.User.Email, data)
	}
	h.broadcastOrder("order.updated", order)

	respondSuccess(c, http.StatusOK, gin.H{"status": normalized}, nil)
}

// DELETE /v1/admin/orders/:id
func (h *OrdersHandler) DeleteOrder(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "order ID is required", nil)
		return
	}
	order, err := h.loadOrder(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusBadRequest, "invalid order id", err.Error())
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", order.ID).Delete(&models.OrderItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("order_id = ?", order.ID).Delete(&models.OrderEvent{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Order{}, "id = ?", order.ID).Error
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete order", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{
		"message": fmt.Sprintf("order %s deleted successfully", order.OrderID),
	}, nil)
}

// GET /v1/track/:orderId   (public)
func (h *OrdersHandler) Track(c *gin.Context) {
	id := strings.TrimSpace(c.Param("orderId"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "order ID is required", nil)
		return
	}
	order, err := h.loadOrder(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusBadRequest, "invalid order id", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, mapOrderResponse(order), nil)
}

// GET /v1/orders/:id/events  (auth, owner-only)
func (h *OrdersHandler) ListEvents(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	order, err := h.loadOrder(c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusBadRequest, "invalid order id", err.Error())
		return
	}
	// Admin can see any; owner sees only own.
	isAdmin := false
	if v, exists := c.Get("admin"); exists {
		if b, ok := v.(bool); ok {
			isAdmin = b
		}
	}
	if !isAdmin && order.UserID != uid {
		respondError(c, http.StatusForbidden, "forbidden", nil)
		return
	}
	var events []models.OrderEvent
	if err := h.DB.Where("order_id = ?", order.ID).Order("created_at ASC").Find(&events).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch events", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, events, nil)
}

func parsePageLimit(c *gin.Context, defaultLimit int) (int, int) {
	if defaultLimit < 1 {
		defaultLimit = 10
	}
	page := 1
	limit := defaultLimit
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func envDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// broadcastOrder pushes a JSON event onto the WebSocket hub if one is wired.
func (h *OrdersHandler) broadcastOrder(eventName string, o *models.Order) {
	if h.Hub == nil || o == nil {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"event":   eventName,
		"orderId": o.OrderID,
		"status":  o.Status,
		"id":      o.ID,
	})
	if err != nil {
		return
	}
	h.Hub.Broadcast(msg)
}

func SendOrderConfirmation(sender *email.Sender, customerEmail string, data map[string]string) error {
	subject := "🖼️ Your FrameLane Order"
	htmlBody, err := email.ParseTemplate("order_confirmation.html", data)
	if err != nil {
		return err
	}
	return sender.Send(customerEmail, subject, htmlBody)
}

func SendOrderStatusUpdate(sender *email.Sender, customerEmail string, data map[string]string) error {
	subject := fmt.Sprintf("Update: Your FrameLane order %s", data["OrderID"])
	htmlBody, err := email.ParseTemplate("order_status_update.html", data)
	if err != nil {
		return err
	}
	return sender.Send(customerEmail, subject, htmlBody)
}

func SendOrderShippedNotification(sender *email.Sender, customerEmail string, data map[string]string) error {
	subject := fmt.Sprintf("Your FrameLane order %s has shipped!", data["OrderID"])
	htmlBody, err := email.ParseTemplate("order_shipped.html", data)
	if err != nil {
		return err
	}
	return sender.Send(customerEmail, subject, htmlBody)
}
