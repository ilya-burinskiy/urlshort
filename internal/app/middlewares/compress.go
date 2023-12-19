package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ilya-burinskiy/urlshort/internal/app/compress"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"go.uber.org/zap"
)

const TokenExp = time.Hour * 3
const SecretKey = "secret"

func GzipCompress(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "gzip") {
			compressReader, err := compress.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = compressReader
			defer compressReader.Close()
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") {
			responseWriterWithCompress := compress.NewWriter(w)
			w = responseWriterWithCompress
			defer responseWriterWithCompress.Close()
		}

		h.ServeHTTP(w, r)
	}
}

func ResponseLogger(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := logger.LoggingResponseWriter{
			ResponseWriter: w,
			ResponseStatus: 0,
			ResponseSize:   0,
		}
		h(&lw, r)
		logger.Log.Info(
			"response",
			zap.Int("status", lw.ResponseStatus),
			zap.Int("size", lw.ResponseSize),
		)
	})
}

func RequestLogger(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h(w, r)
		duration := time.Since(start)
		logger.Log.Info("got incoming HTTP request",
			zap.String("method", r.Method),
			zap.String("URI", r.RequestURI),
			zap.String("duration", duration.String()),
		)
	})
}

func CookieAuth(s storage.Storage) func(h http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("jwt")
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
					cookie, err := generateCookie(s)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					http.SetCookie(w, cookie)
					h(w, r)
					return
				}

				http.Error(w, fmt.Sprintf("failed to get JWT: %s", err.Error()), http.StatusInternalServerError)
				return
			}

			claims := &models.Claims{}
			token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(SecretKey), nil
			})
			if err != nil || !token.Valid {
				cookie, err := generateCookie(s)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, cookie)
			}

			h(w, r)
		})
	}
}

func generateCookie(s storage.Storage) (*http.Cookie, error) {
	token, err := buildJWTString(s)
	if err != nil {
		return nil, fmt.Errorf("could not generate cookie: %s", err.Error())
	}

	return &http.Cookie{
		Name:     "jwt",
		Value:    token,
		MaxAge:   int(TokenExp / time.Second),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

func buildJWTString(s storage.Storage) (string, error) {
	user, err := s.CreateUser(context.Background())
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
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
