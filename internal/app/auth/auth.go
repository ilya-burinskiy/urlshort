package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

const TokenExp = time.Hour * 3
const SecretKey = "secret"

func BuildJWTString(user models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: user.ID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
