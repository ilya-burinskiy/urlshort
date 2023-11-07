package main

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ilya-burinskiy/urlshort/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"

	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type want struct {
	code        int
	response    string
	contentType string
}

var defaultConfig = configs.Config{
	ShortenedURLBaseAddr: "http://localhost:8080",
	ServerAddress:        "http://localhost:8080",
}

func TestCreateShortenedURLHandler(t *testing.T) {
	oldRandomHexImpl := randomHexImpl
	defer func() { randomHexImpl = oldRandomHexImpl }()
	successfulRandomHexImpl := func(n int) (string, error) { return "123", nil }
	unsuccessfulRandomHexImpl := func(n int) (string, error) { return "", errors.New("error") }

	testServer := httptest.NewServer(ShortenURLRouter(defaultConfig))
	defer testServer.Close()

	testCases := []struct {
		name          string
		httpMethod    string
		path          string
		contentType   string
		randomHexImpl func(int) (string, error)
		want          want
	}{
		{
			name:          "responses with created status",
			httpMethod:    http.MethodPost,
			path:          "/",
			contentType:   "text/plain",
			randomHexImpl: successfulRandomHexImpl,
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/123",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:          "responses with method not allowed if method is not POST",
			httpMethod:    http.MethodGet,
			path:          "/",
			contentType:   "text/plain",
			randomHexImpl: successfulRandomHexImpl,
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name:          `responses with bad request if content-type is not "text/plain"`,
			httpMethod:    http.MethodPost,
			path:          "/",
			contentType:   "application/json",
			randomHexImpl: successfulRandomHexImpl,
			want: want{
				code:        http.StatusUnsupportedMediaType,
				response:    "",
				contentType: "",
			},
		},
		{
			name:          "responses with unprocessable entity status",
			httpMethod:    http.MethodPost,
			path:          "/",
			contentType:   "text/plain",
			randomHexImpl: unsuccessfulRandomHexImpl,
			want: want{
				code:        http.StatusUnprocessableEntity,
				response:    "error\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			randomHexImpl = tc.randomHexImpl
			defer storage.Clear()

			request, err := http.NewRequest(
				tc.httpMethod,
				testServer.URL+tc.path,
				strings.NewReader("http://example.com"),
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", tc.contentType)

			response, err := testServer.Client().Do(request)
			require.NoError(t, err)
			resBody, err := io.ReadAll(response.Body)
			defer response.Body.Close()

			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.response, string(resBody))
			assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))

		})
	}
}

func TestGetShortenedURLHandler(t *testing.T) {
	testServer := httptest.NewServer(ShortenURLRouter(defaultConfig))
	defer testServer.Close()

	testCases := []struct {
		name         string
		httpMethod   string
		path         string
		contentType  string
		existingURLs map[string]string
		want         want
	}{
		{
			name:         "responses with temporary redirect status",
			httpMethod:   http.MethodGet,
			path:         "/123",
			contentType:  "text/plain",
			existingURLs: map[string]string{"http://example.com": "123"},
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "<a href=\"http://example.com\">Temporary Redirect</a>.\n\n",
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name:         "responses with method not allowed if method is not GET",
			httpMethod:   http.MethodPost,
			path:         "/123",
			contentType:  "text/plain",
			existingURLs: map[string]string{"http://example.com": "123"},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name:         "responses with bad request if original URL could nog be found",
			httpMethod:   http.MethodGet,
			path:         "/321",
			contentType:  "text/plain",
			existingURLs: map[string]string{"http://example.com": "123"},
			want: want{
				code:        http.StatusBadRequest,
				response:    "Original URL for \"321\" not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for origURL, shortenedPath := range tc.existingURLs {
				storage.Put(origURL, shortenedPath)
			}
			defer storage.Clear()

			request, err := http.NewRequest(
				tc.httpMethod,
				testServer.URL+tc.path,
				nil,
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", tc.contentType)

			transport := http.Transport{}
			response, err := transport.RoundTrip(request)
			require.NoError(t, err)
			resBody, err := io.ReadAll(response.Body)
			defer response.Body.Close()

			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.response, string(resBody))
			assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))

		})
	}
}
