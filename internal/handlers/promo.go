package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
	"github.com/olamideolayemi/framelane-api/internal/pricing"
)

type PromoHandler struct {
	DB *gorm.DB
}

// POST /v1/promo/validate
// Body: { "code": "WELCOME10", "subtotal": 25000 }
// Returns the discount that would apply against the supplied subtotal.
func (h *PromoHandler) Validate(c *gin.Context) {
	var in struct {
		Code     string `json:"code" binding:"required"`
		Subtotal int    `json:"subtotal" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	var p models.PromoCode
	if err := h.DB.Where("LOWER(code) = ?", strings.ToLower(strings.TrimSpace(in.Code))).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "promo code not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to look up promo", err.Error())
		return
	}
	discount, ok := pricing.ApplyPromo(&p, in.Subtotal)
	if !ok {
		respondError(c, http.StatusBadRequest, "promo not valid for this order", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{
		"code":     p.Code,
		"discount": discount,
		"total":    in.Subtotal - discount,
	}, nil)
}

// GET /v1/admin/promos
func (h *PromoHandler) List(c *gin.Context) {
	var rows []models.PromoCode
	if err := h.DB.Order("created_at DESC").Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list promos", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

// POST /v1/admin/promos
func (h *PromoHandler) Create(c *gin.Context) {
	var in struct {
		Code           string     `json:"code" binding:"required"`
		DiscountType   string     `json:"discountType" binding:"required,oneof=percent fixed"`
		DiscountValue  int        `json:"discountValue" binding:"required,min=1"`
		MaxRedemptions int        `json:"maxRedemptions"`
		MinOrderAmount int        `json:"minOrderAmount"`
		ExpiresAt      *time.Time `json:"expiresAt"`
		Status         string     `json:"status" binding:"omitempty,oneof=active inactive"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.Status == "" {
		in.Status = "active"
	}
	p := models.PromoCode{
		ID:             uuid.New(),
		Code:           strings.ToUpper(strings.TrimSpace(in.Code)),
		DiscountType:   in.DiscountType,
		DiscountValue:  in.DiscountValue,
		MaxRedemptions: in.MaxRedemptions,
		MinOrderAmount: in.MinOrderAmount,
		ExpiresAt:      in.ExpiresAt,
		Status:         in.Status,
	}
	if err := h.DB.Create(&p).Error; err != nil {
		if isDuplicateKeyError(err) {
			respondError(c, http.StatusConflict, "promo code already exists", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create promo", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, p, nil)
}

// PUT /v1/admin/promos/:id
func (h *PromoHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	var p models.PromoCode
	if err := h.DB.First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "promo not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch", err.Error())
		return
	}
	var in struct {
		DiscountType   *string    `json:"discountType,omitempty" binding:"omitempty,oneof=percent fixed"`
		DiscountValue  *int       `json:"discountValue,omitempty"`
		MaxRedemptions *int       `json:"maxRedemptions,omitempty"`
		MinOrderAmount *int       `json:"minOrderAmount,omitempty"`
		ExpiresAt      *time.Time `json:"expiresAt,omitempty"`
		Status         *string    `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.DiscountType != nil {
		p.DiscountType = *in.DiscountType
	}
	if in.DiscountValue != nil {
		p.DiscountValue = *in.DiscountValue
	}
	if in.MaxRedemptions != nil {
		p.MaxRedemptions = *in.MaxRedemptions
	}
	if in.MinOrderAmount != nil {
		p.MinOrderAmount = *in.MinOrderAmount
	}
	if in.ExpiresAt != nil {
		p.ExpiresAt = in.ExpiresAt
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	if err := h.DB.Save(&p).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, p, nil)
}

// DELETE /v1/admin/promos/:id
func (h *PromoHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	result := h.DB.Delete(&models.PromoCode{}, "id = ?", id)
	if result.Error != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "promo not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "deleted"}, nil)
}
