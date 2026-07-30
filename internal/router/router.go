package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(shortenURLHandler, getFullURLHandler http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.With(middleware.AllowContentType("text/plain")).
		Method(http.MethodPost, "/", shortenURLHandler)
	router.Method(http.MethodGet, "/{id}", getFullURLHandler)
	return router
}
