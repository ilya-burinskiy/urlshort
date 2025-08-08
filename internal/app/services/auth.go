package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
)

var ErrInvalidJWT = errors.New("invalid JWT")

type UserAuthenticator interface {
	AuthOrRegister(context.Context, string) (models.User, string, error)
	Auth(string) (models.User, error)
}


type UserCreator interface {
	CreateUser(ctx context.Context) (models.User, error)
}

type authUserService struct {
	usrCreator UserCreator
}

func NewUserAuthenticator(usrCreator UserCreator) UserAuthenticator {
	return authUserService{usrCreator: usrCreator}
}

func (a authUserService) AuthOrRegister(ctx context.Context, jwtStr string) (models.User, string, error) {
	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(jwtStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(auth.SecretKey), nil
	})
	var user models.User
	if err != nil || !token.Valid {
		newUser, err := a.usrCreator.CreateUser(ctx)
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

func (a authUserService) Auth(jwtStr string) (models.User, error) {
	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(jwtStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(auth.SecretKey), nil
	})
	var user models.User
	if err != nil || !token.Valid {
		return user, ErrInvalidJWT
	}

	return user, nil
}
