package logger

import (
	"net/http"

	"go.uber.org/zap"
)

// Logger
var Log *zap.Logger = zap.NewNop()

// Logging response writer
type LoggingResponseWriter struct {
	http.ResponseWriter
	ResponseStatus int
	ResponseSize   int
}

// Write
func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.ResponseSize += size

	return size, err
}

// WriteHeader
func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseStatus = statusCode
}

// Initialize Log
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	config := zap.NewProductionConfig()
	config.Level = lvl
	zLogger, err := config.Build()
	if err != nil {
		return err
	}

	Log = zLogger
	return nil
}
