package apishortenbatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_handler.gen.go -package=mocks

// Shortener creates short IDs for a batch of original URLs.
type Shortener interface {
	ShortenBatch(ctx context.Context, fullURLBatch []*model.OriginalURLRecord, userID string) ([]*model.ShortURLRecord, error)
}

// Handler serves the JSON batch shortening endpoint.
type Handler struct {
	log                 zap.Logger
	service             Shortener
	defaultShortAddress string
}

const maxBatchBodyBytes = 10 << 20

// NewAPIShortenBatchHandler creates a handler for POST /api/shorten/batch.
func NewAPIShortenBatchHandler(log zap.Logger, service Shortener, defaultShortAddress string) Handler {
	return Handler{
		log:                 log,
		service:             service,
		defaultShortAddress: defaultShortAddress,
	}
}

// ServeHTTP returns short URLs with their correlation IDs and status 201.
func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var decodedBody model.ShortenBatchRequest
	decoder := json.NewDecoder(http.MaxBytesReader(res, req.Body, maxBatchBodyBytes))
	if err := decoder.Decode(&decodedBody); err != nil {
		h.writeDecodeError(res, err)
		return
	}
	if err := decoder.Decode(&json.RawMessage{}); err == nil {
		http.Error(res, "multiple JSON values", http.StatusBadRequest)
		return
	} else if !errors.Is(err, io.EOF) {
		h.writeDecodeError(res, err)
		return
	}
	shortURLs, err := h.service.ShortenBatch(req.Context(), decodedBody, authentication.UserID(req.Context()))
	if err != nil {
		if errors.Is(err, service.ErrEmptyURLList) || errors.Is(err, service.ErrEmptyURL) {
			http.Error(res, err.Error(), http.StatusBadRequest)
		} else {
			h.log.Error("can't shorten URL", zap.Error(err))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	finalURL := convertShortURLToFinalURL(h.defaultShortAddress, shortURLs)
	encodedBody := model.ShortenBatchResponse(finalURL)
	responseBody, err := json.Marshal(encodedBody)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	_, _ = res.Write(responseBody)
}

func (h Handler) writeDecodeError(res http.ResponseWriter, err error) {
	var limitErr *http.MaxBytesError
	if errors.As(err, &limitErr) {
		http.Error(res, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		http.Error(res, "unexpected end of JSON input", http.StatusBadRequest)
		return
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	h.log.Error("can't read request body", zap.Error(err))
	http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func convertShortURLToFinalURL(shortAddress string, shortURLs []*model.ShortURLRecord) []*model.ShortURLRecord {
	for _, shortURL := range shortURLs {
		shortURL.ShortURL = fmt.Sprintf("%s/%s", shortAddress, shortURL.ShortURL)
	}
	return shortURLs
}
