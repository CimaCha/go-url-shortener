package post_shorten_url

import (
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"io"
	"net/http"
)

type ShortenURLHandler struct {
	service service.Service
}

func NewShortenURLHandler(service service.Service) ShortenURLHandler {
	return ShortenURLHandler{service: service}
}

func (h ShortenURLHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(res, "Content-Type must be text/plain", http.StatusUnsupportedMediaType)
		return
	}

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

	finalURL := fmt.Sprintf("http://%s/%s\n", req.Host, url)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write([]byte(finalURL))
}
