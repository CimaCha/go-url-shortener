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
	Shorten(ctx context.Context, fullURL, userID string) (string, error)
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
	url, err := h.service.Shorten(req.Context(), string(body), req.Header.Get("userID"))
	if err != nil {
		if errors.Is(err, service.ErrFullURLExists) {
			res = h.createResponse(res, url, http.StatusConflict)
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

	res = h.createResponse(res, url, http.StatusCreated)
}

func (h Handler) createResponse(res http.ResponseWriter, url string, status int) http.ResponseWriter {
	finalURL := fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(status)
	_, err := res.Write([]byte(finalURL))
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return res
	}
	return res
}
