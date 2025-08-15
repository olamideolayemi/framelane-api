package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type FrameHandler struct {
	DB *gorm.DB
}

// List all frame sizes (user & admin)
func (h *FrameHandler) ListFrameSizes(c *gin.Context) {
	var frames []models.FrameSize
	if err := h.DB.Order("price ASC").Find(&frames).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch frames"})
		return
	}
	c.JSON(http.StatusOK, frames)
}

// Admin: Create a new frame size
func (h *FrameHandler) CreateFrameSize(c *gin.Context) {
	type CreateFrameRequest struct {
		Name   string `json:"name" binding:"required"`
		Price  int    `json:"price" binding:"required"`
		Status string `json:"status"` // optional, default to available
	}

	var req CreateFrameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var existing models.FrameSize
	if err := h.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "frame size already exists"})
		return
	}

	if req.Status == "" {
		req.Status = "available"
	} else if req.Status != "available" && req.Status != "out_of_stock" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'available' or 'out_of_stock'"})
		return
	}

	frame := models.FrameSize{
		ID:     uuid.New(),
		Name:   req.Name,
		Price:  req.Price,
		Status: req.Status,
	}

	if err := h.DB.Create(&frame).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create frame"})
		return
	}

	c.JSON(http.StatusCreated, frame)
}

// Admin: Update an existing frame size (price, name, status)
func (h *FrameHandler) UpdateFrameSize(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frame ID"})
		return
	}

	var frame models.FrameSize
	if err := h.DB.First(&frame, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "frame not found"})
		return
	}

	// Request payload struct
	type UpdateFrameRequest struct {
		Name   string `json:"name,omitempty"`
		Price  *int   `json:"price,omitempty"`  // pointer to detect if sent
		Status string `json:"status,omitempty"` // "available" or "out_of_stock"
	}

	var req UpdateFrameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Name != "" {
		frame.Name = req.Name
	}
	if req.Price != nil {
		frame.Price = *req.Price
	}
	if req.Status != "" {
		if req.Status != "available" && req.Status != "out_of_stock" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'available' or 'out_of_stock'"})
			return
		}
		frame.Status = req.Status
	}

	if err := h.DB.Save(&frame).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update frame"})
		return
	}

	c.JSON(http.StatusOK, frame)
}

// Admin: Delete a frame size
func (h *FrameHandler) DeleteFrameSize(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frame ID"})
		return
	}

	if err := h.DB.Delete(&models.FrameSize{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete frame"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List all frame types
func (h *FrameHandler) ListFrameTypes(c *gin.Context) {
	var frames []models.Frame
	if err := h.DB.Order("name ASC").Find(&frames).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch frame types"})
		return
	}
	c.JSON(http.StatusOK, frames)
}

// Admin: Create a new frame type
func (h *FrameHandler) CreateFrameType(c *gin.Context) {
	type Request struct {
		Name   string `json:"name" binding:"required"`
		Status string `json:"status"` // optional
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Status == "" {
		req.Status = "available"
	} else if req.Status != "available" && req.Status != "out_of_stock" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'available' or 'out_of_stock'"})
		return
	}

	frame := models.Frame{
		ID:     uuid.New(),
		Name:   req.Name,
		Status: req.Status,
	}

	if err := h.DB.Create(&frame).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create frame type"})
		return
	}

	c.JSON(http.StatusCreated, frame)
}

// Admin: Update a frame type
func (h *FrameHandler) UpdateFrameType(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frame ID"})
		return
	}

	var frame models.Frame
	if err := h.DB.First(&frame, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "frame type not found"})
		return
	}

	type Request struct {
		Name   string `json:"name,omitempty"`
		Status string `json:"status,omitempty"`
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Name != "" {
		frame.Name = req.Name
	}
	if req.Status != "" {
		if req.Status != "available" && req.Status != "out_of_stock" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'available' or 'out_of_stock'"})
			return
		}
		frame.Status = req.Status
	}

	if err := h.DB.Save(&frame).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update frame type"})
		return
	}

	c.JSON(http.StatusOK, frame)
}

// Admin: Delete a frame type
func (h *FrameHandler) DeleteFrameType(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frame ID"})
		return
	}

	if err := h.DB.Delete(&models.Frame{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete frame type"})
		return
	}

	c.Status(http.StatusNoContent)
}
