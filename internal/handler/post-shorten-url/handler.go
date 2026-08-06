package shortenurl

import (
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"io"
	"net/http"
)

type Handler struct {
	service             service.Service
	defaultShortAddress string
}

func NewShortenURLHandler(service service.Service, defaultShortAddress string) Handler {
	return Handler{service: service, defaultShortAddress: defaultShortAddress}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	url, err := h.service.SetShortURL(string(body))
	if err != nil {
		if errors.Is(err, service.ErrEmptyURL) {
			http.Error(res, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	finalURL := fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write([]byte(finalURL))
}
