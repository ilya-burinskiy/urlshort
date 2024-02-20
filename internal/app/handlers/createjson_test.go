package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"

	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateShortenedURLFromJSONHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		FindByOriginalURL(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(models.Record{}, storage.ErrNotFound)
	storageMock.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)
	storageMock.EXPECT().
		CreateUser(gomock.Any()).
		AnyTimes().
		Return(models.User{ID: 1}, nil)

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
	router.Post("/api/shorten", handler.CreateURLFromJSON(urlCreateService))
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	toJSON := func(v interface{}) string {
		result, err := json.Marshal(v)
		require.NoError(t, err)

		return string(result)
	}

	type generatorCallResult struct {
		returnValue string
		error       error
	}
	testCases := []struct {
		name                string
		httpMethod          string
		path                string
		requestBody         string
		contentType         string
		generatorCallResult generatorCallResult
		want                want
	}{
		{
			name:                "responses with created status",
			httpMethod:          http.MethodPost,
			path:                "/api/shorten",
			requestBody:         toJSON(map[string]string{"url": "http://example.com"}),
			contentType:         "application/json",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusCreated,
				response:    toJSON(map[string]string{"result": "http://localhost:8080/123"}) + "\n",
				contentType: "application/json",
			},
		},
		{
			name:                "responses with method not allowed if method is not POST",
			httpMethod:          http.MethodGet,
			path:                "/api/shorten",
			contentType:         "application/json",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name:                `responses with bad request if content-type is not "application/json"`,
			httpMethod:          http.MethodPost,
			path:                "/api/shorten",
			requestBody:         toJSON(map[string]string{"url": "http://example.com"}),
			contentType:         "text/plain",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusUnsupportedMediaType,
				response:    "",
				contentType: "",
			},
		},
		{
			name:                "responses with unprocessable entity if in body invalid json",
			httpMethod:          http.MethodPost,
			path:                "/api/shorten",
			requestBody:         `url: http://example.com`,
			contentType:         "application/json",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusUnprocessableEntity,
				response:    toJSON("invalid request") + "\n",
				contentType: "application/json",
			},
		},
		{
			name:                "responses with unprocessable entity status if could not create shortened URL",
			httpMethod:          http.MethodPost,
			path:                "/api/shorten",
			requestBody:         toJSON(map[string]string{"url": "http://example.com"}),
			contentType:         "application/json",
			generatorCallResult: generatorCallResult{returnValue: "", error: errors.New("error")},
			want: want{
				code:        http.StatusUnprocessableEntity,
				response:    toJSON("could not create shortened URL") + "\n",
				contentType: "application/json",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCall := generatorMock.On("Call", mock.Anything).Return(
				tc.generatorCallResult.returnValue,
				tc.generatorCallResult.error,
			)
			defer mockCall.Unset()

			request, err := http.NewRequest(
				tc.httpMethod,
				testServer.URL+tc.path,
				strings.NewReader(tc.requestBody),
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("Accept-Encoding", "identity")

			response, err := testServer.Client().Do(request)
			require.NoError(t, err)
			responseBody, err := io.ReadAll(response.Body)
			defer response.Body.Close()

			assert.NoError(t, err)
			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.Equal(t, tc.want.response, string(responseBody))
			assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))
		})
	}
}
