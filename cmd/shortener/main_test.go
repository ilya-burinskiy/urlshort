package main

import (
	"errors"
	"io"
	"net/http"
	"strings"

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

func TestCreateShortenedURLHandler(t *testing.T) {
	oldRandomHexImpl := randomHexImpl
	defer func() { randomHexImpl = oldRandomHexImpl }()
	successfulRandomHexImpl := func(n int) (string, error) { return "123", nil }
	unsuccessfulRandomHexImpl := func(n int) (string, error) { return "", errors.New("error") }

	testServer := httptest.NewServer(ShortenURLRouter())
	defer testServer.Close()

	testCases := []struct {
		name          string
		httpMethod    string
		path          string
		contentType   string
		storage       Storage
		randomHexImpl func(int) (string, error)
		want          want
	}{
		{
			name:          "responses with created status",
			httpMethod:    http.MethodPost,
			path:          "/",
			contentType:   "text/plain",
			storage:       make(Storage),
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
			storage:       make(Storage),
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
			storage:       make(Storage),
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
			storage:       make(Storage),
			randomHexImpl: unsuccessfulRandomHexImpl,
			want: want{
				code:        http.StatusUnprocessableEntity,
				response:    "error\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tc := range testCases {
		randomHexImpl = tc.randomHexImpl
		storage = tc.storage

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
	}
}

func TestGetShortenedURLHandler(t *testing.T) {
	testCases := []struct {
		name        string
		httpMethod  string
		path        string
		contentType string
		storage     Storage
		want        want
	}{
		{
			name:        "responses with ok status",
			httpMethod:  http.MethodGet,
			path:        "/123",
			contentType: "text/plain",
			storage:     Storage{"http://example.com": "123"},
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "<a href=\"http://example.com\">Temporary Redirect</a>.\n\n",
				contentType: "text/html; charset=utf-8",
			},
		},
		{
			name:        "responses with bad request if method is not GET",
			httpMethod:  http.MethodPost,
			path:        "/123",
			contentType: "text/plain",
			storage:     Storage{"http://example.com": "123"},
			want: want{
				code:        http.StatusBadRequest,
				response:    "Only GET accepted\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        `responses with bad request if path not "/{id}"`,
			httpMethod:  http.MethodGet,
			path:        "/",
			contentType: "text/plain",
			storage:     Storage{"http://examample.com": "123"},
			want: want{
				code:        http.StatusBadRequest,
				response:    "Bad request\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "responses with bad request if original URL could nog be found",
			httpMethod:  http.MethodGet,
			path:        "/321",
			contentType: "text/plain",
			storage:     Storage{"http://example.com": "123"},
			want: want{
				code:        http.StatusBadRequest,
				response:    "Original URL for \"321\" not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tc := range testCases {
		storage = tc.storage

		request := httptest.NewRequest(
			tc.httpMethod,
			tc.path,
			strings.NewReader("http://example.com"),
		)
		request.Header.Set("Content-Type", tc.contentType)
		recorder := httptest.NewRecorder()

		GetShortenedURLHandler(recorder, request)
		response := recorder.Result()
		resBody, err := io.ReadAll(response.Body)
		defer response.Body.Close()

		assert.Equal(t, tc.want.code, response.StatusCode)
		assert.NoError(t, err)
		assert.Equal(t, tc.want.response, string(resBody))
		assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))
	}
}
