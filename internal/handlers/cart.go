package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/olamideolayemi/framelane-api/internal/pricing"
)

type CartHandler struct {
	DB *gorm.DB
}

func (h *CartHandler) listForUser(uid uuid.UUID) ([]models.CartItem, error) {
	var items []models.CartItem
	err := h.DB.
		Preload("Frame").Preload("Size").
		Preload("Glass").Preload("Lamination").Preload("Finish").
		Where("user_id = ?", uid).
		Order("created_at ASC").
		Find(&items).Error
	return items, err
}

func cartSubtotal(items []models.CartItem) int {
	total := 0
	for _, it := range items {
		total += it.LineTotal
	}
	return total
}

// GET /v1/cart
func (h *CartHandler) List(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	items, err := h.listForUser(uid)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch cart", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{
		"items":    items,
		"subtotal": cartSubtotal(items),
		"currency": "NGN",
	}, nil)
}

type cartItemInput struct {
	FrameID      string `json:"frameId" binding:"required"`
	SizeID       string `json:"sizeId" binding:"required"`
	GlassID      string `json:"glassId"`
	LaminationID string `json:"laminationId"`
	FinishID     string `json:"finishId"`
	ImageURL     string `json:"imageUrl" binding:"required"`
	PreviewURL   string `json:"previewUrl"`
	GiftMessage  string `json:"giftMessage"`
	Quantity     int    `json:"quantity"`
}

// resolveItem hydrates DB rows referenced by the input and computes price.
// Returns the configured CartItem (without ID/UserID/Timestamps) and any
// validation error message + HTTP status.
func (h *CartHandler) resolveItem(in cartItemInput) (models.CartItem, int, string) {
	frameID, err := uuid.Parse(in.FrameID)
	if err != nil {
		return models.CartItem{}, http.StatusBadRequest, "invalid frameId"
	}
	sizeID, err := uuid.Parse(in.SizeID)
	if err != nil {
		return models.CartItem{}, http.StatusBadRequest, "invalid sizeId"
	}

	var frame models.Frame
	if err := h.DB.First(&frame, "id = ?", frameID).Error; err != nil {
		return models.CartItem{}, http.StatusNotFound, "frame not found"
	}
	if strings.ToLower(frame.Status) != "available" {
		return models.CartItem{}, http.StatusConflict, "frame is not available"
	}

	var size models.FrameSize
	if err := h.DB.First(&size, "id = ?", sizeID).Error; err != nil {
		return models.CartItem{}, http.StatusNotFound, "frame size not found"
	}
	if strings.ToLower(size.Status) != "available" {
		return models.CartItem{}, http.StatusConflict, "frame size is not available"
	}

	var glassPtr *models.Glass
	var glassIDPtr *uuid.UUID
	if in.GlassID != "" {
		gid, err := uuid.Parse(in.GlassID)
		if err != nil {
			return models.CartItem{}, http.StatusBadRequest, "invalid glassId"
		}
		var glass models.Glass
		if err := h.DB.First(&glass, "id = ?", gid).Error; err != nil {
			return models.CartItem{}, http.StatusNotFound, "glass not found"
		}
		glassPtr = &glass
		glassIDPtr = &gid
	}

	var lamPtr *models.Lamination
	var lamIDPtr *uuid.UUID
	if in.LaminationID != "" {
		lid, err := uuid.Parse(in.LaminationID)
		if err != nil {
			return models.CartItem{}, http.StatusBadRequest, "invalid laminationId"
		}
		var lam models.Lamination
		if err := h.DB.First(&lam, "id = ?", lid).Error; err != nil {
			return models.CartItem{}, http.StatusNotFound, "lamination not found"
		}
		lamPtr = &lam
		lamIDPtr = &lid
	}

	var finPtr *models.FrameFinish
	var finIDPtr *uuid.UUID
	if in.FinishID != "" {
		fid, err := uuid.Parse(in.FinishID)
		if err != nil {
			return models.CartItem{}, http.StatusBadRequest, "invalid finishId"
		}
		var fin models.FrameFinish
		if err := h.DB.First(&fin, "id = ?", fid).Error; err != nil {
			return models.CartItem{}, http.StatusNotFound, "finish not found"
		}
		if fin.FrameID != frame.ID {
			return models.CartItem{}, http.StatusBadRequest, "finish does not belong to selected frame"
		}
		finPtr = &fin
		finIDPtr = &fid
	}

	qty := in.Quantity
	if qty < 1 {
		qty = 1
	}
	unit := pricing.ItemUnitPrice(size, glassPtr, lamPtr, finPtr)

	return models.CartItem{
		FrameID:      frameID,
		SizeID:       sizeID,
		GlassID:      glassIDPtr,
		LaminationID: lamIDPtr,
		FinishID:     finIDPtr,
		ImageURL:     strings.TrimSpace(in.ImageURL),
		PreviewURL:   strings.TrimSpace(in.PreviewURL),
		GiftMessage:  strings.TrimSpace(in.GiftMessage),
		Quantity:     qty,
		UnitPrice:    unit,
		LineTotal:    pricing.LineTotal(unit, qty),
	}, 0, ""
}

// POST /v1/cart
func (h *CartHandler) Add(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	var in cartItemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	item, status, msg := h.resolveItem(in)
	if status != 0 {
		respondError(c, status, msg, nil)
		return
	}
	item.ID = uuid.New()
	item.UserID = uid
	if err := h.DB.Create(&item).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to add to cart", err.Error())
		return
	}
	if err := h.DB.Preload("Frame").Preload("Size").Preload("Glass").Preload("Lamination").Preload("Finish").
		First(&item, "id = ?", item.ID).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to load cart item", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, item, nil)
}

// PATCH /v1/cart/:id
func (h *CartHandler) Update(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid cart item id", nil)
		return
	}
	var item models.CartItem
	if err := h.DB.Preload("Size").Preload("Glass").Preload("Lamination").Preload("Finish").
		First(&item, "id = ? AND user_id = ?", id, uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "cart item not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch cart item", err.Error())
		return
	}

	var in struct {
		Quantity    *int    `json:"quantity,omitempty"`
		GiftMessage *string `json:"giftMessage,omitempty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.Quantity != nil {
		q := *in.Quantity
		if q < 1 {
			q = 1
		}
		item.Quantity = q
		item.LineTotal = pricing.LineTotal(item.UnitPrice, q)
	}
	if in.GiftMessage != nil {
		item.GiftMessage = strings.TrimSpace(*in.GiftMessage)
	}
	if err := h.DB.Save(&item).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update cart item", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, item, nil)
}

// DELETE /v1/cart/:id
func (h *CartHandler) Remove(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid cart item id", nil)
		return
	}
	result := h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&models.CartItem{})
	if result.Error != nil {
		respondError(c, http.StatusInternalServerError, "failed to remove cart item", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "cart item not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "removed"}, nil)
}

// DELETE /v1/cart  (clear)
func (h *CartHandler) Clear(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	if err := h.DB.Where("user_id = ?", uid).Delete(&models.CartItem{}).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to clear cart", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "cart cleared"}, nil)
}

// uidFromCtx pulls the uid set by RequireAuth and returns it parsed as UUID.
// Writes an error response if anything is wrong; callers should check ok.
func uidFromCtx(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get("uid")
	if !exists {
		respondError(c, http.StatusUnauthorized, "authentication required", nil)
		return uuid.Nil, false
	}
	s, ok := v.(string)
	if !ok {
		respondError(c, http.StatusInternalServerError, "invalid user context", nil)
		return uuid.Nil, false
	}
	uid, err := uuid.Parse(s)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid user id", nil)
		return uuid.Nil, false
	}
	return uid, true
}
