package router

import (
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/compression"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func New(log *zap.Logger, shortenURLHandler, apiShortenURLHandler, getFullURLHandler http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(logger.RequestLogger(log))
	router.Use(compression.GzipMiddleware(log))
	router.With(middleware.AllowContentType("text/plain")).
		Method(http.MethodPost, "/", shortenURLHandler)
	router.With(middleware.AllowContentType("application/json")).
		Method(http.MethodPost, "/api/shorten", apiShortenURLHandler)
	router.Method(http.MethodGet, "/{id}", getFullURLHandler)
	return router
}
