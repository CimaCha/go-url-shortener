package apishortenurl

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"io"
	"net/http"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_handler.gen.go -package=mocks

type URLService interface {
	SetShortURL(fullURL string) (string, error)
}

type Handler struct {
	service             URLService
	defaultShortAddress string
}

func NewAPIShortenURLHandler(service URLService, defaultShortAddress string) Handler {
	return Handler{service: service, defaultShortAddress: defaultShortAddress}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var decodedBody model.ShortenURLRequest
	var encodedBody model.ShortenURLResponse
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &decodedBody)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	url, err := h.service.SetShortURL(decodedBody.URL)
	if err != nil {
		if errors.Is(err, service.ErrEmptyURL) {
			http.Error(res, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	finalURL := fmt.Sprintf("%s/%s", h.defaultShortAddress, url)
	encodedBody.Result = finalURL
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	responseBody, err := json.Marshal(encodedBody)
	if err != nil {
		return
	}
	_, _ = res.Write(responseBody)
}
