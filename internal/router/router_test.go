package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		wantStatus  int
		wantHandler string
		wantID      string
	}{
		{name: "shorten URL", method: http.MethodPost, path: "/", contentType: "text/plain", wantStatus: http.StatusCreated, wantHandler: "shorten"},
		{name: "unsupported content type", method: http.MethodPost, path: "/", contentType: "application/json", wantStatus: http.StatusUnsupportedMediaType},
		{name: "get full URL", method: http.MethodGet, path: "/short", wantStatus: http.StatusTemporaryRedirect, wantHandler: "full", wantID: "short"},
		{name: "method not allowed", method: http.MethodPut, path: "/", wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotHandler string
			var gotID string
			shortenURLHandler := http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
				gotHandler = "shorten"
				res.WriteHeader(http.StatusCreated)
			})
			getFullURLHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				gotHandler = "full"
				gotID = chi.URLParam(req, "id")
				res.WriteHeader(http.StatusTemporaryRedirect)
			})
			router := New(shortenURLHandler, getFullURLHandler)
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader("https://example.com"))
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantHandler, gotHandler)
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}
