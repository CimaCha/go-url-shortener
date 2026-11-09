package apishortenurl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/audit"
	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_handler.gen.go -package=mocks

// Shortener creates a short ID for a user's original URL.
type Shortener interface {
	Shorten(ctx context.Context, fullURL, userID string) (string, error)
}

// Handler serves the JSON shortening endpoint.
type Handler struct {
	log                 zap.Logger
	service             Shortener
	defaultShortAddress string
}

// NewAPIShortenURLHandler creates a handler for POST /api/shorten.
func NewAPIShortenURLHandler(log zap.Logger, service Shortener, defaultShortAddress string) Handler {
	return Handler{
		log:                 log,
		service:             service,
		defaultShortAddress: defaultShortAddress,
	}
}

// ServeHTTP reads a JSON URL request and returns a JSON short URL with status 201.
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
	url, err := h.service.Shorten(req.Context(), decodedBody.URL, authentication.UserID(req.Context()))
	if err != nil {
		if errors.Is(err, service.ErrFullURLExists) {
			var encodedBody model.ShortenURLResponse
			encodedBody.Result = fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
			responseBody, err := json.Marshal(encodedBody)
			if err != nil {
				http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusConflict)
			_, _ = res.Write(responseBody)
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

	audit.RecordURL(req.Context(), decodedBody.URL)
	var encodedBody model.ShortenURLResponse
	encodedBody.Result = fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
	responseBody, err := json.Marshal(encodedBody)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write(responseBody)
}
