package router

import (
	"github.com/CimaCha/go-url-shortener/internal/encryption"
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(shortenURLHandler, apiShortenURLHandler, getFullURLHandler http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(logger.RequestLogger)
	router.Use(encryption.GzipMiddleware)
	router.With(middleware.AllowContentType("text/plain")).
		Method(http.MethodPost, "/", shortenURLHandler)
	router.With(middleware.AllowContentType("application/json")).
		Method(http.MethodPost, "/api/shorten", apiShortenURLHandler)
	router.Method(http.MethodGet, "/{id}", getFullURLHandler)
	return router
}
