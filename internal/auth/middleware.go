package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/olamideolayemi/framelane-api/internal/api"
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
