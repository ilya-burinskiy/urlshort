package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ilya-burinskiy/urlshort/internal/app/logger"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

// Get original URL
func (h Handlers) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	shortenedPath := chi.URLParam(r, "id")
	record, err := h.store.FindByShortenedPath(context.Background(), shortenedPath)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, fmt.Sprintf("Original URL for \"%v\" not found", shortenedPath), http.StatusBadRequest)
		return
	}

	if record.IsDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}

	http.RedirectHandler(record.OriginalURL, http.StatusTemporaryRedirect).
		ServeHTTP(w, r)
}

// Create shorened URL
func (h Handlers) CreateURL(
	shortener services.URLShortener,
	userAuthenticator services.UserAuthenticator) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		jwtStr := getJWT(r)
		user, jwtStr, err := userAuthenticator.AuthOrRegister(r.Context(), jwtStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setJWTCookie(w, jwtStr)

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		originalURL := string(bytes)

		record, err := shortener.Shortify(originalURL, user)
		if err != nil {
			var notUniqErr *storage.ErrNotUnique
			if errors.As(err, &notUniqErr) {
				w.WriteHeader(http.StatusConflict)
				if _, err = w.Write([]byte(h.config.BaseURL + "/" + notUniqErr.Record.ShortenedPath)); err != nil {
					logger.Log.Info("failed to write response", zap.Error(err))
				}
				return
			}
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if _, err = w.Write([]byte(h.config.BaseURL + "/" + record.ShortenedPath)); err != nil {
			logger.Log.Info("failed to write response", zap.Error(err))
		}
	}
}

// Create shortened URL from JSON
func (h Handlers) CreateURLFromJSON(
	shortener services.URLShortener,
	userAuthenticator services.UserAuthenticator) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var requestBody map[string]string
		encoder := json.NewEncoder(w)
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err = encoder.Encode("invalid request"); err != nil {
				logger.Log.Info("failed to encode response", zap.Error(err))
			}
			return
		}

		jwtStr := getJWT(r)
		user, jwtStr, err := userAuthenticator.AuthOrRegister(r.Context(), jwtStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setJWTCookie(w, jwtStr)

		originalURL := requestBody["url"]
		record, err := shortener.Shortify(originalURL, user)
		if err != nil {
			var notUniqErr *storage.ErrNotUnique
			if errors.As(err, &notUniqErr) {
				w.WriteHeader(http.StatusConflict)
				err = encoder.Encode(
					map[string]string{"result": h.config.BaseURL + "/" +
						notUniqErr.Record.ShortenedPath},
				)
				if err != nil {
					logger.Log.Info("failed to encode response", zap.Error(err))
				}
				return
			}
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err = encoder.Encode("could not create shortened URL"); err != nil {
				logger.Log.Info("failed to encode response", zap.Error(err))
			}
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err = encoder.Encode(map[string]string{"result": h.config.BaseURL + "/" + record.ShortenedPath}); err != nil {
			logger.Log.Info("failed to encode response", zap.Error(err))
		}
	}
}

// Create multiple shortened URLs
func (h Handlers) BatchCreateURL(
	shortener services.URLShortener,
	userAuthenticator services.UserAuthenticator) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		records := make([]models.Record, 0)
		encoder := json.NewEncoder(w)
		err := json.NewDecoder(r.Body).Decode(&records)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if err = encoder.Encode(fmt.Sprintf("failed to parse request body: %s", err.Error())); err != nil {
				logger.Log.Info("failed to encode response", zap.Error(err))
			}
			return
		}

		jwtStr := getJWT(r)
		user, jwtStr, err := userAuthenticator.AuthOrRegister(r.Context(), jwtStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setJWTCookie(w, jwtStr)

		savedRecords, err := shortener.BatchShortify(records, user)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err = encoder.Encode(err.Error()); err != nil {
				logger.Log.Info("failed to encode response", zap.Error(err))
			}
			return
		}

		type responseItem struct {
			CorrelationID string `json:"correlation_id"`
			ShortURL      string `json:"short_url"`
		}
		response := make([]responseItem, len(savedRecords))
		for i := range records {
			response[i] = responseItem{
				CorrelationID: savedRecords[i].CorrelationID,
				ShortURL:      h.config.BaseURL + "/" + savedRecords[i].ShortenedPath,
			}
		}
		w.WriteHeader(http.StatusCreated)
		if err = encoder.Encode(response); err != nil {
			logger.Log.Info("failed to encode response", zap.Error(err))
		}
	}
}

// Get user shortened URLs
func (h Handlers) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID, _ := middlewares.UserIDFromContext(r.Context())
	user := models.User{ID: userID}
	records, err := h.store.FindByUser(r.Context(), user)
	encoder := json.NewEncoder(w)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err = encoder.Encode(fmt.Sprintf("failed to fetch records: %s", err.Error())); err != nil {
			logger.Log.Info("failed to encode response", zap.Error(err))
		}
		return
	}

	if len(records) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	type responseItem struct {
		OriginalURL string `json:"original_url"`
		ShortURL    string `json:"short_url"`
	}
	response := make([]responseItem, len(records))
	for i := range records {
		response[i] = responseItem{
			OriginalURL: records[i].OriginalURL,
			ShortURL:    h.config.BaseURL + "/" + records[i].ShortenedPath,
		}
	}
	if err = encoder.Encode(response); err != nil {
		logger.Log.Info("failed to encode response", zap.Error(err))
	}
}

// Delete user shortened URLs
func (h Handlers) DeleteUserURLs(urlDeleter services.BatchDeleter) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		var shortPaths []string
		if err := json.NewDecoder(r.Body).Decode(&shortPaths); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err = encoder.Encode("invalid request body"); err != nil {
				logger.Log.Info("failed to encode response", zap.Error(err))
			}
			return
		}

		userID, _ := middlewares.UserIDFromContext(r.Context())
		for _, shortPath := range shortPaths {
			urlDeleter.Delete(models.Record{
				ShortenedPath: shortPath,
				UserID:        userID,
			})
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
