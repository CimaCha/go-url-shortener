package fullurl

import (
	"errors"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_service.gen.go -package=mocks

type URLService interface {
	GetFullURL(shortURL string) (string, error)
}

type Handler struct {
	service URLService
}

func NewGetFullURLHandler(service URLService) Handler {
	return Handler{service: service}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	id := chi.URLParam(req, "id")

	fullURL, err := h.service.GetFullURL(id)
	if err != nil {
		if errors.Is(err, service.ErrURLNotFound) {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrEmptyURL) {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Header().Add("Location", fullURL)
	res.WriteHeader(http.StatusTemporaryRedirect)
}
