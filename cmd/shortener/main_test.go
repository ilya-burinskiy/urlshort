package main

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
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
				contentType: "text/plain",
			},
		},
		{
			name:          "responses with bad request if method is not POST",
			httpMethod:    http.MethodGet,
			path:          "/",
			contentType:   "text/plain",
			storage:       make(Storage),
			randomHexImpl: successfulRandomHexImpl,
			want: want{
				code:        http.StatusBadRequest,
				response:    "Only POST accepted\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:          `responses with bad request if path does not equal "/"`,
			httpMethod:    http.MethodPost,
			path:          "/abc",
			contentType:   "text/plain",
			storage:       make(Storage),
			randomHexImpl: successfulRandomHexImpl,
			want: want{
				code:        http.StatusBadRequest,
				response:    "Bad request\n",
				contentType: "text/plain; charset=utf-8",
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
				code:        http.StatusBadRequest,
				response:    "Only \"text/plain\" accepted\n",
				contentType: "text/plain; charset=utf-8",
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
		request := httptest.NewRequest(
			tc.httpMethod,
			tc.path,
			strings.NewReader("http://example.com"),
		)
		request.Header.Set("Content-Type", tc.contentType)
		recorder := httptest.NewRecorder()

		CreateShortenedURLHandler(tc.storage)(recorder, request)
		response := recorder.Result()
		resBody, err := io.ReadAll(response.Body)
		defer response.Body.Close()

		assert.Equal(t, tc.want.code, response.StatusCode)
		assert.NoError(t, err)
		assert.Equal(t, tc.want.response, string(resBody))
		assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))
	}
}
