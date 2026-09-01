package fullurl

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_handler.gen.go -package=mocks

type Resolver interface {
	Resolve(ctx context.Context, shortURL string) (string, error)
}

type Handler struct {
	log     zap.Logger
	service Resolver
}

func NewGetFullURLHandler(log zap.Logger, service Resolver) Handler {
	return Handler{
		log:     log,
		service: service}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	id := chi.URLParam(req, "id")

	fullURL, err := h.service.Resolve(req.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrURLNotFound) {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrEmptyURL) {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		h.log.Error("can't resolve URL", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Header().Add("Location", fullURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}
