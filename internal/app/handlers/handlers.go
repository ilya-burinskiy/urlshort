package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/jackc/pgx/v5"
)

type Handlers struct {
	config           configs.Config
	urlCreateService services.CreateURLService
	urlDeleter       *services.BatchDeleter
	store            storage.Storage
}

func NewHandlers(
	config configs.Config,
	urlCreateSerivce services.CreateURLService,
	urlDeleter *services.BatchDeleter,
	store storage.Storage) Handlers {

	return Handlers{
		config:           config,
		urlCreateService: urlCreateSerivce,
		urlDeleter:       urlDeleter,
		store:            store,
	}

}

func (h Handlers) PingDB(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), h.config.DatabaseDSN)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	defer conn.Close(context.Background())
	w.WriteHeader(http.StatusOK)
}

func (h Handlers) GetUser(r *http.Request) (models.User, error) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		return models.User{}, err
	}

	claims := &auth.Claims{}
	token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(auth.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return models.User{}, err
	}

	return models.User{ID: claims.UserID}, nil
}

func setJWTCookie(w http.ResponseWriter, token string) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "jwt",
			Value:    token,
			MaxAge:   int(auth.TokenExp / time.Second),
			HttpOnly: true,
		},
	)
}
