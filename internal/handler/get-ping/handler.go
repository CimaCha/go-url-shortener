package ping

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_pinger.gen.go -package=mocks

// Pinger checks whether a dependency is reachable.
type Pinger interface {
	Ping(context.Context) error
}

// Handler serves the database health endpoint.
type Handler struct {
	log    zap.Logger
	dbPool Pinger
}

// NewDBConnectionPingHandler creates a handler for GET /ping.
func NewDBConnectionPingHandler(log zap.Logger, dbPool Pinger) Handler {
	return Handler{
		log:    log,
		dbPool: dbPool,
	}
}

// ServeHTTP reports status 200 when Ping succeeds and 500 otherwise.
func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	err := h.dbPool.Ping(req.Context())
	if err != nil {
		h.log.Error("can't connect to database", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// PingFunc adapts a function to the Pinger interface.
type PingFunc func(context.Context) error

// Ping calls the wrapped function.
func (f PingFunc) Ping(ctx context.Context) error {
	return f(ctx)
}
