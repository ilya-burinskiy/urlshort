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

type Storage map[string]string

func (s Storage) Get(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

func (s Storage) Put(key, value string) {
	s[key] = value
}

func (s Storage) KeyByValue(value string) (string, bool) {
	for k, v := range s {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func (s Storage) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// NOTE: to mock randomHex in tests
var randomHexImpl = func(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

var storage = make(Storage)

func main() {
	if err := http.ListenAndServe(`:8080`, ShortenURLRouter()); err != nil {
		panic(err)
	}
}

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

	shortenedURL, ok := storage.Get(url)
	if !ok {
		path, err := randomHex(8)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		shortenedURL = fmt.Sprintf("http://localhost:8080/%v", path)
		storage.Put(url, path)
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortenedURL))
}

func randomHex(n int) (string, error) {
	return randomHexImpl(n)
}
