package get_full_url

import (
	"github.com/CimaCha/go-url-shortener/internal/service"
	"net/http"
	"net/url"
)

type GetFullHandler struct {
	service service.Service
}

func NewGetFullURLHandler(service service.Service) GetFullHandler {
	return GetFullHandler{service: service}
}

func (h GetFullHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(res, "Only GET requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	id := req.PathValue("id")

	fullURL, err := h.service.GetFullURL(id)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	fullURL, err = url.QueryUnescape(fullURL)
	if err != nil {
		http.Error(res, "Escaping URL error", http.StatusInternalServerError)
		return
	}

	res.Header().Add("Location", fullURL)

	res.WriteHeader(307)
}
