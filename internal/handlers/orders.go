package handlers

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	// "net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
)

type OrdersHandler struct {
	DB    *gorm.DB
	Email *email.Sender
}

func randID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b) // 12 hex chars
}

// POST /v1/orders (guest or logged-in)
func (h *OrdersHandler) Create(c *gin.Context) {
	uidVal, exists := c.Get("uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be logged in to place an order"})
		return
	}

	uid, err := uuid.Parse(uidVal.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Parse Frame ID
	frameID, err := uuid.Parse(in.FrameID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid frame ID"})
		return
	}

	var frame models.Frame
	if err := h.DB.First(&frame, "id = ?", frameID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Frame not found"})
		return
	}

	// Parse FrameSize ID
	sizeID, err := uuid.Parse(in.SizeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid frame size ID"})
		return
	}

	var size models.FrameSize
	if err := h.DB.First(&size, "id = ?", sizeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Frame size not found"})
		return
	}

	// Create order
	order := models.Order{
		OrderID:  strings.ToUpper("FL-" + randID()),
		UserID:   uid,
		FrameID:  frame.ID,
		Frame:    frame,
		SizeID:   size.ID,
		Size:     size,
		ImageURL: in.ImageURL,
		Notes:    in.Notes,
	}

	if err := h.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create order", "details": err.Error()})
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
			"Status":       "Pending",
			"Year":         fmt.Sprintf("%d", time.Now().Year()),
		}
		if err := SendOrderConfirmation(h.Email, user.Email, data); err != nil {
			log.Printf("Error sending order confirmation email: %v", err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Order placed successfully",
		"orderId":   order.OrderID,
		"id":        order.ID,
		"frame":     frame.Name,
		"size":      size.Name,
		"price":     size.Price,
		"imageUrl":  order.ImageURL,
		"notes":     order.Notes,
		"createdAt": order.CreatedAt,
		"updatedAt": order.UpdatedAt,
	})
}

// GET /v1/orders (auth) -> list own
func (h *OrdersHandler) ListMine(c *gin.Context) {
	uidVal, exists := c.Get("uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be logged in"})
		return
	}

	uid, err := uuid.Parse(uidVal.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	page := 1
	limit := 10

	if p, ok := c.GetQuery("page"); ok {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}

	if l, ok := c.GetQuery("limit"); ok {
		fmt.Sscanf(l, "%d", &limit)
		if limit < 1 {
			limit = 10
		}
	}

	offset := (page - 1) * limit

	var total int64
	h.DB.Model(&models.Order{}).
		Where("user_id = ?", uid).
		Count(&total)

	var orders []models.Order
	if err := h.DB.
		Where("user_id = ?", uid).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name")
		}).
		Preload("Frame").
		Preload("Size").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Map to safe response
	responses := make([]models.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = models.OrderResponse{
			ID:      o.ID,
			OrderID: o.OrderID,
			User: struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			}{
				ID:   o.User.ID,
				Name: o.User.Name,
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
			ImageURL:  o.ImageURL,
			Status:    o.Status,
			Notes:     o.Notes,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}

	}

	c.JSON(http.StatusOK, gin.H{
		"page":   page,
		"limit":  limit,
		"total":  total,
		"orders": responses,
	})
}

// GET /v1/admin/orders (admin)
func (h *OrdersHandler) ListAll(c *gin.Context) {
	page := 1
	limit := 10

	if p, ok := c.GetQuery("page"); ok {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}

	if l, ok := c.GetQuery("limit"); ok {
		fmt.Sscanf(l, "%d", &limit)
		if limit < 1 {
			limit = 10
		}
	}

	offset := (page - 1) * limit

	// Count total records (for pagination)
	var total int64
	h.DB.Model(&models.Order{}).Count(&total)

	// Fetch paginated records
	var orders []models.Order
	if err := h.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name")
		}).
		Preload("Frame").
		Preload("Size").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build response
	responses := make([]models.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = models.OrderResponse{
			ID:      o.ID,
			OrderID: o.OrderID,
			User: struct {
				ID   uuid.UUID `json:"id"`
				Name string    `json:"name"`
			}{
				ID:   o.User.ID,
				Name: o.User.Name,
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
			ImageURL:  o.ImageURL,
			Status:    o.Status,
			Notes:     o.Notes,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"page":   page,
		"limit":  limit,
		"total":  total,
		"orders": responses,
	})
}

// PATCH /v1/admin/orders/:id/status (admin)
func (h *OrdersHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var in struct {
		Status string `json:"status"`
	}

	if err := c.BindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": "bad input"})
		return
	}
	if in.Status == "" {
		c.JSON(400, gin.H{"error": "status required"})
		return
	}

	var order models.Order

	// Decide how to query based on the format of `id`
	if strings.HasPrefix(id, "FL-") {
		if err := h.DB.Where("order_id = ?", id).First(&order).Error; err != nil {
			c.JSON(404, gin.H{"error": "order not found"})
			return
		}
	} else {
		if err := h.DB.Where("id = ?", id).First(&order).Error; err != nil {
			c.JSON(404, gin.H{"error": "order not found"})
			return
		}
	}

	if err := h.DB.Model(&order).Update("status", in.Status).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Fetch the user linked to the order
	var user models.User
	if err := h.DB.First(&user, "id = ?", order.UserID).Error; err == nil {
		if h.Email != nil && user.Email != "" {
			data := map[string]string{
				"CustomerName": user.Name,
				"OrderID":      order.OrderID,
				"NewStatus":    in.Status,
				"OrderLink":    fmt.Sprintf("https://framelane.com/track/%s", order.OrderID),
				"Year":         fmt.Sprintf("%d", time.Now().Year()),
			}
			_ = SendOrderStatusUpdate(h.Email, user.Email, data)

		}
	}

	c.JSON(200, gin.H{"ok": true, "status": in.Status})
}

// DELETE /v1/admin/orders/:id (admin)
func (h *OrdersHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")

	var order models.Order
	// Check if order exists
	if strings.HasPrefix(id, "FL-") {
		if err := h.DB.Where("order_id = ?", id).First(&order).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
	} else {
		if err := h.DB.Where("id = ?", id).First(&order).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
	}

	// Delete the order
	if err := h.DB.Delete(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Order %s deleted successfully", order.OrderID),
	})
}

// GET /v1/track/:orderId  (public)
func (h *OrdersHandler) Track(c *gin.Context) {
	oid := c.Param("orderId")
	var o models.Order
	if err := h.DB.Where("order_id = ?", oid).First(&o).Error; err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, gin.H{"orderId": o.OrderID, "status": o.Status, "frame": o.Frame, "size": o.Size})
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
