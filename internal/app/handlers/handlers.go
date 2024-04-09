package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

// Handlers
type Handlers struct {
	store  storage.Storage
	config configs.Config
}

// New handlers
func NewHandlers(
	config configs.Config,
	store storage.Storage) Handlers {

	return Handlers{
		config: config,
		store:  store,
	}

}

// Ping database
func (h Handlers) PingDB(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), h.config.DatabaseDSN)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err = w.Write([]byte(err.Error())); err != nil {
			logger.Log.Info("failed to write response", zap.Error(err))
		}

		return
	}

	defer func() {
		if err := conn.Close(context.Background()); err != nil {
			logger.Log.Info("failed close db connection", zap.Error(err))
		}
	}()
	w.WriteHeader(http.StatusOK)
}

// GetStats
func (h Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	usersCount, err := h.store.UsersCount(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	urlsCount, err := h.store.URLsCount(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(map[string]int{
		"urls":  urlsCount,
		"users": usersCount,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Log.Info("failed to encode response", zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(response); err != nil {
		logger.Log.Info("failed to write response", zap.Error(err))
	}
}

// Get user from jwt cookie
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
