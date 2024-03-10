package handlers_test

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"

	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetShortenedURLHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		CreateUser(gomock.Any()).
		AnyTimes().
		Return(models.User{ID: 1}, nil)
	gomock.InOrder(
		storageMock.EXPECT().
			FindByShortenedPath(gomock.Any(), gomock.Any()).
			Return(models.Record{OriginalURL: "http://example.com"}, nil),
		storageMock.EXPECT().
			FindByShortenedPath(gomock.Any(), gomock.Any()).
			Return(models.Record{}, storage.ErrNotFound),
	)

	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("application/json", "application/x-gzip"),
	)
	router.Get("/{id}", handler.GetOriginalURL)
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	testCases := []struct {
		name        string
		httpMethod  string
		path        string
		contentType string
		want        want
	}{
		{
			name:        "responses with temporary redirect status",
			httpMethod:  http.MethodGet,
			path:        "/123",
			contentType: "text/plain",
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "<a href=\"http://example.com\">Temporary Redirect</a>.\n\n",
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name:        "responses with method not allowed if method is not GET",
			httpMethod:  http.MethodPost,
			path:        "/123",
			contentType: "text/plain",
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name:        "responses with bad request if original URL could not be found",
			httpMethod:  http.MethodGet,
			path:        "/321",
			contentType: "text/plain",
			want: want{
				code:        http.StatusBadRequest,
				response:    "Original URL for \"321\" not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(
				tc.httpMethod,
				testServer.URL+tc.path,
				nil,
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("Accept-Encoding", "identity")

			transport := http.Transport{}
			response, err := transport.RoundTrip(request)
			require.NoError(t, err)
			resBody, err := io.ReadAll(response.Body)
			defer func() {
				err = response.Body.Close()
				require.NoError(t, err)
			}()

			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.response, string(resBody))
			assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))

		})
	}
}
