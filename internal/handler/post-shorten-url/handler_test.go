package post_shorten_url

import (
	"errors"
	"github.com/CimaCha/go-url-shortener/internal/config/flag"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestShortenUrlHandler(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		readError   bool
		setup       func(*mocks.MockURLStorage)
		wantStatus  int
		wantBody    string
	}{
		{name: "method not allowed", method: http.MethodGet, wantStatus: http.StatusMethodNotAllowed},
		{
			name:        "successful request",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com/path",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().SetShortURL("q8T575iSknB5NIL7Yf_g5s9Bnjk", "https://example.com/path")
			},
			wantStatus: http.StatusCreated,
			wantBody:   "http://localhost:8080/q8T575iSknB5NIL7Yf_g5s9Bnjk",
		},
		{name: "unsupported content type", method: http.MethodPost, contentType: "application/json", body: "https://example.com/path", wantStatus: http.StatusUnsupportedMediaType},
		{name: "empty URL", method: http.MethodPost, contentType: "text/plain", wantStatus: http.StatusBadRequest, wantBody: "empty URL\n"},
		{name: "body read error", method: http.MethodPost, contentType: "text/plain", readError: true, wantStatus: http.StatusInternalServerError, wantBody: "Internal Server Error\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockURLStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}
			shortAddress := flag.NewBasicShortAddress("http://localhost:8080")
			handler := NewShortenURLHandler(service.NewService(storage), *shortAddress)
			router := chi.NewRouter()
			router.With(middleware.AllowContentType("text/plain")).
				Method(http.MethodPost, "/", handler)
			var body io.Reader = strings.NewReader(tt.body)
			if tt.readError {
				body = errorReader{}
			}
			request := httptest.NewRequest(tt.method, "/", body)
			request.Host = "short.test"
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantBody, response.Body.String())
		})
	}
}
