package handlers_test

import (
	"encoding/json"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"

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
