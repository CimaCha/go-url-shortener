package router

import (
	"github.com/CimaCha/go-url-shortener/internal/logger"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(shortenURLHandler, getFullURLHandler http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(logger.RequestLogger)
	router.With(middleware.AllowContentType("text/plain")).
		Method(http.MethodPost, "/", shortenURLHandler)
	router.Method(http.MethodGet, "/{id}", getFullURLHandler)
	return router
}
