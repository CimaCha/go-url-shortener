package post_shorten_url

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errHandlerStorage = errors.New("storage error")

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestShortenUrlHandler(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		readError   bool
		setup       func(*mocks.MockUrlStorage)
		wantStatus  int
		wantBody    string
	}{
		{name: "method not allowed", method: http.MethodGet, wantStatus: http.StatusMethodNotAllowed, wantBody: "Only POST requests are allowed!\n"},
		{
			name:        "successful request",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com/path",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().SetShortUrl("q8T575iSknB5NIL7Yf_g5s9Bnjk", "https://example.com/path").Return(nil)
			},
			wantStatus: http.StatusCreated,
			wantBody:   "http://short.test/q8T575iSknB5NIL7Yf_g5s9Bnjk\n",
		},
		{name: "unsupported content type", method: http.MethodPost, contentType: "application/json", body: "https://example.com/path", wantStatus: http.StatusUnsupportedMediaType, wantBody: "Content-Type must be text/plain\n"},
		{name: "empty URL", method: http.MethodPost, contentType: "text/plain", wantStatus: http.StatusBadRequest, wantBody: "empty URL\n"},
		{name: "body read error", method: http.MethodPost, contentType: "text/plain", readError: true, wantStatus: http.StatusInternalServerError, wantBody: "Internal Server Error\n"},
		{
			name:        "storage error",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "https://example.com",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().SetShortUrl("Mnw_2ofOKGhIpXSYLd0LfHSH-BY", "https://example.com").Return(errHandlerStorage)
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockUrlStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}
			handler := NewShortenURLHandler(service.NewService(storage))
			var body io.Reader = strings.NewReader(tt.body)
			if tt.readError {
				body = errorReader{}
			}
			request := httptest.NewRequest(tt.method, "/", body)
			request.Host = "short.test"
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantBody, response.Body.String())
		})
	}
}
