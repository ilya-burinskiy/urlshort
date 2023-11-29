package logger

import (
	"net/http"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

type LoggingResponseWriter struct {
	http.ResponseWriter
	ResponseStatus int
	ResponseSize   int
}

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.ResponseSize += size

	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseStatus = statusCode
}

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
