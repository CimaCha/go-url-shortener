package get_ping

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_url_handler.gen.go -package=mocks

type Service interface {
	Ping() error
}

type Handler struct {
	ctx    context.Context
	dbPool *pgxpool.Pool
}

func NewDBConnectionPingHandler(ctx context.Context, dbPool *pgxpool.Pool) Handler {
	return Handler{ctx: ctx, dbPool: dbPool}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, _ *http.Request) {
	err := h.dbPool.Ping(h.ctx)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}
