package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/auth"
	"github.com/olamideolayemi/framelane-api/internal/email"
	"github.com/olamideolayemi/framelane-api/internal/models"
)

type AuthHandler struct {
	DB        *gorm.DB
	JWTSecret string
	JWTHours  int
	Email     *email.Sender
	FrontendBaseURL string
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

	verifyToken, verifyHash, err := generateToken()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create verification token", err.Error())
		return
	}
	now := time.Now()

	u := models.User{
		Name:                in.Name,
		Email:               in.Email,
		Phone:               in.Phone,
		Address:             in.Address,
		Password:            string(hash),
		EmailVerified:       false,
		EmailVerifyTokenHash: verifyHash,
		EmailVerifySentAt:   &now,
	}

	if err := h.DB.Create(&u).Error; err != nil {
		if isDuplicateKeyError(err) {
			respondError(c, http.StatusConflict, "email already registered", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create user", err.Error())
		return
	}

	if h.Email == nil {
		_ = h.DB.Delete(&u).Error
		respondError(c, http.StatusServiceUnavailable, "email service unavailable", nil)
		return
	}

	verifyURL := fmt.Sprintf("%s/v1/auth/verify?token=%s", requestBaseURL(c), verifyToken)
	name := u.Name
	if strings.TrimSpace(name) == "" {
		name = "there"
	}
	html, err := email.ParseTemplate("verify_email.html", map[string]any{
		"Name":      name,
		"VerifyURL": verifyURL,
	})
	if err != nil {
		_ = h.DB.Delete(&u).Error
		respondError(c, http.StatusInternalServerError, "failed to build verification email", err.Error())
		return
	}
	if err := h.Email.Send(u.Email, "Verify your email", html); err != nil {
		_ = h.DB.Delete(&u).Error
		respondError(c, http.StatusInternalServerError, "failed to send verification email", err.Error())
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
		"verificationRequired": true,
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
	if !u.EmailVerified {
		if u.EmailVerifySentAt != nil && time.Now().After(u.EmailVerifySentAt.Add(24*time.Hour)) {
			_ = h.DB.Delete(&u).Error
			respondError(c, http.StatusUnauthorized, "verification expired; account deleted", nil)
			return
		}
		respondError(c, http.StatusForbidden, "email not verified", nil)
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

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		respondError(c, http.StatusBadRequest, "verification token is required", nil)
		return
	}
	hash := hashToken(token)

	var user models.User
	if err := h.DB.Where("email_verify_token_hash = ?", hash).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "invalid or expired verification token", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch user", err.Error())
		return
	}

	if user.EmailVerified {
		respondSuccess(c, http.StatusOK, gin.H{"message": "email already verified"}, nil)
		return
	}

	if user.EmailVerifySentAt == nil {
		respondError(c, http.StatusBadRequest, "verification token is invalid", nil)
		return
	}

	if time.Now().After(user.EmailVerifySentAt.Add(24 * time.Hour)) {
		_ = h.DB.Delete(&user).Error
		respondError(c, http.StatusUnauthorized, "verification expired; account deleted", nil)
		return
	}

	now := time.Now()
	if err := h.DB.Model(&user).Updates(map[string]interface{}{
		"email_verified":        true,
		"email_verified_at":     &now,
		"email_verify_token_hash": "",
		"email_verify_sent_at":  nil,
	}).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to verify email", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "email verified"}, nil)
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var in struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email == "" {
		respondError(c, http.StatusBadRequest, "email is required", nil)
		return
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		respondError(c, http.StatusBadRequest, "invalid email address", nil)
		return
	}

	var user models.User
	if err := h.DB.Where("email = ?", in.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondSuccess(c, http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"}, nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch user", err.Error())
		return
	}

	if h.Email == nil {
		respondError(c, http.StatusServiceUnavailable, "email service unavailable", nil)
		return
	}

	resetToken, resetHash, err := generateToken()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create reset token", err.Error())
		return
	}
	now := time.Now()
	expiry := now.Add(time.Hour)
	if err := h.DB.Model(&user).Updates(map[string]interface{}{
		"password_reset_token_hash": resetHash,
		"password_reset_sent_at":    &now,
		"password_reset_expires_at": &expiry,
	}).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save reset token", err.Error())
		return
	}

	resetURL := fmt.Sprintf("%s/login?mode=reset&token=%s", frontendBaseURL(c, h.FrontendBaseURL), resetToken)
	name := user.Name
	if strings.TrimSpace(name) == "" {
		name = "there"
	}
	html, err := email.ParseTemplate("reset_password.html", map[string]any{
		"Name":        name,
		"ResetURL":    resetURL,
		"ExpiryHours": 1,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to build reset email", err.Error())
		return
	}
	if err := h.Email.Send(user.Email, "Reset your password", html); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to send reset email", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"}, nil)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var in struct {
		Token           string `json:"token" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.Token = strings.TrimSpace(in.Token)
	in.Password = strings.TrimSpace(in.Password)
	in.ConfirmPassword = strings.TrimSpace(in.ConfirmPassword)

	if in.Token == "" {
		respondError(c, http.StatusBadRequest, "reset token is required", nil)
		return
	}
	if len(in.Password) < 8 {
		respondError(c, http.StatusBadRequest, "password must be at least 8 characters", nil)
		return
	}
	if in.ConfirmPassword != "" && in.Password != in.ConfirmPassword {
		respondError(c, http.StatusBadRequest, "passwords do not match", nil)
		return
	}

	hash := hashToken(in.Token)
	var user models.User
	if err := h.DB.Where("password_reset_token_hash = ?", hash).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "invalid or expired reset token", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch user", err.Error())
		return
	}

	if user.PasswordResetExpiresAt == nil || time.Now().After(*user.PasswordResetExpiresAt) {
		_ = h.DB.Model(&user).Updates(map[string]interface{}{
			"password_reset_token_hash": "",
			"password_reset_sent_at":    nil,
			"password_reset_expires_at": nil,
		}).Error
		respondError(c, http.StatusUnauthorized, "reset token expired", nil)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to hash password", err.Error())
		return
	}

	if err := h.DB.Model(&user).Updates(map[string]interface{}{
		"password":                  string(passwordHash),
		"password_reset_token_hash": "",
		"password_reset_sent_at":    nil,
		"password_reset_expires_at": nil,
	}).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to reset password", err.Error())
		return
	}

	respondSuccess(c, http.StatusOK, gin.H{"message": "password reset successful"}, nil)
}

func generateToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	hash := hashToken(token)
	return token, hash, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func frontendBaseURL(c *gin.Context, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimRight(strings.TrimSpace(configured), "/")
	}
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin != "" {
		return strings.TrimRight(origin, "/")
	}
	return requestBaseURL(c)
}

func requestBaseURL(c *gin.Context) string {
	proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s", proto, host)
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
