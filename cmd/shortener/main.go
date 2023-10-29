package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Storage map[string]string

var storage Storage = make(Storage)

func (s Storage) get(key string) (string, bool) {
	value, ok := s[key]
	return value, ok
}

func (s Storage) put(key, value string) {
	s[key] = value
}

func (s Storage) keyByValue(value string) (string, bool) {
	for k, v := range s {
		if v == value {
			return k, true
		}
	}
	return "", false
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			shortenURL(w, r)
		case http.MethodGet:
			getShortenURL(w, r)
		default:
			http.Error(w, "Only POST GET accepted", http.StatusBadRequest)
			w.WriteHeader(http.StatusOK)
		}
	})

	if err := http.ListenAndServe(`:8080`, mux); err != nil {
		panic(err)
	}
}

func getShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == `/` {
		http.Error(w, "Bad request",  http.StatusBadRequest)
		return
	}
	pathSplitted := strings.Split(r.URL.Path, `/`)
	if len(pathSplitted) != 2 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	shortenedPath := pathSplitted[len(pathSplitted)-1]
	originalURL, ok := storage.keyByValue(shortenedPath)
	if !ok {
		http.Error(w, fmt.Sprintf("Original URL for \"%v\" not found", shortenedPath), http.StatusBadRequest)
		return
	}
	http.RedirectHandler(originalURL, http.StatusTemporaryRedirect).ServeHTTP(w, r)
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if r.Header["Content-Type"][0] != "text/plain" {
		http.Error(w, `Only "text/plain" accepted`, http.StatusBadRequest)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	url := string(bytes)

	shortenedURL, ok := storage.get(url)
	if !ok {
		path, err := randomHex(8)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		shortenedURL = fmt.Sprintf("http://localhost:8080/%v", path)
		storage.put(url, path)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortenedURL))
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
