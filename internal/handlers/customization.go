package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type CustomizationHandler struct {
	DB *gorm.DB
}

// ---------- Glass ----------

func (h *CustomizationHandler) ListGlasses(c *gin.Context) {
	var rows []models.Glass
	if err := h.DB.Where("status = ?", "available").Order("price_modifier ASC").Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch glasses", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

func (h *CustomizationHandler) CreateGlass(c *gin.Context) {
	var in struct {
		Name          string `json:"name" binding:"required,min=1,max=80"`
		Description   string `json:"description"`
		PriceModifier int    `json:"priceModifier"`
		Status        string `json:"status" binding:"omitempty,oneof=available out_of_stock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Status == "" {
		in.Status = "available"
	}
	g := models.Glass{
		ID: uuid.New(), Name: in.Name, Description: strings.TrimSpace(in.Description),
		PriceModifier: in.PriceModifier, Status: in.Status,
	}
	if err := h.DB.Create(&g).Error; err != nil {
		if isDuplicateKeyError(err) {
			respondError(c, http.StatusConflict, "glass already exists", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create glass", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, g, nil)
}

func (h *CustomizationHandler) UpdateGlass(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	var g models.Glass
	if err := h.DB.First(&g, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "glass not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch glass", err.Error())
		return
	}
	var in struct {
		Name          *string `json:"name,omitempty"`
		Description   *string `json:"description,omitempty"`
		PriceModifier *int    `json:"priceModifier,omitempty"`
		Status        *string `json:"status,omitempty" binding:"omitempty,oneof=available out_of_stock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.Name != nil {
		g.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		g.Description = strings.TrimSpace(*in.Description)
	}
	if in.PriceModifier != nil {
		g.PriceModifier = *in.PriceModifier
	}
	if in.Status != nil {
		g.Status = *in.Status
	}
	if err := h.DB.Save(&g).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update glass", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, g, nil)
}

func (h *CustomizationHandler) DeleteGlass(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	result := h.DB.Delete(&models.Glass{}, "id = ?", id)
	if result.Error != nil {
		if isForeignKeyConstraintError(result.Error) {
			respondError(c, http.StatusBadRequest, "cannot delete glass linked to existing orders", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to delete", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "glass not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "deleted"}, nil)
}

// ---------- Lamination ----------

func (h *CustomizationHandler) ListLaminations(c *gin.Context) {
	var rows []models.Lamination
	if err := h.DB.Where("status = ?", "available").Order("price_modifier ASC").Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch laminations", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

func (h *CustomizationHandler) CreateLamination(c *gin.Context) {
	var in struct {
		Name          string `json:"name" binding:"required,min=1,max=80"`
		Description   string `json:"description"`
		PriceModifier int    `json:"priceModifier"`
		Status        string `json:"status" binding:"omitempty,oneof=available out_of_stock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.Status == "" {
		in.Status = "available"
	}
	l := models.Lamination{
		ID: uuid.New(), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description),
		PriceModifier: in.PriceModifier, Status: in.Status,
	}
	if err := h.DB.Create(&l).Error; err != nil {
		if isDuplicateKeyError(err) {
			respondError(c, http.StatusConflict, "lamination already exists", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create lamination", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, l, nil)
}

func (h *CustomizationHandler) UpdateLamination(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	var l models.Lamination
	if err := h.DB.First(&l, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "lamination not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch", err.Error())
		return
	}
	var in struct {
		Name          *string `json:"name,omitempty"`
		Description   *string `json:"description,omitempty"`
		PriceModifier *int    `json:"priceModifier,omitempty"`
		Status        *string `json:"status,omitempty" binding:"omitempty,oneof=available out_of_stock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.Name != nil {
		l.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		l.Description = strings.TrimSpace(*in.Description)
	}
	if in.PriceModifier != nil {
		l.PriceModifier = *in.PriceModifier
	}
	if in.Status != nil {
		l.Status = *in.Status
	}
	if err := h.DB.Save(&l).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, l, nil)
}

func (h *CustomizationHandler) DeleteLamination(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	result := h.DB.Delete(&models.Lamination{}, "id = ?", id)
	if result.Error != nil {
		if isForeignKeyConstraintError(result.Error) {
			respondError(c, http.StatusBadRequest, "cannot delete lamination linked to existing orders", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to delete", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "lamination not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "deleted"}, nil)
}

// ---------- Frame Finish ----------

func (h *CustomizationHandler) ListFinishes(c *gin.Context) {
	frameID := strings.TrimSpace(c.Query("frameId"))
	q := h.DB.Where("status = ?", "available")
	if frameID != "" {
		fid, err := uuid.Parse(frameID)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid frameId", nil)
			return
		}
		q = q.Where("frame_id = ?", fid)
	}
	var rows []models.FrameFinish
	if err := q.Order("name ASC").Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch finishes", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

func (h *CustomizationHandler) CreateFinish(c *gin.Context) {
	var in struct {
		FrameID       string `json:"frameId" binding:"required"`
		Name          string `json:"name" binding:"required,min=1,max=80"`
		HexColor      string `json:"hexColor"`
		ImageURL      string `json:"imageUrl"`
		PriceModifier int    `json:"priceModifier"`
		Status        string `json:"status" binding:"omitempty,oneof=available out_of_stock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	fid, err := uuid.Parse(in.FrameID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frameId", nil)
		return
	}
	if in.Status == "" {
		in.Status = "available"
	}
	f := models.FrameFinish{
		ID: uuid.New(), FrameID: fid,
		Name:          strings.TrimSpace(in.Name),
		HexColor:      strings.TrimSpace(in.HexColor),
		ImageURL:      strings.TrimSpace(in.ImageURL),
		PriceModifier: in.PriceModifier,
		Status:        in.Status,
	}
	if err := h.DB.Create(&f).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create finish", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, f, nil)
}

func (h *CustomizationHandler) UpdateFinish(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	var f models.FrameFinish
	if err := h.DB.First(&f, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "finish not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch", err.Error())
		return
	}
	var in struct {
		Name          *string `json:"name,omitempty"`
		HexColor      *string `json:"hexColor,omitempty"`
		ImageURL      *string `json:"imageUrl,omitempty"`
		PriceModifier *int    `json:"priceModifier,omitempty"`
		Status        *string `json:"status,omitempty" binding:"omitempty,oneof=available out_of_stock"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if in.Name != nil {
		f.Name = strings.TrimSpace(*in.Name)
	}
	if in.HexColor != nil {
		f.HexColor = strings.TrimSpace(*in.HexColor)
	}
	if in.ImageURL != nil {
		f.ImageURL = strings.TrimSpace(*in.ImageURL)
	}
	if in.PriceModifier != nil {
		f.PriceModifier = *in.PriceModifier
	}
	if in.Status != nil {
		f.Status = *in.Status
	}
	if err := h.DB.Save(&f).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, f, nil)
}

func (h *CustomizationHandler) DeleteFinish(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	result := h.DB.Delete(&models.FrameFinish{}, "id = ?", id)
	if result.Error != nil {
		if isForeignKeyConstraintError(result.Error) {
			respondError(c, http.StatusBadRequest, "cannot delete finish linked to existing orders", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to delete", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "finish not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "deleted"}, nil)
}
