package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
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

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, shortenURL)

	if err := http.ListenAndServe(`:8080`, mux); err != nil {
		panic(err)
	}
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST accepted", http.StatusBadRequest)
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
