package handlers

import (
	"crypto/rand"
	"fmt"
	// "net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
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

// POST /v1/orders  (guest or logged-in)
func (h *OrdersHandler) Create(c *gin.Context) {
	var in struct {
		UserEmail string `json:"email"`
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Address   string `json:"address"`
		Frame     string `json:"frame"`
		Size      string `json:"size"`
		Room      string `json:"room"`
		ImageURL  string `json:"imageUrl"`
	}
	if err := c.BindJSON(&in); err != nil { c.JSON(400, gin.H{"error":"bad input"}); return }

	var uid *uint
	if v, ok := c.Get("uid"); ok {
		id := v.(uint); uid = &id
	}
	o := models.Order{
		OrderID:   strings.ToUpper("FL-" + randID()),
		UserID:    uid,
		UserEmail: in.UserEmail,
		Name:      in.Name,
		Phone:     in.Phone,
		Address:   in.Address,
		Frame:     in.Frame,
		Size:      in.Size,
		Room:      in.Room,
		ImageURL:  in.ImageURL,
	}
	if err := h.DB.Create(&o).Error; err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }

	// email (best-effort)
	if h.Email != nil && in.UserEmail != "" {
		_ = h.Email.Send(in.UserEmail, "🖼️ Your FrameLane Order",
			fmt.Sprintf("<p>Hi %s,</p><p>Thanks! Your order id is <b>%s</b>.</p>", o.Name, o.OrderID))
	}

	c.JSON(201, gin.H{"orderId": o.OrderID, "id": o.ID})
}

// GET /v1/orders (auth) -> list own
func (h *OrdersHandler) ListMine(c *gin.Context) {
	uid, _ := c.Get("uid")
	var res []models.Order
	if err := h.DB.Where("user_id = ?", uid.(uint)).Order("id DESC").Find(&res).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()}); return
	}
	c.JSON(200, res)
}

// GET /v1/admin/orders (admin)
func (h *OrdersHandler) ListAll(c *gin.Context) {
	var res []models.Order
	if err := h.DB.Order("id DESC").Find(&res).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()}); return
	}
	c.JSON(200, res)
}

// PATCH /v1/admin/orders/:id/status (admin)
func (h *OrdersHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	var in struct{ Status string `json:"status"` }
	if err := c.BindJSON(&in); err != nil { c.JSON(400, gin.H{"error":"bad input"}); return }
	if in.Status == "" { c.JSON(400, gin.H{"error":"status required"}); return }
	if err := h.DB.Model(&models.Order{}).Where("id = ?", id).Update("status", in.Status).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()}); return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GET /v1/track/:orderId  (public)
func (h *OrdersHandler) Track(c *gin.Context) {
	oid := c.Param("orderId")
	var o models.Order
	if err := h.DB.Where("order_id = ?", oid).First(&o).Error; err != nil {
		c.JSON(404, gin.H{"error":"not found"}); return
	}
	c.JSON(200, gin.H{"orderId": o.OrderID, "status": o.Status, "frame": o.Frame, "size": o.Size})
}
