package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/api"
	"github.com/olamideolayemi/framelane-api/internal/models"
)

func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.NewError("missing token", "unauthorized", nil))
			return
		}
		tok := strings.TrimPrefix(h, "Bearer ")
		claims := &Claims{}
		_, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.NewError("invalid token", "unauthorized", nil))
			return
		}
		c.Set("uid", claims.UserID)
		c.Set("admin", claims.IsAdmin)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAdmin, _ := c.Get("admin"); isAdmin != true {
			c.AbortWithStatusJSON(http.StatusForbidden, api.NewError("admin only", "forbidden", nil))
			return
		}
		c.Next()
	}
}

func RequireVerified(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, ok := c.Get("uid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.NewError("missing user", "unauthorized", nil))
			return
		}

		var user models.User
		if err := db.First(&user, "id = ?", uid).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.NewError("user not found", "unauthorized", nil))
			return
		}

		if !user.EmailVerified {
			if user.EmailVerifySentAt != nil {
				expiry := user.EmailVerifySentAt.Add(24 * time.Hour)
				if time.Now().After(expiry) {
					_ = db.Delete(&user).Error
					c.AbortWithStatusJSON(http.StatusUnauthorized, api.NewError("verification expired; account deleted", "unauthorized", nil))
					return
				}
			}
			c.AbortWithStatusJSON(http.StatusForbidden, api.NewError("email not verified", "forbidden", nil))
			return
		}

		c.Next()
	}
}
