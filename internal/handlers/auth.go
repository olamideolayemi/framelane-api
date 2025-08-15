package handlers

import (
	// "net/http"
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
		Email, Password, Name, Phone, Address string
	}
	if err := c.BindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": "bad input"})
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), 12)

	u := models.User{
		Name:     in.Name,
		Email:    in.Email,
		Phone:    in.Phone,
		Address:  in.Address,
		Password: string(hash),
	}

	if err := h.DB.Create(&u).Error; err != nil {
		c.JSON(400, gin.H{"error": "email exists?"})
		return
	}

	t, _ := auth.MakeToken(h.JWTSecret, u.ID.String(), u.IsAdmin, h.JWTHours)
	c.JSON(201, gin.H{
		"token": t,
		"user": gin.H{
			"id":      u.ID.String(),
			"name":    u.Name,
			"email":   u.Email,
			"phone":   u.Phone,
			"address": u.Address,
			"isAdmin": u.IsAdmin,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in struct{ Email, Password string }
	if err := c.BindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": "bad input"})
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	var u models.User
	if err := h.DB.Where("email = ?", in.Email).First(&u).Error; err != nil {
		c.JSON(401, gin.H{"error": "invalid creds"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		c.JSON(401, gin.H{"error": "invalid creds"})
		return
	}

	t, _ := auth.MakeToken(h.JWTSecret, u.ID.String(), u.IsAdmin, h.JWTHours)
	c.JSON(200, gin.H{
		"token": t,
		"user": gin.H{
			"id":      u.ID.String(),
			"name":    u.Name,
			"email":   u.Email,
			"phone":   u.Phone,
			"address": u.Address,
			"isAdmin": u.IsAdmin,
		},
	})
}

// CreateIntent handles payment intent creation (stub implementation)
func (h *AuthHandler) CreateIntent(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Payment intent created"})
}

// func (h *AuthHandler) Me(c *gin.Context) {
// 	uid, _ := c.Get("uid")
// 	var u models.User
// 	if err := h.DB.First(&u, uid).Error; err != nil { c.JSON(404, gin.H{"error":"user not found"}); return }
// 	c.JSON(200, gin.H{"user": gin.H{"id": u.ID, "email": u.Email, "name": u.Name, "isAdmin": u.IsAdmin}})
// }
