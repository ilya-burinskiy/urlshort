package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
)

func BenchmarkCreateShortenedURLHandler(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		FindByOriginalURL(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(models.Record{}, storage.ErrNotFound)
	storageMock.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	urlCreateService := services.NewCreateURLService(8, services.StdRandHexStringGenerator{}, storageMock)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).CreateURL(urlCreateService),
	)

	authCookie := generateAuthCookie(b, models.User{ID: 1})
	request, err := http.NewRequest(http.MethodPost, "/", strings.NewReader("http://example.com"))
	require.NoError(b, err)
	request.AddCookie(authCookie)
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkCreateURLFromJSON(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		FindByOriginalURL(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(models.Record{}, storage.ErrNotFound)
	storageMock.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	urlCreateService := services.NewCreateURLService(8, services.StdRandHexStringGenerator{}, storageMock)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).CreateURL(urlCreateService),
	)

	authCookie := generateAuthCookie(b, models.User{ID: 1})
	reqBody := toJSON(b, map[string]string{"url": "http://example.com"})
	request, err := http.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	require.NoError(b, err)
	request.AddCookie(authCookie)
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkGetOriginalURLHandler(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		FindByShortenedPath(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(models.Record{OriginalURL: "http://example.com"}, nil)

	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).GetOriginalURL,
	)
	request, err := http.NewRequest(http.MethodPost, "/123", nil)
	require.NoError(b, err)
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkBatchCreateURL(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	urlCreateService := new(urlCreaterMock)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).BatchCreateURL(urlCreateService),
	)

	n := 100
	records := make([]models.Record, n)
	for i := 0; i < n; i++ {
		records[i] = models.Record{
			UserID:        1,
			OriginalURL:   fmt.Sprintf("http://example%d.com", i),
			CorrelationID: strconv.Itoa(i),
		}
	}
	urlCreateService.On("BatchCreate", mock.Anything, mock.Anything).Return(records, nil)

	reqBody := toJSON(b, records)
	authCookie := generateAuthCookie(b, models.User{ID: 1})
	request, err := http.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(reqBody))
	require.NoError(b, err)
	request.AddCookie(authCookie)
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkGetUserURLs(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("application/json", "application/x-gzip"),
		middlewares.Authenticate,
	)
	router.Get("/api/user/urls", handler.GetUserURLs)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	authCookie := generateAuthCookie(b, models.User{ID: 1})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		request, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/api/user/urls",
			nil,
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept-Encoding", "identity")
		request.AddCookie(authCookie)
		require.NoError(b, err)
		b.StartTimer()

		response, err := testServer.Client().Do(request)
		b.StopTimer()
		require.NoError(b, err)
		response.Body.Close()
	}
}

func BenchmarkDeleteUserURLs(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	urlDeleter := new(urlDeleterMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("application/json", "application/x-gzip"),
		middlewares.Authenticate,
	)
	router.Delete("/api/user/urls", handler.DeleteUserURLs(urlDeleter))
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	records := make([]models.Record, 1000)
	authCookie := generateAuthCookie(b, models.User{ID: 1})
	for i := 0; i < 100; i++ {
		records[i] = models.Record{
			UserID:        1,
			OriginalURL:   fmt.Sprintf("http://example%d.com", i),
			CorrelationID: strconv.Itoa(i),
		}
	}
	reqBody := toJSON(b, records)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		request, err := http.NewRequest(
			http.MethodPost,
			testServer.URL+"/api/user/urls",
			strings.NewReader(reqBody),
		)
		require.NoError(b, err)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept-Encoding", "identity")
		request.AddCookie(authCookie)
		b.StartTimer()

		response, err := testServer.Client().Do(request)
		b.StopTimer()
		require.NoError(b, err)
		response.Body.Close()
	}
}
