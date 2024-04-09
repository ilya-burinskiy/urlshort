package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ilya-burinskiy/urlshort/internal/app/compress"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
)

type contextKey string

const userIDKey contextKey = "user_id"

// Gzip compress middleware
func GzipCompress(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "gzip") {
			compressReader, err := compress.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = compressReader
			defer func() {
				if err := compressReader.Close(); err != nil {
					logger.Log.Info("compressed reader middleware", zap.Error(err))
				}
			}()
		}

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") {
			responseWriterWithCompress := compress.NewWriter(w)
			w = responseWriterWithCompress
			defer func() {
				if err := responseWriterWithCompress.Close(); err != nil {
					logger.Log.Info("compressed writer middleware", zap.Error(err))
				}
			}()
		}

		h.ServeHTTP(w, r)
	})
}

// Response logging middleware
func ResponseLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := logger.LoggingResponseWriter{
			ResponseWriter: w,
			ResponseStatus: 0,
			ResponseSize:   0,
		}
		h.ServeHTTP(&lw, r)
		logger.Log.Info(
			"response",
			zap.Int("status", lw.ResponseStatus),
			zap.Int("size", lw.ResponseSize),
		)
	})
}

// Request logging middleware
func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		duration := time.Since(start)
		logger.Log.Info("got incoming HTTP request",
			zap.String("method", r.Method),
			zap.String("URI", r.RequestURI),
			zap.String("duration", duration.String()),
		)
	})
}

// Authentication middleware
func Authenticate(userAuthenticator services.UserAuthService) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			encoder := json.NewEncoder(w)
			cookie, err := r.Cookie("jwt")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				if err = encoder.Encode(err.Error()); err != nil {
					logger.Log.Info("authenticate middleware", zap.Error(err))
				}
				return
			}

			user, err := userAuthenticator.Auth(cookie.Value)
			if errors.Is(err, services.ErrInvalidJWT) {
				w.WriteHeader(http.StatusUnauthorized)
				if err = encoder.Encode("invalid JWT"); err != nil {
					logger.Log.Info("authenticate middleware", zap.Error(err))
				}
			}
			ctx := context.WithValue(r.Context(), userIDKey, user.ID)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OnlyTrustedIP
func OnlyTrustedIP(cnf configs.Config) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ipv4Net, err := net.ParseCIDR(cnf.TrustedSubnet)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			realIP := net.ParseIP(r.Header.Get("X-Real-IP"))
			if !ipv4Net.Contains(realIP) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

// Get user ID from context
func UserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDKey).(int)
	return userID, ok
}
