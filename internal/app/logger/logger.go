package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

type LoggingResponseWriter struct {
	http.ResponseWriter
	responseStatus int
	responseSize   int
}

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseSize += size

	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseStatus = statusCode
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

func ResponseLogger(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := LoggingResponseWriter{
			ResponseWriter: w,
			responseStatus: 0,
			responseSize:   0,
		}
		h(&lw, r)
		Log.Info(
			"response",
			zap.Int("status", lw.responseStatus),
			zap.Int("size", lw.responseSize),
		)
	})
}

func RequestLogger(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h(w, r)
		duration := time.Since(start)
		Log.Info("got incoming HTTP request",
			zap.String("method", r.Method),
			zap.String("URI", r.RequestURI),
			zap.String("duration", duration.String()),
		)
	})
}
