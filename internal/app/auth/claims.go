package auth

import (
	"github.com/golang-jwt/jwt/v4"
)

// JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}
