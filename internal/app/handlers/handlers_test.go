package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/golang/mock/gomock"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"

	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type want struct {
	code        int
	response    string
	contentType string
}

type mockRandHexStringGenerator struct{ mock.Mock }

func (m *mockRandHexStringGenerator) Call(n int) (string, error) {
	args := m.Called(n)
	return args.String(0), args.Error(1)
}

var defaultConfig = configs.Config{
	ShortenedURLBaseAddr: "http://localhost:8080",
	ServerAddress:        "http://localhost:8080",
	FileStoragePath:      "storage",
}

func TestCreateShortenedURLHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		GetShortenedPath(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return("", storage.ErrNotFound)
	storageMock.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	generatorMock := new(mockRandHexStringGenerator)
	testServer := httptest.NewServer(ShortenURLRouter(defaultConfig, generatorMock, storageMock))
	defer testServer.Close()

	type generatorCallResult struct {
		returnValue string
		error       error
	}
	testCases := []struct {
		name                string
		httpMethod          string
		path                string
		contentType         string
		generatorCallResult generatorCallResult
		want                want
	}{
		{
			name:                "responses with created status",
			httpMethod:          http.MethodPost,
			path:                "/",
			contentType:         "text/plain",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusCreated,
				response:    "http://localhost:8080/123",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:                "responses with method not allowed if method is not POST",
			httpMethod:          http.MethodGet,
			path:                "/",
			contentType:         "text/plain",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name:                `responses with bad request if content-type is not "text/plain"`,
			httpMethod:          http.MethodPost,
			path:                "/",
			contentType:         "application/json",
			generatorCallResult: generatorCallResult{returnValue: "123", error: nil},
			want: want{
				code:        http.StatusUnsupportedMediaType,
				response:    "",
				contentType: "",
			},
		},
		{
			name:                "responses with unprocessable entity status",
			httpMethod:          http.MethodPost,
			path:                "/",
			contentType:         "text/plain",
			generatorCallResult: generatorCallResult{returnValue: "", error: errors.New("error")},
			want: want{
				code:        http.StatusUnprocessableEntity,
				response:    "error\n",
				contentType: "text/plain; charset=utf-8",
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
				strings.NewReader("http://example.com"),
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", tc.contentType)
			request.Header.Set("Accept-Encoding", "identity")

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

func TestCreateShortenedURLFromJSONHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	storageMock.EXPECT().
		GetShortenedPath(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return("", storage.ErrNotFound)
	storageMock.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	generatorMock := new(mockRandHexStringGenerator)
	testServer := httptest.NewServer(ShortenURLRouter(defaultConfig, generatorMock, storageMock))
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

func TestGetShortenedURLHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	gomock.InOrder(
		storageMock.EXPECT().GetOriginalURL(gomock.Any(), gomock.Any()).Return("http://example.com", nil),
		storageMock.EXPECT().GetOriginalURL(gomock.Any(), gomock.Any()).Return("", storage.ErrNotFound),
	)

	generatorMock := new(mockRandHexStringGenerator)
	testServer := httptest.NewServer(ShortenURLRouter(defaultConfig, generatorMock, storageMock))
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
			defer response.Body.Close()

			assert.Equal(t, tc.want.code, response.StatusCode)
			assert.NoError(t, err)
			assert.Equal(t, tc.want.response, string(resBody))
			assert.Equal(t, tc.want.contentType, response.Header.Get("Content-Type"))

		})
	}
}
