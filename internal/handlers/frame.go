package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type FrameHandler struct {
	DB *gorm.DB
}

// List all frame sizes with pagination and filtering
func (h *FrameHandler) ListFrameSizes(c *gin.Context) {
	var query struct {
		Page    int    `form:"page,default=1"`
		PerPage int    `form:"per_page,default=20"`
		Status  string `form:"status"`
		SortBy  string `form:"sort_by"`
		SortDir string `form:"sort_dir,default=asc"`
		Search  string `form:"search"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, http.StatusBadRequest, "invalid query parameters", err.Error())
		return
	}

	// Validate pagination parameters
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PerPage < 1 || query.PerPage > 100 {
		query.PerPage = 20
	}

	// Build query
	db := h.DB.Model(&models.FrameSize{})

	query.Status = strings.ToLower(strings.TrimSpace(query.Status))
	query.Search = strings.TrimSpace(query.Search)

	// Apply status filter
	if query.Status != "" {
		if query.Status != "available" && query.Status != "out_of_stock" {
			respondError(c, http.StatusBadRequest, "status must be 'available' or 'out_of_stock'", nil)
			return
		}
		db = db.Where("status = ?", query.Status)
	}

	// Apply search filter
	if query.Search != "" {
		db = db.Where("name ILIKE ?", "%"+query.Search+"%")
	}

	// Apply sorting
	orderBy := "price ASC"
	if query.SortBy != "" {
		validSortFields := map[string]bool{
			"name":   true,
			"price":  true,
			"status": true,
		}
		if validSortFields[query.SortBy] {
			orderBy = query.SortBy
			if strings.ToLower(query.SortDir) == "desc" {
				orderBy += " DESC"
			} else {
				orderBy += " ASC"
			}
		}
	}
	db = db.Order(orderBy)

	// Get total count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count frames", err.Error())
		return
	}

	// Calculate offset
	offset := (query.Page - 1) * query.PerPage

	// Get paginated results
	var frames []models.FrameSize
	if err := db.Limit(query.PerPage).Offset(offset).Find(&frames).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch frames", err.Error())
		return
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(query.PerPage) - 1) / int64(query.PerPage))

	respondSuccess(c, http.StatusOK, frames, gin.H{
		"page":        query.Page,
		"per_page":    query.PerPage,
		"total":       total,
		"total_pages": totalPages,
	})
}

// Admin: Create a new frame size with enhanced validation
func (h *FrameHandler) CreateFrameSize(c *gin.Context) {
	type CreateFrameRequest struct {
		Name   string `json:"name" binding:"required,min=1,max=100"`
		Price  int    `json:"price" binding:"required,min=0"`
		Status string `json:"status" binding:"omitempty,oneof=available out_of_stock"`
	}

	var req CreateFrameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Trim whitespace
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respondError(c, http.StatusBadRequest, "name cannot be empty", nil)
		return
	}

	var existing models.FrameSize
	if err := h.DB.Where("LOWER(name) = ?", strings.ToLower(req.Name)).First(&existing).Error; err == nil {
		respondError(c, http.StatusConflict, "frame size already exists", nil)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusInternalServerError, "database error during duplicate check", err.Error())
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = "available"
	} else if status != "available" && status != "out_of_stock" {
		respondError(c, http.StatusBadRequest, "status must be 'available' or 'out_of_stock'", nil)
		return
	}

	frame := models.FrameSize{
		ID:     uuid.New(),
		Name:   req.Name,
		Price:  req.Price,
		Status: status,
	}

	if err := h.DB.Create(&frame).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create frame", err.Error())
		return
	}

	respondSuccess(c, http.StatusCreated, frame, nil)
}

// Admin: Update an existing frame size with enhanced validation
func (h *FrameHandler) UpdateFrameSize(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frame ID format", nil)
		return
	}

	var frame models.FrameSize
	if err := h.DB.First(&frame, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "frame not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch frame", err.Error())
		return
	}

	type UpdateFrameRequest struct {
		Name   *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
		Price  *int    `json:"price,omitempty" binding:"omitempty,min=0"`
		Status *string `json:"status,omitempty" binding:"omitempty,oneof=available out_of_stock"`
	}

	var req UpdateFrameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	// Check for duplicate name if updating
	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		if trimmedName == "" {
			respondError(c, http.StatusBadRequest, "name cannot be empty", nil)
			return
		}

		var existing models.FrameSize
		if err := h.DB.Where("LOWER(name) = ? AND id != ?", strings.ToLower(trimmedName), id).First(&existing).Error; err == nil {
			respondError(c, http.StatusConflict, "frame size with this name already exists", nil)
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusInternalServerError, "database error during duplicate check", err.Error())
			return
		}
		frame.Name = trimmedName
	}

	if req.Price != nil {
		frame.Price = *req.Price
	}
	if req.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*req.Status))
		if status == "" {
			respondError(c, http.StatusBadRequest, "status cannot be empty", nil)
			return
		}
		if status != "available" && status != "out_of_stock" {
			respondError(c, http.StatusBadRequest, "status must be 'available' or 'out_of_stock'", nil)
			return
		}
		frame.Status = status
	}

	if err := h.DB.Save(&frame).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update frame", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, frame, nil)
}

// Admin: Delete a frame size
func (h *FrameHandler) DeleteFrameSize(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frame ID", nil)
		return
	}

	result := h.DB.Delete(&models.FrameSize{}, "id = ?", id)
	if result.Error != nil {
		if isForeignKeyConstraintError(result.Error) {
			respondError(c, http.StatusBadRequest, "cannot delete frame size because it is linked to existing orders", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to delete frame", result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "frame not found", nil)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "frame size deleted"}, nil)
}

// List all frame types with pagination and filtering
func (h *FrameHandler) ListFrameTypes(c *gin.Context) {
	var query struct {
		Page    int    `form:"page,default=1"`
		PerPage int    `form:"per_page,default=20"`
		Status  string `form:"status"`
		SortBy  string `form:"sort_by"`
		SortDir string `form:"sort_dir,default=asc"`
		Search  string `form:"search"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, http.StatusBadRequest, "invalid query parameters", err.Error())
		return
	}

	// Validate pagination parameters
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PerPage < 1 || query.PerPage > 100 {
		query.PerPage = 20
	}

	// Build query
	db := h.DB.Model(&models.Frame{})

	query.Status = strings.ToLower(strings.TrimSpace(query.Status))
	query.Search = strings.TrimSpace(query.Search)

	// Apply status filter
	if query.Status != "" {
		if query.Status != "available" && query.Status != "out_of_stock" {
			respondError(c, http.StatusBadRequest, "status must be 'available' or 'out_of_stock'", nil)
			return
		}
		db = db.Where("status = ?", query.Status)
	}

	// Apply search filter
	if query.Search != "" {
		db = db.Where("name ILIKE ?", "%"+query.Search+"%")
	}

	// Apply sorting
	orderBy := "name ASC"
	if query.SortBy != "" {
		validSortFields := map[string]bool{
			"name":   true,
			"status": true,
		}
		if validSortFields[query.SortBy] {
			orderBy = query.SortBy
			if strings.ToLower(query.SortDir) == "desc" {
				orderBy += " DESC"
			} else {
				orderBy += " ASC"
			}
		}
	}
	db = db.Order(orderBy)

	// Get total count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count frame types", err.Error())
		return
	}

	// Calculate offset
	offset := (query.Page - 1) * query.PerPage

	// Get paginated results
	var frames []models.Frame
	if err := db.Limit(query.PerPage).Offset(offset).Find(&frames).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch frame types", err.Error())
		return
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(query.PerPage) - 1) / int64(query.PerPage))

	respondSuccess(c, http.StatusOK, frames, gin.H{
		"page":        query.Page,
		"per_page":    query.PerPage,
		"total":       total,
		"total_pages": totalPages,
	})
}

// Admin: Create a new frame type with enhanced validation
func (h *FrameHandler) CreateFrameType(c *gin.Context) {
	// Parse multipart form
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		respondError(c, http.StatusBadRequest, "name is required", nil)
		return
	}

	// Validate name
	if len(name) < 1 || len(name) > 100 {
		respondError(c, http.StatusBadRequest, "name must be between 1 and 100 characters", nil)
		return
	}

	// Get description
	description := c.PostForm("description")
	description = strings.TrimSpace(description)
	if len(description) > 500 {
		respondError(c, http.StatusBadRequest, "description must be less than 500 characters", nil)
		return
	}

	// Check for duplicate name
	var existing models.Frame
	if err := h.DB.Where("LOWER(name) = ?", strings.ToLower(name)).First(&existing).Error; err == nil {
		respondError(c, http.StatusConflict, "frame type with this name already exists", nil)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusInternalServerError, "database error during duplicate check", err.Error())
		return
	}

	status := strings.ToLower(strings.TrimSpace(c.PostForm("status")))
	if status == "" {
		status = "available"
	} else if status != "available" && status != "out_of_stock" {
		respondError(c, http.StatusBadRequest, "status must be 'available' or 'out_of_stock'", nil)
		return
	}

	// Handle file upload
	var imageURL string
	file, err := c.FormFile("image")
	if err == nil {
		// File was provided
		if file.Size > 5*1024*1024 { // 5MB limit
			respondError(c, http.StatusBadRequest, "file size must be less than 5MB", nil)
			return
		}

		// Validate file type
		allowedTypes := map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/webp": true,
		}
		if !allowedTypes[file.Header.Get("Content-Type")] {
			respondError(c, http.StatusBadRequest, "only JPEG, PNG, and WebP images are allowed", nil)
			return
		}

		filename := uuid.New().String() + "_" + file.Filename

		// For now, we'll use local storage for development
		// In production, this would upload to S3
		imageURL = "/uploads/frames/" + filename

		// Ensure upload directory exists
		if err := ensureUploadDir("./uploads/frames"); err != nil {
			respondError(c, http.StatusInternalServerError, "failed to create upload directory", err.Error())
			return
		}

		// Save the file locally
		if err := c.SaveUploadedFile(file, "./uploads/frames/"+filename); err != nil {
			respondError(c, http.StatusInternalServerError, "failed to save uploaded file", err.Error())
			return
		}
	} else if err != http.ErrMissingFile {
		respondError(c, http.StatusBadRequest, "invalid file upload", err.Error())
		return
	}

	frame := models.Frame{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Status:      status,
		ImageURL:    imageURL,
	}

	if err := h.DB.Create(&frame).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create frame type", err.Error())
		return
	}

	respondSuccess(c, http.StatusCreated, frame, nil)
}

// Admin: Update a frame type with enhanced validation
func (h *FrameHandler) UpdateFrameType(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frame ID format", nil)
		return
	}

	var frame models.Frame
	if err := h.DB.First(&frame, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "frame type not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch frame type", err.Error())
		return
	}

	// Handle multipart form for updates
	updated := false
	if name, ok := c.GetPostForm("name"); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			respondError(c, http.StatusBadRequest, "name cannot be empty", nil)
			return
		}
		if len(name) > 100 {
			respondError(c, http.StatusBadRequest, "name must be between 1 and 100 characters", nil)
			return
		}

		// Check for duplicate name
		var existing models.Frame
		if err := h.DB.Where("LOWER(name) = ? AND id != ?", strings.ToLower(name), id).First(&existing).Error; err == nil {
			respondError(c, http.StatusConflict, "frame type with this name already exists", nil)
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusInternalServerError, "database error during duplicate check", err.Error())
			return
		}
		frame.Name = name
		updated = true
	}

	// Handle description update
	if description, ok := c.GetPostForm("description"); ok {
		description = strings.TrimSpace(description)
		if len(description) > 500 {
			respondError(c, http.StatusBadRequest, "description must be less than 500 characters", nil)
			return
		}
		frame.Description = description
		updated = true
	}

	if status, ok := c.GetPostForm("status"); ok {
		status = strings.ToLower(strings.TrimSpace(status))
		if status == "" {
			respondError(c, http.StatusBadRequest, "status cannot be empty", nil)
			return
		}
		if status != "available" && status != "out_of_stock" {
			respondError(c, http.StatusBadRequest, "status must be 'available' or 'out_of_stock'", nil)
			return
		}
		frame.Status = status
		updated = true
	}

	// Handle file upload for updates
	file, err := c.FormFile("image")
	if err == nil {
		// File was provided
		if file.Size > 5*1024*1024 { // 5MB limit
			respondError(c, http.StatusBadRequest, "file size must be less than 5MB", nil)
			return
		}

		// Validate file type
		allowedTypes := map[string]bool{
			"image/jpeg": true,
			"image/png":  true,
			"image/webp": true,
		}
		if !allowedTypes[file.Header.Get("Content-Type")] {
			respondError(c, http.StatusBadRequest, "only JPEG, PNG, and WebP images are allowed", nil)
			return
		}

		filename := uuid.New().String() + "_" + file.Filename

		// Ensure upload directory exists
		if err := ensureUploadDir("./uploads/frames"); err != nil {
			respondError(c, http.StatusInternalServerError, "failed to create upload directory", err.Error())
			return
		}

		// Save the file locally
		if err := c.SaveUploadedFile(file, "./uploads/frames/"+filename); err != nil {
			respondError(c, http.StatusInternalServerError, "failed to save uploaded file", err.Error())
			return
		}

		frame.ImageURL = "/uploads/frames/" + filename
		updated = true
	} else if err != nil && err != http.ErrMissingFile {
		respondError(c, http.StatusBadRequest, "invalid file upload", err.Error())
		return
	}

	if !updated {
		respondError(c, http.StatusBadRequest, "no fields to update", nil)
		return
	}

	if err := h.DB.Save(&frame).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update frame type", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, frame, nil)
}

// Admin: Delete a frame type
func (h *FrameHandler) DeleteFrameType(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid frame ID", nil)
		return
	}

	result := h.DB.Delete(&models.Frame{}, "id = ?", id)
	if result.Error != nil {
		if isForeignKeyConstraintError(result.Error) {
			respondError(c, http.StatusBadRequest, "cannot delete frame type because it is linked to existing orders", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to delete frame type", result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "frame type not found", nil)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "frame type deleted"}, nil)
}

// Helper function to ensure upload directory exists
func ensureUploadDir(path string) error {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "" {
		return os.ErrInvalid
	}
	return os.MkdirAll(cleanPath, 0o755)
}
