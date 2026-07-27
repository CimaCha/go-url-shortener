package post_shorten_url

import (
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/config/flag"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"io"
	"net/http"
)

type ShortenURLHandler struct {
	service             service.Service
	defaultShortAddress flag.BasicShortAddress
}

func NewShortenURLHandler(service service.Service, defaultShortAddress flag.BasicShortAddress) ShortenURLHandler {
	return ShortenURLHandler{service: service, defaultShortAddress: defaultShortAddress}
}

func (h ShortenURLHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

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

	finalURL := fmt.Sprintf("%s/%s", h.defaultShortAddress.String(), url)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write([]byte(finalURL))
}
