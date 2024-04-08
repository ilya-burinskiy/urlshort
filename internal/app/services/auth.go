package services

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

type UserAuthService interface {
	AuthOrRegister(context.Context, string) (models.User, string, error)
}

func NewUserAuthService(store storage.Storage) UserAuthService {
	return authUserService{store: store}
}

type authUserService struct {
	store storage.Storage
}

func (a authUserService) AuthOrRegister(ctx context.Context, jwtStr string) (models.User, string, error) {
	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(jwtStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(auth.SecretKey), nil
	})
	var user models.User
	if err != nil || !token.Valid {
		newUser, err := a.store.CreateUser(ctx)
		if err != nil {
			return user, "", fmt.Errorf("failed to authenticate guest: %w", err)
		}

		newJWTStr, err := auth.BuildJWTString(newUser)
		if err != nil {
			return user, "", fmt.Errorf("failed to authenticate guest: %w", err)
		}

		user.ID = newUser.ID
		jwtStr = newJWTStr
	} else {
		user.ID = claims.UserID
	}

	return user, jwtStr, nil
}
