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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ilya-burinskiy/urlshort/internal/app/handlers"
	"github.com/ilya-burinskiy/urlshort/internal/app/services"
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
)

func TestDeleterUserURLsHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	urlDeleter := services.NewDeferredDeleter(storageMock)
	userAuthenticator := new(userAuthenticatorMock)
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router := chi.NewRouter()
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("application/json", "application/x-gzip"),
		middlewares.Authenticate(userAuthenticator),
	)
	router.Delete("/api/user/urls", handler.DeleteUserURLs(urlDeleter))
	testServer := httptest.NewServer(router)

	user := models.User{ID: 1}
	authCookie := generateAuthCookie(t, user)
	testCases := []struct {
		name       string
		authCookie *http.Cookie
		authResult authResult
		reqBody    string
		want       want
	}{
		{
			name:       "responses with accepted status",
			authCookie: authCookie,
			authResult: authResult{user: user},
			reqBody:    toJSON(t, []string{"1", "2"}),
			want: want{
				code:        http.StatusAccepted,
				contentType: "applicationg/json; charset=utf-8",
			},
		},
		{
			name:       "responses with unauthorized status if cookie isn't present",
			authCookie: &http.Cookie{},
			want: want{
				code:     http.StatusUnauthorized,
				response: toJSON(t, "http: named cookie not present") + "\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authCall := userAuthenticator.On("Auth", mock.Anything).
				Return(tc.authResult.user, tc.authResult.err)
			defer authCall.Unset()

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
			defer func() {
				err = response.Body.Close()
				require.NoError(t, err)
			}()

			resBody, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.Equal(t, tc.want.response, string(resBody))
		})
	}
}
