package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
)

func TestGetUserURLsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	user := models.User{ID: 1}
	storageMock.EXPECT().
		FindByUser(gomock.Any(), user).
		AnyTimes().
		Return(
			[]models.Record{
				{OriginalURL: "http://example1.com", ShortenedPath: "1"},
				{OriginalURL: "http://example2.com", ShortenedPath: "2"},
			},
			nil,
		)

	router := chi.NewRouter()
	handler := handlers.NewHandlers(defaultConfig, storageMock)
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

	authCookie := generateAuthCookie(t, user)
	testCases := []struct {
		name       string
		authCookie *http.Cookie
		want       want
	}{
		{
			name:       "responses with ok status",
			authCookie: authCookie,
			want: want{
				code:        http.StatusOK,
				contentType: "application/json; charset=utf-8",
				response: toJSON(
					t,
					[]map[string]string{
						{"short_url": defaultConfig.ShortenedURLBaseAddr + "/1", "original_url": "http://example1.com"},
						{"short_url": defaultConfig.ShortenedURLBaseAddr + "/2", "original_url": "http://example2.com"},
					},
				) + "\n",
			},
		},
		{
			name:       "responses with unauthorized status",
			authCookie: &http.Cookie{},
			want: want{
				code:     http.StatusUnauthorized,
				response: toJSON(t, "http: named cookie not present") + "\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(
				http.MethodGet,
				testServer.URL+"/api/user/urls",
				nil,
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept-Encoding", "identity")
			request.AddCookie(tc.authCookie)

			response, err := testServer.Client().Do(request)
			require.NoError(t, err)
			defer response.Body.Close()

			resBody, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.Equal(t, tc.want.response, string(resBody))
		})
	}
}
