package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ilya-burinskiy/urlshort/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func ShortenURLRouter(config configs.Config) chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.AllowContentType("text/plain"))

	router.Post("/", CreateShortenedURLHandler(config))
	router.Get("/{id}", GetShortenedURLHandler)

	return router
}

func GetShortenedURLHandler(w http.ResponseWriter, r *http.Request) {
	shortenedPath := chi.URLParam(r, "id")
	originalURL, ok := storage.KeyByValue(shortenedPath)
	if !ok {
		http.Error(w, fmt.Sprintf("Original URL for \"%v\" not found", shortenedPath), http.StatusBadRequest)
		return
	}
	http.RedirectHandler(originalURL, http.StatusTemporaryRedirect).ServeHTTP(w, r)
}

func CreateShortenedURLHandler(config configs.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		url := string(bytes)

		shortenedURLPath, ok := storage.Get(url)
		if !ok {
			shortenedURLPath, err = randomHex(8)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}

			storage.Put(url, shortenedURLPath)
		}

		w.WriteHeader(http.StatusCreated)
		// TODO: maybe use some URL builder
		w.Write([]byte(config.ShortenedURLBaseAddr + "/" + shortenedURLPath))
	}
}

func randomHex(n int) (string, error) {
	return randomHexImpl(n)
}

// NOTE: to mock randomHex in tests
var randomHexImpl = func(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
