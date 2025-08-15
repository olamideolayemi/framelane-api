package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID  string `json:"uid"`
	IsAdmin bool `json:"adm"`
	jwt.RegisteredClaims
}

func MakeToken(secret string, uid string, admin bool, hours int) (string, error) {
	claims := &Claims{
		UserID:  uid,
		IsAdmin: admin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(hours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}
