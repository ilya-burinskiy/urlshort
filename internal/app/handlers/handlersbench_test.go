package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
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
	userAuthenticator := new(userAuthenticatorMock)
	userAuthenticator.On("Call", mock.Anything, mock.Anything).Return(models.User{ID: 1}, "123", nil)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).CreateURL(urlCreateService, userAuthenticator),
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

	userAuthenticator := new(userAuthenticatorMock)
	userAuthenticator.On("Call", mock.Anything, mock.Anything).Return(models.User{ID: 1}, "123", nil)
	urlCreateService := services.NewCreateURLService(8, services.StdRandHexStringGenerator{}, storageMock)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).CreateURL(urlCreateService, userAuthenticator),
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
	userAuthenticator := new(userAuthenticatorMock)
	userAuthenticator.On("Call", mock.Anything, mock.Anything).Return(models.User{ID: 1}, "123", nil)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).BatchCreateURL(urlCreateService, userAuthenticator),
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
	userRecords := make([]models.Record, 100)
	storageMock.EXPECT().
		FindByUser(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(userRecords, nil)
	handler := http.HandlerFunc(handlers.NewHandlers(defaultConfig, storageMock).GetUserURLs)

	authCookie := generateAuthCookie(b, models.User{ID: 1})
	request, err := http.NewRequest(http.MethodGet, "/api/user/urls", nil)
	require.NoError(b, err)
	request.AddCookie(authCookie)
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}

func BenchmarkDeleteUserURLs(b *testing.B) {
	ctrl := gomock.NewController(b)
	storageMock := mocks.NewMockStorage(ctrl)
	urlDeleter := new(urlDeleterMock)
	handler := http.HandlerFunc(
		handlers.NewHandlers(defaultConfig, storageMock).DeleteUserURLs(urlDeleter),
	)

	records := make([]models.Record, 100)
	authCookie := generateAuthCookie(b, models.User{ID: 1})
	for i := 0; i < 100; i++ {
		records[i] = models.Record{
			UserID:        1,
			OriginalURL:   fmt.Sprintf("http://example%d.com", i),
			CorrelationID: strconv.Itoa(i),
		}
	}
	reqBody := toJSON(b, records)
	request, err := http.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(reqBody))
	require.NoError(b, err)
	request.AddCookie(authCookie)
	recorder := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(recorder, request)
	}
}
