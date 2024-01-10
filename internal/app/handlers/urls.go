package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func (h handlers) get(w http.ResponseWriter, r *http.Request) {
	shortenedPath := chi.URLParam(r, "id")
	record, err := h.s.FindByShortenedPath(context.Background(), shortenedPath)
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

func (h handlers) create(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUser(r)
	if err != nil {
		user, err = h.s.CreateUser(r.Context())
		if err != nil {
			http.Error(w, "failed to create user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		token, err := auth.BuildJWTString(user)
		if err != nil {
			http.Error(w, "failed to build JWT string: "+err.Error(), http.StatusInternalServerError)
			return
		}

		setJWTCookie(w, token)
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	originalURL := string(bytes)

	record, err := h.urlCreateService.Create(originalURL, user)
	if err != nil {
		var notUniqErr *storage.ErrNotUnique
		if errors.As(err, &notUniqErr) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(h.config.ShortenedURLBaseAddr + "/" + notUniqErr.Record.ShortenedPath))
			return
		}
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(h.config.ShortenedURLBaseAddr + "/" + record.ShortenedPath))
}

func (h handlers) createFromJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var requestBody map[string]string
	encoder := json.NewEncoder(w)
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder.Encode("invalid request")
		return
	}

	user, err := h.getUser(r)
	if err != nil {
		user, err = h.s.CreateUser(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode("failed to create user: " + err.Error())
			return
		}

		token, err := auth.BuildJWTString(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode("failed to build JWT string: " + err.Error())
			return
		}

		setJWTCookie(w, token)
	}
	originalURL := requestBody["url"]
	record, err := h.urlCreateService.Create(originalURL, user)
	if err != nil {
		var notUniqErr *storage.ErrNotUnique
		if errors.As(err, &notUniqErr) {
			w.WriteHeader(http.StatusConflict)
			encoder.Encode(
				map[string]string{"result": h.config.ShortenedURLBaseAddr + "/" +
					notUniqErr.Record.ShortenedPath},
			)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder.Encode("could not create shortened URL")
		return
	}

	w.WriteHeader(http.StatusCreated)
	encoder.Encode(map[string]string{"result": h.config.ShortenedURLBaseAddr + "/" + record.ShortenedPath})
}

func (h handlers) batchCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	records := make([]models.Record, 0)
	encoder := json.NewEncoder(w)
	err := json.NewDecoder(r.Body).Decode(&records)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		encoder.Encode(fmt.Sprintf("failed to parse request body: %s", err.Error()))
		return
	}

	user, err := h.getUser(r)
	if err != nil {
		user, err = h.s.CreateUser(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode("failed to create user: " + err.Error())
			return
		}

		token, err := auth.BuildJWTString(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			encoder.Encode("failed to build JWT string: " + err.Error())
			return
		}

		setJWTCookie(w, token)
	}
	err = h.urlCreateService.BatchCreate(records, user)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder.Encode(err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := make([]map[string]string, len(records))
	for i := range records {
		response[i] = map[string]string{
			"correlation_id": records[i].CorrelationID,
			"short_url":      h.config.ShortenedURLBaseAddr + "/" + records[i].ShortenedPath,
		}
	}
	w.WriteHeader(http.StatusCreated)
	encoder.Encode(response)
}

func (h handlers) getUserURLs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user, err := h.getUser(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	records, err := h.s.FindByUser(r.Context(), user)
	encoder := json.NewEncoder(w)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder.Encode(fmt.Sprintf("failed to fetch records: %s", err.Error()))
		return
	}

	if len(records) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]map[string]string, len(records))
	for i := range records {
		response[i] = map[string]string{
			"short_url":    h.config.ShortenedURLBaseAddr + "/" + records[i].ShortenedPath,
			"original_url": records[i].OriginalURL,
		}
	}
	encoder.Encode(response)
}

func (h handlers) deleteUserURLs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	var shortPaths []string
	if err := json.NewDecoder(r.Body).Decode(&shortPaths); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		encoder.Encode("invalid request body")
		return
	}

	user, err := h.getUser(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	for _, shortPath := range shortPaths {
		h.urlDeleter.Delete(models.Record{
			ShortenedPath: shortPath,
			UserID: user.ID,
		})
	}

	w.WriteHeader(http.StatusAccepted)
}