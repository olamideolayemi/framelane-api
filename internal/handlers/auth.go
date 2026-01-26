package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/auth"
	"github.com/olamideolayemi/framelane-api/internal/models"
)

type AuthHandler struct {
	DB        *gorm.DB
	JWTSecret string
	JWTHours  int
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
		Address  string `json:"address"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Password = strings.TrimSpace(in.Password)
	in.Name = strings.TrimSpace(in.Name)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Address = strings.TrimSpace(in.Address)

	if in.Email == "" {
		respondError(c, http.StatusBadRequest, "email is required", nil)
		return
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		respondError(c, http.StatusBadRequest, "invalid email address", nil)
		return
	}
	if len(in.Password) < 8 {
		respondError(c, http.StatusBadRequest, "password must be at least 8 characters", nil)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to hash password", err.Error())
		return
	}

	u := models.User{
		Name:     in.Name,
		Email:    in.Email,
		Phone:    in.Phone,
		Address:  in.Address,
		Password: string(hash),
	}

	if err := h.DB.Create(&u).Error; err != nil {
		if isDuplicateKeyError(err) {
			respondError(c, http.StatusConflict, "email already registered", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create user", err.Error())
		return
	}

	t, err := auth.MakeToken(h.JWTSecret, u.ID.String(), u.IsAdmin, h.JWTHours)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create token", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, gin.H{
		"token": t,
		"user": gin.H{
			"id":      u.ID.String(),
			"name":    u.Name,
			"email":   u.Email,
			"phone":   u.Phone,
			"address": u.Address,
			"isAdmin": u.IsAdmin,
		},
	}, nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Password = strings.TrimSpace(in.Password)
	if in.Email == "" || in.Password == "" {
		respondError(c, http.StatusBadRequest, "email and password are required", nil)
		return
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		respondError(c, http.StatusBadRequest, "invalid email address", nil)
		return
	}

	var u models.User
	if err := h.DB.Where("email = ?", in.Email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusUnauthorized, "invalid email or password", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch user", err.Error())
		return
	}
	if !u.IsActive {
		respondError(c, http.StatusForbidden, "account is suspended", nil)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		respondError(c, http.StatusUnauthorized, "invalid email or password", nil)
		return
	}

	t, err := auth.MakeToken(h.JWTSecret, u.ID.String(), u.IsAdmin, h.JWTHours)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create token", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{
		"token": t,
		"user": gin.H{
			"id":      u.ID.String(),
			"name":    u.Name,
			"email":   u.Email,
			"phone":   u.Phone,
			"address": u.Address,
			"isAdmin": u.IsAdmin,
		},
	}, nil)
}

// CreateIntent handles payment intent creation (stub implementation)
func (h *AuthHandler) CreateIntent(c *gin.Context) {
	respondSuccess(c, http.StatusOK, gin.H{"message": "payment intent created"}, nil)
}

// func (h *AuthHandler) Me(c *gin.Context) {
// 	uid, _ := c.Get("uid")
// 	var u models.User
// 	if err := h.DB.First(&u, uid).Error; err != nil { c.JSON(404, gin.H{"error":"user not found"}); return }
// 	c.JSON(200, gin.H{"user": gin.H{"id": u.ID, "email": u.Email, "name": u.Name, "isAdmin": u.IsAdmin}})
// }
