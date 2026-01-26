package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type UsersHandler struct {
	DB *gorm.DB
}

// UserResponse is a safe representation of user data for API responses
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Phone     string    `json:"phone"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name     string `json:"name,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Address  string `json:"address,omitempty"`
	Password string `json:"password,omitempty"`
}

// List all users
func (h *UsersHandler) ListUsers(c *gin.Context) {
	// Parse query params with defaults
	page := 1
	limit := 10
	if p := strings.TrimSpace(c.Query("page")); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := strings.TrimSpace(c.Query("limit")); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Fetch total count for pagination
	var total int64
	if err := h.DB.Model(&models.User{}).Count(&total).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to count users", err.Error())
		return
	}

	// Fetch paginated results
	var users []models.User
	if err := h.DB.Offset(offset).Limit(limit).Order("created_at DESC").Find(&users).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "database error", err.Error())
		return
	}

	// Build safe response with timestamps
	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = UserResponse{
			ID:        u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Address:   u.Address,
			Phone:     u.Phone,
			IsAdmin:   u.IsAdmin,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		}
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	respondSuccess(c, http.StatusOK, response, gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	})
}

// Get user by ID
func (h *UsersHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID", nil)
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "user not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "database error", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Address:   user.Address,
		Phone:     user.Phone,
		IsAdmin:   user.IsAdmin,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil)
}

// Suspend user
func (h *UsersHandler) SuspendUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID", nil)
		return
	}

	var user models.User
	if err := h.DB.First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "user not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "database error", err.Error())
		return
	}

	user.IsActive = false
	if err := h.DB.Save(&user).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to suspend user", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "user suspended successfully"}, nil)
}

// Delete user
func (h *UsersHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID", nil)
		return
	}

	result := h.DB.Delete(&models.User{}, "id = ?", id)
	if result.Error != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete user", result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "user not found", nil)
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "user deleted successfully"}, nil)
}

func (h *UsersHandler) UpdateUserProfile(c *gin.Context) {
	// Get authenticated user ID from context
	userIDVal, exists := c.Get("uid")
	if !exists {
		respondError(c, http.StatusUnauthorized, "authentication required", nil)
		return
	}

	userID, ok := userIDVal.(string)
	if !ok {
		respondError(c, http.StatusInternalServerError, "invalid user context", nil)
		return
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid user ID", nil)
		return
	}

	// Bind request body
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Address = strings.TrimSpace(req.Address)
	req.Password = strings.TrimSpace(req.Password)

	// Fetch user
	var user models.User
	if err := h.DB.First(&user, "id = ?", parsedUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "user not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch user", err.Error())
		return
	}

	// Update only provided fields
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Address != "" {
		user.Address = req.Address
	}
	if req.Password != "" {
		if len(req.Password) < 8 {
			respondError(c, http.StatusBadRequest, "password must be at least 8 characters", nil)
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to hash password", err.Error())
			return
		}
		user.Password = string(hashedPassword)
	}

	// Save changes
	if err := h.DB.Save(&user).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update profile", err.Error())
		return
	}

	// Build safe user response
	response := UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Address:   user.Address,
		Phone:     user.Phone,
		IsAdmin:   user.IsAdmin,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	respondSuccess(c, http.StatusOK, gin.H{
		"message": "profile updated successfully",
		"user":    response,
	}, nil)
}
