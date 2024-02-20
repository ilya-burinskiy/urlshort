package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
)

func BenchmarkCreateShortenedURLHandler(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	generatorMock := new(mockRandHexStringGenerator)
	urlCreateService := services.NewCreateURLService(8, generatorMock, storageMock)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("text/plain", "application/x-gzip"),
	)
	router.Post("/", handler.CreateURL(urlCreateService))
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		request, err := http.NewRequest(http.MethodPost, testServer.URL+"/", strings.NewReader(fmt.Sprintf("http://example%d.com", i)))
		require.NoError(b, err)
		b.StartTimer()

		response, err := testServer.Client().Do(request)
		b.StopTimer()
		require.NoError(b, err)
		response.Body.Close()
	}
}

func BenchmarkCreateURLFromJSON(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	generatorMock := new(mockRandHexStringGenerator)
	urlCreateService := services.NewCreateURLService(8, generatorMock, storageMock)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("application/json", "application/x-gzip"),
	)
	router.Post("/api/shorten", handler.CreateURL(urlCreateService))
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		request, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/",
			strings.NewReader(
				toJSON(b, map[string]string{"url": fmt.Sprintf("http://example%d.com", i)}),
			),
		)
		require.NoError(b, err)
		b.StartTimer()

		response, err := testServer.Client().Do(request)
		b.StopTimer()
		require.NoError(b, err)
		response.Body.Close()
	}
}

func BenchmarkGetOriginalURLHandler(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("text/plain", "application/x-gzip"),
	)
	router.Get("/api/shorten", handler.GetOriginalURL)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		request, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/123",
			nil,
		)
		require.NoError(b, err)
		b.StartTimer()

		response, err := testServer.Client().Do(request)
		b.StopTimer()
		require.NoError(b, err)
		response.Body.Close()
	}
}

func toJSON(b *testing.B, v interface{}) string {
	result, err := json.Marshal(v)
	require.NoError(b, err)

	return string(result)
}
