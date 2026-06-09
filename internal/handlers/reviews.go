package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type ReviewsHandler struct {
	DB *gorm.DB
}

// GET /v1/reviews/frame/:frameId  (public — approved only)
func (h *ReviewsHandler) ListByFrame(c *gin.Context) {
	fid, err := uuid.Parse(c.Param("frameId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frameId", nil)
		return
	}
	var rows []models.Review
	if err := h.DB.
		Where("frame_id = ? AND status = ?", fid, models.ReviewStatusApproved).
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name") }).
		Order("created_at DESC").Limit(50).Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list reviews", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

// POST /v1/reviews  (auth + verified)
// Body: { "orderItemId": "...", "rating": 1..5, "body": "...", "photoUrl": "..." }
func (h *ReviewsHandler) Create(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	var in struct {
		OrderItemID string `json:"orderItemId" binding:"required"`
		Rating      int    `json:"rating" binding:"required,min=1,max=5"`
		Body        string `json:"body"`
		PhotoURL    string `json:"photoUrl"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	itemID, err := uuid.Parse(in.OrderItemID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid orderItemId", nil)
		return
	}
	// Verify ownership + delivered status.
	var item models.OrderItem
	if err := h.DB.First(&item, "id = ?", itemID).Error; err != nil {
		respondError(c, http.StatusNotFound, "order item not found", nil)
		return
	}
	var order models.Order
	if err := h.DB.First(&order, "id = ?", item.OrderID).Error; err != nil {
		respondError(c, http.StatusNotFound, "order not found", nil)
		return
	}
	if order.UserID != uid {
		respondError(c, http.StatusForbidden, "not your order", nil)
		return
	}
	if !strings.EqualFold(order.Status, OrderStatusDelivered) {
		respondError(c, http.StatusBadRequest, "you can review after delivery", nil)
		return
	}
	// One review per order item.
	var existing models.Review
	if err := h.DB.Where("order_item_id = ?", itemID).First(&existing).Error; err == nil {
		respondError(c, http.StatusConflict, "you already reviewed this item", nil)
		return
	}

	r := models.Review{
		ID: uuid.New(), UserID: uid, OrderItemID: itemID, FrameID: item.FrameID,
		Rating: in.Rating, Body: strings.TrimSpace(in.Body), PhotoURL: strings.TrimSpace(in.PhotoURL),
		Status: models.ReviewStatusPending,
	}
	if err := h.DB.Create(&r).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create review", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, r, nil)
}

// GET /v1/admin/reviews
func (h *ReviewsHandler) AdminList(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	q := h.DB.Model(&models.Review{}).
		Preload("User", func(db *gorm.DB) *gorm.DB { return db.Select("id", "name", "email") }).
		Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []models.Review
	if err := q.Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list reviews", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

// PATCH /v1/admin/reviews/:id/status
func (h *ReviewsHandler) SetStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	var in struct {
		Status string `json:"status" binding:"required,oneof=pending approved hidden"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	result := h.DB.Model(&models.Review{}).Where("id = ?", id).Update("status", in.Status)
	if result.Error != nil {
		respondError(c, http.StatusInternalServerError, "failed to update", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "review not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"status": in.Status}, nil)
}

// DELETE /v1/admin/reviews/:id
func (h *ReviewsHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	result := h.DB.Delete(&models.Review{}, "id = ?", id)
	if result.Error != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "review not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "deleted"}, nil)
}

