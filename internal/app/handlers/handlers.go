package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
)

func ShortenURLRouter(
	config configs.Config,
	rndGen services.RandHexStringGenerator,
	storage storage.Storage) chi.Router {

	router := chi.NewRouter()
	router.Use(middleware.AllowContentType("text/plain"))

	router.Post("/", CreateShortenedURLHandler(config, rndGen, storage))
	router.Get("/{id}", GetShortenedURLHandler(storage))

	return router
}

func GetShortenedURLHandler(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortenedPath := chi.URLParam(r, "id")
		originalURL, ok := storage.KeyByValue(shortenedPath)
		if !ok {
			http.Error(w, fmt.Sprintf("Original URL for \"%v\" not found", shortenedPath), http.StatusBadRequest)
			return
		}
		http.RedirectHandler(originalURL, http.StatusTemporaryRedirect).ServeHTTP(w, r)
	}
}

func CreateShortenedURLHandler(
	config configs.Config,
	rndGen services.RandHexStringGenerator,
	storage storage.Storage) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		originalURL := string(bytes)

		shortenedURL, err := services.CreateShortenedURLService(
			originalURL,
			config.ShortenedURLBaseAddr,
			8,
			rndGen,
			storage,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortenedURL))
	}
}
