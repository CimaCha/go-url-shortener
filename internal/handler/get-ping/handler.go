package ping

import (
	"context"
	"go.uber.org/zap"
	"net/http"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_pinger.gen.go -package=mocks

type Pinger interface {
	Ping(context.Context) error
}

type Handler struct {
	log    zap.Logger
	dbPool Pinger
}

func NewDBConnectionPingHandler(log zap.Logger, dbPool Pinger) Handler {
	return Handler{
		log:    log,
		dbPool: dbPool,
	}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	err := h.dbPool.Ping(req.Context())
	if err != nil {
		h.log.Error("can't connect to database", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

type PingFunc func(context.Context) error

func (f PingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}
