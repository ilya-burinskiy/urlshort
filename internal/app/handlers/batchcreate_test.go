package handlers_test

import (
	"errors"
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
	"github.com/ilya-burinskiy/urlshort/internal/app/middlewares"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"
	"github.com/ilya-burinskiy/urlshort/internal/app/storage/mocks"
)

func TestBatchCreateURLTest(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := mocks.NewMockStorage(ctrl)
	user := models.User{ID: 1}
	storageMock.EXPECT().
		CreateUser(gomock.Any()).
		AnyTimes().
		Return(user, nil)
	createrMock := new(urlCreaterMock)

	router := chi.NewRouter()
	handler := handlers.NewHandlers(defaultConfig, storageMock)
	router.Use(
		middlewares.ResponseLogger,
		middlewares.RequestLogger,
		middlewares.GzipCompress,
		middleware.AllowContentEncoding("gzip"),
		middleware.AllowContentType("application/json", "application/x-gzip"),
	)
	router.Post("/api/shorten/batch", handler.BatchCreateURL(createrMock))
	testServer := httptest.NewServer(router)
	defer testServer.Close()

	testCases := []struct {
		name              string
		batchCreateResult urlCreaterBatchCreateResult
		reqBody           string
		want              want
	}{
		{
			name: "responses with created status",
			batchCreateResult: urlCreaterBatchCreateResult{
				returnValue: []models.Record{
					{OriginalURL: "http://example0.com", CorrelationID: "1", ShortenedPath: "1"},
					{OriginalURL: "http://example1.com", CorrelationID: "2", ShortenedPath: "2"},
				},
			},
			reqBody: toJSON(
				t,
				[]models.Record{
					{OriginalURL: "http://example0.com", CorrelationID: "1"},
					{OriginalURL: "http://example1.com", CorrelationID: "2"},
				},
			),
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				response: toJSON(
					t,
					[]map[string]string{
						{"correlation_id": "1", "short_url": defaultConfig.ShortenedURLBaseAddr + "/" + "1"},
						{"correlation_id": "2", "short_url": defaultConfig.ShortenedURLBaseAddr + "/" + "2"},
					},
				) + "\n",
			},
		},
		{
			name: "responses with bad request status if could not create batch",
			batchCreateResult: urlCreaterBatchCreateResult{
				err: errors.New("error"),
			},
			reqBody: toJSON(
				t,
				[]models.Record{
					{OriginalURL: "http://example0.com", CorrelationID: "1"},
					{OriginalURL: "http://example1.com", CorrelationID: "2"},
				},
			),
			want: want{
				code:        http.StatusUnprocessableEntity,
				contentType: "application/json; charset=utf-8",
				response:    toJSON(t, "error") + "\n",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCall := createrMock.On("BatchCreate", mock.Anything, mock.Anything).
				Return(tc.batchCreateResult.returnValue, tc.batchCreateResult.err)
			defer mockCall.Unset()

			request, err := http.NewRequest(
				http.MethodPost,
				testServer.URL+"/api/shorten/batch",
				strings.NewReader(tc.reqBody),
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept-Encoding", "identity")

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
