package shortenurl

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/service"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_handler.gen.go -package=mocks

type Shortener interface {
	Shorten(ctx context.Context, fullURL string) (string, error)
}

type Handler struct {
	log                 zap.Logger
	service             Shortener
	defaultShortAddress string
}

func NewShortenURLHandler(log zap.Logger, service Shortener, defaultShortAddress string) Handler {
	return Handler{
		log:                 log,
		service:             service,
		defaultShortAddress: defaultShortAddress,
	}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.log.Error("can't read body", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	url, err := h.service.Shorten(req.Context(), string(body))
	if err != nil {
		if errors.Is(err, service.ErrFullURLExists) {
			finalURL := fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
			res.WriteHeader(http.StatusConflict)
			res.Header().Set("Content-Type", "application/json")
			_, err = res.Write([]byte(finalURL))
			if err != nil {
				http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			return
		}
		if errors.Is(err, service.ErrEmptyURL) {
			http.Error(res, err.Error(), http.StatusBadRequest)
		} else {
			h.log.Error("can't shorten URL", zap.Error(err))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	finalURL := fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write([]byte(finalURL))
}
