package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"

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

func toJSON(t require.TestingT, v interface{}) string {
	result, err := json.Marshal(v)
	require.NoError(t, err)

	return string(result)
}

type urlCreaterMock struct{ mock.Mock }

func (m *urlCreaterMock) Create(originalURL string, user models.User) (models.Record, error) {
	args := m.Called(originalURL, user)
	return args.Get(0).(models.Record), args.Error(1)
}

func (m *urlCreaterMock) BatchCreate(records []models.Record, user models.User) ([]models.Record, error) {
	args := m.Called(records, user)
	return args.Get(0).([]models.Record), args.Error(1)
}

type urlCreaterBatchCreateResult struct {
	returnValue []models.Record
	err         error
}

func generateAuthCookie(t *testing.T, user models.User) *http.Cookie {
	jwtStr, err := auth.BuildJWTString(user)
	require.NoError(t, err)

	return &http.Cookie{
		Name:     "jwt",
		Value:    jwtStr,
		HttpOnly: true,
	}
}
