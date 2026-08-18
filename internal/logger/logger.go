package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Initialize инициализирует логера с необходимым уровнем логирования.
func Initialize(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return zl, nil
}

// RequestLogger — middleware-логер для входящих HTTP-запросов.
func RequestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			defer func() {
				log.Info("handled HTTP request",
					zap.String("method", request.Method),
					zap.String("uri", request.RequestURI),
					zap.Duration("duration", time.Since(started)),
				)
			}()

			handler.ServeHTTP(writer, request)
		})
	}
}
