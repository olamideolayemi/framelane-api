package handlers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
	"gorm.io/gorm"
)

type OrdersHandler struct {
	DB    *gorm.DB
	Email *email.Sender
}

func randID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil // 12 hex chars
}

// POST /v1/orders (guest or logged-in)
func (h *OrdersHandler) Create(c *gin.Context) {
	uidVal, exists := c.Get("uid")
	if !exists {
		respondError(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	uidStr, ok := uidVal.(string)
	if !ok {
		respondError(c, http.StatusInternalServerError, "invalid user context", nil)
		return
	}

	uid, err := uuid.Parse(uidStr)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid user ID", nil)
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
		Address  string `json:"address" binding:"required"`
		FrameID  string `json:"frameId" binding:"required"`
		SizeID   string `json:"sizeId" binding:"required"`
		Notes    string `json:"notes" binding:"omitempty"`
		ImageURL string `json:"imageUrl" binding:"required"`
	}

	// Bind JSON
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request data", err.Error())
		return
	}
	in.Address = strings.TrimSpace(in.Address)
	in.Notes = strings.TrimSpace(in.Notes)
	in.ImageURL = strings.TrimSpace(in.ImageURL)
	in.FrameID = strings.TrimSpace(in.FrameID)
	in.SizeID = strings.TrimSpace(in.SizeID)
	if in.Address == "" {
		respondError(c, http.StatusBadRequest, "address is required", nil)
		return
	}
	if in.ImageURL == "" {
		respondError(c, http.StatusBadRequest, "image URL is required", nil)
		return
	}

	// Parse Frame ID
	frameID, err := uuid.Parse(in.FrameID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frame ID", nil)
		return
	}

	var frame models.Frame
	if err := h.DB.First(&frame, "id = ?", frameID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "frame not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch frame", err.Error())
		return
	}
	if strings.ToLower(frame.Status) != "available" {
		respondError(c, http.StatusConflict, "frame type is not available", nil)
		return
	}

	// Parse FrameSize ID
	sizeID, err := uuid.Parse(in.SizeID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frame size ID", nil)
		return
	}

	var size models.FrameSize
	if err := h.DB.First(&size, "id = ?", sizeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "frame size not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch frame size", err.Error())
		return
	}
	if strings.ToLower(size.Status) != "available" {
		respondError(c, http.StatusConflict, "frame size is not available", nil)
		return
	}

	// Create order
	rid, err := randID()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate order ID", err.Error())
		return
	}

	order := models.Order{
		OrderID:  strings.ToUpper("FL-" + rid),
		UserID:   uid,
		FrameID:  frame.ID,
		Frame:    frame,
		SizeID:   size.ID,
		Size:     size,
		ImageURL: in.ImageURL,
		Notes:    in.Notes,
		Status:   OrderStatusPending,
	}

	if err := h.DB.Create(&order).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "could not create order", err.Error())
		return
	}

	// Send confirmation email
	if h.Email != nil && user.Email != "" {
		data := map[string]string{
			"CustomerName": user.Name,
			"OrderID":      order.OrderID,
			"Frame":        frame.Name,
			"Size":         size.Name,
			"Price":        fmt.Sprintf("₦%d", size.Price),
			"ImageURL":     order.ImageURL,
			"Address":      in.Address,
			"Notes":        in.Notes,
			"Total":        "₦0",
			"Status":       OrderStatusPending,
			"Year":         fmt.Sprintf("%d", time.Now().Year()),
		}
		if err := SendOrderConfirmation(h.Email, user.Email, data); err != nil {
			log.Printf("Error sending order confirmation email: %v", err)
		}
	}

	respondSuccess(c, http.StatusCreated, gin.H{
		"message":   "order placed successfully",
		"orderId":   order.OrderID,
		"id":        order.ID,
		"frame":     frame.Name,
		"size":      size.Name,
		"price":     size.Price,
		"imageUrl":  order.ImageURL,
		"notes":     order.Notes,
		"status":    order.Status,
		"createdAt": order.CreatedAt,
		"updatedAt": order.UpdatedAt,
	}, nil)
}

// GET /v1/orders (auth) -> list own
func (h *OrdersHandler) ListMine(c *gin.Context) {
	uidVal, exists := c.Get("uid")
	if !exists {
		respondError(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	uidStr, ok := uidVal.(string)
	if !ok {
		respondError(c, http.StatusInternalServerError, "invalid user context", nil)
		return
	}

	uid, err := uuid.Parse(uidStr)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid user ID", nil)
		return
	}

	page, limit := parsePageLimit(c, 10)

	offset := (page - 1) * limit

	var total int64
	if err := h.DB.Model(&models.Order{}).
		Where("user_id = ?", uid).
		Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count orders", err.Error())
		return
	}

	var orders []models.Order
	if err := h.DB.
		Where("user_id = ?", uid).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "phone", "email", "address")
		}).
		Preload("Frame").
		Preload("Size").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch orders", err.Error())
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Map to safe response
	responses := make([]models.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = models.OrderResponse{
			ID:      o.ID,
			OrderID: o.OrderID,
			User: struct {
				ID      uuid.UUID `json:"id"`
				Name    string    `json:"name"`
				Phone   string    `json:"phone"`
				Email   string    `json:"email"`
				Address string    `json:"address"`
			}{
				ID:      o.User.ID,
				Name:    o.User.Name,
				Phone:   o.User.Phone,
				Email:   o.User.Email,
				Address: o.User.Address,
			},
			Frame: struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			}{
				ID:   o.Frame.ID,
				Name: o.Frame.Name,
			},
			Size: struct {
				ID    uuid.UUID `json:"id"`
				Name  string    `json:"name"`
				Price int       `json:"price"`
			}{
				ID:    o.Size.ID,
				Name:  o.Size.Name,
				Price: o.Size.Price,
			},
			Price:     o.Size.Price,
			ImageURL:  o.ImageURL,
			Status:    o.Status,
			Notes:     o.Notes,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}

	}

	respondSuccess(c, http.StatusOK, responses, gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	})
}

// GET /v1/admin/orders (admin)
func (h *OrdersHandler) ListAll(c *gin.Context) {
	page, limit := parsePageLimit(c, 10)

	offset := (page - 1) * limit

	// Count total records (for pagination)
	var total int64
	if err := h.DB.Model(&models.Order{}).Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count orders", err.Error())
		return
	}

	// Fetch paginated records
	var orders []models.Order
	if err := h.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "phone", "email", "address")
		}).
		Preload("Frame").
		Preload("Size").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch orders", err.Error())
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Build response
	responses := make([]models.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = models.OrderResponse{
			ID:      o.ID,
			OrderID: o.OrderID,
			User: struct {
				ID      uuid.UUID `json:"id"`
				Name    string    `json:"name"`
				Phone   string    `json:"phone"`
				Email   string    `json:"email"`
				Address string    `json:"address"`
			}{
				ID:      o.User.ID,
				Name:    o.User.Name,
				Phone:   o.User.Phone,
				Email:   o.User.Email,
				Address: o.User.Address,
			},
			Frame: struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			}{
				ID:   o.Frame.ID,
				Name: o.Frame.Name,
			},
			Size: struct {
				ID    uuid.UUID `json:"id"`
				Name  string    `json:"name"`
				Price int       `json:"price"`
			}{
				ID:    o.Size.ID,
				Name:  o.Size.Name,
				Price: o.Size.Price,
			},
			Price:     o.Size.Price,
			ImageURL:  o.ImageURL,
			Status:    o.Status,
			Notes:     o.Notes,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}
	}

	respondSuccess(c, http.StatusOK, responses, gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	})
}

// PATCH /v1/admin/orders/:id/status (admin)
func (h *OrdersHandler) UpdateStatus(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "order ID is required", nil)
		return
	}
	var in struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	normalizedStatus, ok := normalizeOrderStatus(in.Status)
	if !ok {
		respondError(c, http.StatusBadRequest, "invalid status", gin.H{"allowed": orderStatuses})
		return
	}

	var order models.Order

	// Decide how to query based on the format of `id`
	if strings.HasPrefix(id, "FL-") {
		if err := h.DB.Where("order_id = ?", id).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusNotFound, "order not found", nil)
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
			return
		}
	} else {
		orderID, err := uuid.Parse(id)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid order ID", nil)
			return
		}
		if err := h.DB.Where("id = ?", orderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusNotFound, "order not found", nil)
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
			return
		}
	}

	if err := h.DB.Model(&order).Update("status", normalizedStatus).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update status", err.Error())
		return
	}

	// Fetch the user linked to the order
	var user models.User
	if err := h.DB.First(&user, "id = ?", order.UserID).Error; err == nil {
		if h.Email != nil && user.Email != "" {
			data := map[string]string{
				"CustomerName": user.Name,
				"OrderID":      order.OrderID,
				"NewStatus":    normalizedStatus,
				"OrderLink":    fmt.Sprintf("https://framelane.com/track/%s", order.OrderID),
				"Year":         fmt.Sprintf("%d", time.Now().Year()),
			}
			_ = SendOrderStatusUpdate(h.Email, user.Email, data)

		}
	}

	respondSuccess(c, http.StatusOK, gin.H{"status": normalizedStatus}, nil)
}

// DELETE /v1/admin/orders/:id (admin)
func (h *OrdersHandler) DeleteOrder(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "order ID is required", nil)
		return
	}

	var order models.Order
	// Check if order exists
	if strings.HasPrefix(id, "FL-") {
		if err := h.DB.Where("order_id = ?", id).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusNotFound, "order not found", nil)
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
			return
		}
	} else {
		orderID, err := uuid.Parse(id)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid order ID", nil)
			return
		}
		if err := h.DB.Where("id = ?", orderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respondError(c, http.StatusNotFound, "order not found", nil)
				return
			}
			respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
			return
		}
	}

	// Delete the order
	if err := h.DB.Delete(&order).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete order", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{
		"message": fmt.Sprintf("order %s deleted successfully", order.OrderID),
	}, nil)
}

// GET /v1/track/:orderId  (public)
func (h *OrdersHandler) Track(c *gin.Context) {
	oid := strings.TrimSpace(c.Param("orderId"))
	if oid == "" {
		respondError(c, http.StatusBadRequest, "order ID is required", nil)
		return
	}

	var order models.Order
	if err := h.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "phone", "email", "address")
		}).
		Preload("Frame").
		Preload("Size").
		Where("order_id = ?", oid).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "order not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
		return
	}

	// Map into the same structure as ListAll/ListMine
	response := models.OrderResponse{
		ID:      order.ID,
		OrderID: order.OrderID,
		User: struct {
			ID      uuid.UUID `json:"id"`
			Name    string    `json:"name"`
			Phone   string    `json:"phone"`
			Email   string    `json:"email"`
			Address string    `json:"address"`
		}{
			ID:      order.User.ID,
			Name:    order.User.Name,
			Phone:   order.User.Phone,
			Email:   order.User.Email,
			Address: order.User.Address,
		},
		Frame: struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		}{
			ID:   order.Frame.ID,
			Name: order.Frame.Name,
		},
		Size: struct {
			ID    uuid.UUID `json:"id"`
			Name  string    `json:"name"`
			Price int       `json:"price"`
		}{
			ID:    order.Size.ID,
			Name:  order.Size.Name,
			Price: order.Size.Price,
		},
		Price:     order.Size.Price,
		ImageURL:  order.ImageURL,
		Status:    order.Status,
		Notes:     order.Notes,
		CreatedAt: order.CreatedAt,
		UpdatedAt: order.UpdatedAt,
	}

	respondSuccess(c, http.StatusOK, response, nil)
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

// In your order placement route
// hub.broadcast <- []byte(`{"event":"order_placed","orderId":"123"}`)

// // In your order update route
// hub.broadcast <- []byte(`{"event":"order_updated","orderId":"123","status":"shipped"}`)
