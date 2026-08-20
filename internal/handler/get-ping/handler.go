package get_ping

import (
	"context"
	"net/http"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_pinger.gen.go -package=mocks

type GetPingService interface {
	Ping(context.Context) error
}

type Handler struct {
	dbPool GetPingService
}

func NewDBConnectionPingHandler(dbPool GetPingService) Handler {
	return Handler{dbPool: dbPool}
}

func (h Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	err := h.dbPool.Ping(req.Context())
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}
