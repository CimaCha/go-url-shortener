package apideletebatch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_deleter.gen.go -package=mocks

type Deleter interface {
	DeleteBatch(ctx context.Context, shortURLBatch []string, userID string) error
}

type Handler struct {
	log     zap.Logger
	service Deleter
}

func NewAPIDeleteBatchHandler(log zap.Logger, service Deleter) Handler {
	return Handler{
		log:     log,
		service: service,
	}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var decodedBody []string
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
	if len(decodedBody) == 0 {
		http.Error(res, "empty body", http.StatusBadRequest)
		return
	}
	userID := req.Header.Get("UserID")
	if err := h.service.DeleteBatch(req.Context(), decodedBody, userID); err != nil {
		h.log.Error("can't delete URL", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusAccepted)
}
