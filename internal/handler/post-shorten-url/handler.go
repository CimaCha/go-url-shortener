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
	service        service.Service
	defaultAddress flag.NetAddress
}

func NewShortenURLHandler(service service.Service, defaultAddress flag.NetAddress) ShortenURLHandler {
	return ShortenURLHandler{service: service, defaultAddress: defaultAddress}
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

	finalURL := fmt.Sprintf("http://%s/%s", h.defaultAddress.String(), url)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write([]byte(finalURL))
}
