package apishortenurl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
	"io"
	"net/http"
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

func NewAPIShortenURLHandler(log zap.Logger, service Shortener, defaultShortAddress string) Handler {
	return Handler{
		log:                 log,
		service:             service,
		defaultShortAddress: defaultShortAddress,
	}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var decodedBody model.ShortenURLRequest
	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.log.Error("can't read request body", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &decodedBody)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := h.service.Shorten(req.Context(), decodedBody.URL)
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
	var encodedBody model.ShortenURLResponse
	encodedBody.Result = fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	responseBody, err := json.Marshal(encodedBody)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return res
	}
	_, _ = res.Write(responseBody)
	return res
}
