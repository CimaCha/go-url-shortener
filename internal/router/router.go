package router

import (
	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/compression"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func New(log *zap.Logger, jwtBuilder authentication.JWTBuilder, shortenURLHandler, apiShortenURLHandler, getFullURLHandler, pingHandler, apiShortenBatchHandler, userURLsHandler http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(logger.RequestLogger(log))
	router.Use(compression.GzipMiddleware(log))
	router.Use(authentication.AuthMiddleware(log, jwtBuilder))
	router.With(middleware.AllowContentType("text/plain")).
		Method(http.MethodPost, "/", shortenURLHandler)
	router.With(middleware.AllowContentType("application/json")).
		Method(http.MethodPost, "/api/shorten", apiShortenURLHandler)
	router.With(middleware.AllowContentType("application/json")).
		Method(http.MethodPost, "/api/shorten/batch", apiShortenBatchHandler)
	router.Method(http.MethodGet, "/{id}", getFullURLHandler)
	router.Method(http.MethodGet, "/ping", pingHandler)
	router.Method(http.MethodGet, "/api/user/urls", userURLsHandler)
	return router
}
