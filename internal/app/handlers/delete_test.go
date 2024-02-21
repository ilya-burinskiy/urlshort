package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang/mock/gomock"
	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type urlDeleterMock struct{ mock.Mock }

func (m *urlDeleterMock) Delete(r models.Record) {}
func (m *urlDeleterMock) Run()                   {}

func TestDeleterUserURLsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	urlDeleterMock := new(urlDeleterMock)
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
	router.Delete("/api/user/urls", handler.DeleteUserURLs(urlDeleterMock))
	testServer := httptest.NewServer(router)

	authCookie := generateAuthCookie(t, models.User{ID: 1})
	testCases := []struct {
		name       string
		authCookie *http.Cookie
		reqBody    string
		want       want
	}{
		{
			name:       "responses with accepted status",
			authCookie: authCookie,
			reqBody:    toJSON(t, []string{"1", "2"}),
			want: want{
				code:        http.StatusAccepted,
				contentType: "applicationg/json; charset=utf-8",
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
				http.MethodDelete,
				testServer.URL+"/api/user/urls",
				strings.NewReader(tc.reqBody),
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
