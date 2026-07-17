package post_shorten_url

import (
	"context"
	"fmt"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"io"
	"net/http"
)

type ShortenUrlHandler struct {
	ctx     context.Context
	service service.Service
}

func NewShortenUrlHandler(ctx context.Context, service service.Service) ShortenUrlHandler {
	return ShortenUrlHandler{ctx: ctx, service: service}
}

func (h ShortenUrlHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		http.Error(res, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "text/plain" {
		http.Error(res, "Content-Type must be text/plain", http.StatusUnsupportedMediaType)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Print(err)
		return
	}
	url, err := h.service.SetShortUrl(h.ctx, string(body))
	if err != nil {
		fmt.Print(err)
		return
	}

	finalUrl := req.Host + "/" + url
	res.WriteHeader(201)
	res.Header().Set("Content-Type", "text/plain")
	_, err = res.Write([]byte(finalUrl))
	if err != nil {
		fmt.Print(err)
		return
	}
}
