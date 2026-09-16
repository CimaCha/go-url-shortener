package apideletebatch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/authentication"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"go.uber.org/zap"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_deleter.gen.go -package=mocks

type Deleter interface {
	ProcessDeleteTasks(onError func(error), batchSize int, timeout time.Duration, inputs ...<-chan service.DeleteTask)
}

type Handler struct {
	log  zap.Logger
	jobs chan service.DeleteTask
	done chan struct{}
}

func NewAPIDeleteBatchHandler(log zap.Logger, deleter Deleter, batchSize int, timeout time.Duration) Handler {
	h := Handler{log: log, jobs: make(chan service.DeleteTask, 100), done: make(chan struct{})}
	go func() {
		defer close(h.done)
		deleter.ProcessDeleteTasks(func(err error) {
			h.log.Error("can't delete URL", zap.Error(err))
		}, batchSize, timeout, h.jobs)
	}()
	return h
}

func (h Handler) Close() {
	close(h.jobs)
	<-h.done
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
	task := service.DeleteTask{
		Context: context.WithoutCancel(req.Context()),
		URLs:    decodedBody,
		UserID:  authentication.UserID(req.Context()),
	}
	select {
	case h.jobs <- task:
		res.WriteHeader(http.StatusAccepted)
	default:
		http.Error(res, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	}
}
