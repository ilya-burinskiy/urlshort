package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"net/http"
)

func ShortenURLRouter() chi.Router {
	router := chi.NewRouter()
	router.Use(middleware.AllowContentType("text/plain"))

	router.Post("/", CreateShortenedURLHandler)
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

func CreateShortenedURLHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	url := string(bytes)

	shortenedURLPath, ok := storage.Get(url)
	if !ok {
		shortenedURLPath, err = randomHex(8)
		storage.Put(url, shortenedURLPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	// TODO: maybe use some URL builder
	w.Write([]byte(config.shortenedURLBaseAddr + "/" + shortenedURLPath))
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