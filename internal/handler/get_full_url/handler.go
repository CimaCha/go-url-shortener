package get_full_url

import (
	"context"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"net/http"
	"net/url"
)

type GetFullHandler struct {
	ctx     context.Context
	service service.Service
}

func NewGetFullUrlHandler(ctx context.Context, service service.Service) GetFullHandler {
	return GetFullHandler{ctx: ctx, service: service}
}

func (h GetFullHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(res, "Only GET requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	id := req.PathValue("id")

	fullUrl, err := h.service.GetFullUrl(h.ctx, id)
	if err != nil {
		fmt.Print(err)
		return
	}

	fullUrl, err = url.QueryUnescape(fullUrl)
	if err != nil {
		fmt.Print(err)
		return
	}
	res.Header().Add("Location", fullUrl)

	res.WriteHeader(307)
}
