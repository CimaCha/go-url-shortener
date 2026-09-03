package userurls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
	"net/http"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_user_urls_getter.gen.go -package=mocks

type UserURLsGetter interface {
	GetUserURLs(ctx context.Context, userID string) ([]*model.UserRecord, error)
}

func NewHandler(log zap.Logger, service UserURLsGetter, defaultShortAddress string) Handler {
	return Handler{log: log, service: service, defaultShortAddress: defaultShortAddress}
}

type Handler struct {
	log                 zap.Logger
	service             UserURLsGetter
	defaultShortAddress string
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	userID := req.Header.Get("userID")

	userURLsList, err := h.service.GetUserURLs(req.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			res.WriteHeader(http.StatusNoContent)
			return
		}
		h.log.Error("can't resolve URL", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(userURLsList) == 0 {
		res.WriteHeader(http.StatusNoContent)
	} else {
		encodedBody := make(model.UserURLsResponse, len(userURLsList))
		for i, record := range userURLsList {
			encodedBody[i] = &model.UserRecord{
				ShortURL:    fmt.Sprintf("%s/%s", h.defaultShortAddress, record.ShortURL),
				OriginalURL: record.OriginalURL,
			}
		}
		responseBody, err := json.Marshal(encodedBody)
		if err != nil {
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		_, _ = res.Write(responseBody)
	}
}
