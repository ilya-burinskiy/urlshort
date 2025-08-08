package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ilya-burinskiy/urlshort/internal/app/auth"
	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
	"github.com/ilya-burinskiy/urlshort/internal/app/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type want struct {
	response    string
	contentType string
	code        int
}

type randHexStrGeneratorMock struct{ mock.Mock }

func (m *randHexStrGeneratorMock) Gen(n int) (string, error) {
	args := m.Called(n)
	return args.String(0), args.Error(1)
}

var defaultConfig = configs.Config{
	BaseURL:         "http://localhost:8080",
	ServerAddress:   "http://localhost:8080",
	FileStoragePath: "storage",
}

func toJSON(t require.TestingT, v interface{}) string {
	result, err := json.Marshal(v)
	require.NoError(t, err)

	return string(result)
}

type urlShortenerMock struct{ mock.Mock }

func (m *urlShortenerMock) Shortify(originalURL string, user models.User) (models.Record, error) {
	args := m.Called(originalURL, user)
	return args.Get(0).(models.Record), args.Error(1)
}

func (m *urlShortenerMock) BatchShortify(records []models.Record, user models.User) ([]models.Record, error) {
	args := m.Called(records, user)
	return args.Get(0).([]models.Record), args.Error(1)
}

type urlCreaterBatchCreateResult struct {
	err         error
	returnValue []models.Record
}

type userAuthenticatorMock struct{ mock.Mock }

func (m *userAuthenticatorMock) AuthOrRegister(ctx context.Context, jwtStr string) (models.User, string, error) {
	args := m.Called(ctx, jwtStr)
	return args.Get(0).(models.User), args.String(1), args.Error(2)
}

func (m *userAuthenticatorMock) Auth(jwtStr string) (models.User, error) {
	args := m.Called(jwtStr)
	return args.Get(0).(models.User), args.Error(1)
}

type authOrRegisterResult struct {
	user   models.User
	jwtStr string
	err    error
}

type authResult struct {
	user models.User
	err  error
}

func generateAuthCookie(t require.TestingT, user models.User) *http.Cookie {
	jwtStr, err := auth.BuildJWTString(user)
	require.NoError(t, err)

	return &http.Cookie{
		Name:     "jwt",
		Value:    jwtStr,
		HttpOnly: true,
	}
}
