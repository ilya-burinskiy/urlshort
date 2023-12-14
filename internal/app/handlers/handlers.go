package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/jackc/pgx/v5"
)

func ShortenURLRouter(
	config configs.Config,
	rndGen services.RandHexStringGenerator,
	urlsStorage storage.Storage) chi.Router {

	router := chi.NewRouter()
	handlers := handlers{
		config:      config,
		rndGen:      rndGen,
		urlsStorage: urlsStorage,
	}
	router.Use(
		handlerFunc2Handler(middlewares.ResponseLogger),
		handlerFunc2Handler(middlewares.RequestLogger),
		handlerFunc2Handler(middlewares.GzipCompress),
		middleware.AllowContentEncoding("gzip"),
	)
	router.Group(func(router chi.Router) {
		router.Use(middleware.AllowContentType("text/plain", "application/x-gzip"))
		router.Post("/", handlers.create)
		router.Get("/{id}", handlers.get)
		router.Get("/ping", handlers.pingDB)
	})
	router.Group(func(router chi.Router) {
		router.Use(middleware.AllowContentType("application/json", "application/x-gzip"))
		router.Post("/api/shorten", handlers.createFromJSON)
		router.Post("/api/shorten/batch", handlers.batchCreateFromJSON)
	})

	return router
}

type handlers struct {
	config      configs.Config
	rndGen      services.RandHexStringGenerator
	urlsStorage storage.Storage
}

func (h handlers) get(w http.ResponseWriter, r *http.Request) {
	shortenedPath := chi.URLParam(r, "id")
	record, err := h.urlsStorage.FindByShortenedPath(context.Background(), shortenedPath)
	if errors.Is(err, storage.ErrNotFound) {
		http.Error(w, fmt.Sprintf("Original URL for \"%v\" not found", shortenedPath), http.StatusBadRequest)
		return
	}
	http.RedirectHandler(record.OriginalURL, http.StatusTemporaryRedirect).
		ServeHTTP(w, r)
}

func (h handlers) create(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	originalURL := string(bytes)

	record, err := services.Create(originalURL, 8, h.rndGen, h.urlsStorage)
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

	originalURL := requestBody["url"]
	record, err := services.Create(originalURL, 8, h.rndGen, h.urlsStorage)
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

func (h handlers) batchCreateFromJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	records := make([]models.Record, 0)
	encoder := json.NewEncoder(w)
	err := json.NewDecoder(r.Body).Decode(&records)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("failed to parse request body: %s", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	err = services.BatchCreate(records, 8, h.rndGen, h.urlsStorage)
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

func (h handlers) pingDB(w http.ResponseWriter, r *http.Request) {
	conn, err := pgx.Connect(context.Background(), h.config.DatabaseDSN)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	defer conn.Close(context.Background())
	w.WriteHeader(http.StatusOK)
}

func handlerFunc2Handler(f func(http.HandlerFunc) http.HandlerFunc) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return f(h.(http.HandlerFunc))
	}
}
