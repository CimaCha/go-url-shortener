package apishortenurl

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/post-api-shorten-url/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errHandlerService = errors.New("service error")

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestAPIShortenURLHandler(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		readError       bool
		setup           func(*mocks.MockURLService)
		wantStatus      int
		wantBody        string
		wantJSON        bool
		wantContentType string
	}{
		{
			name: "successful request",
			body: `{"url":"https://example.com/path"}`,
			setup: func(urlService *mocks.MockURLService) {
				urlService.EXPECT().
					SetShortURL("https://example.com/path").
					Return("short", nil)
			},
			wantStatus:      http.StatusCreated,
			wantBody:        `{"result":"http://localhost:8080/short"}`,
			wantJSON:        true,
			wantContentType: "application/json",
		},
		{
			name: "empty URL",
			body: `{}`,
			setup: func(urlService *mocks.MockURLService) {
				urlService.EXPECT().SetShortURL("").Return("", service.ErrEmptyURL)
			},
			wantStatus:      http.StatusBadRequest,
			wantBody:        "empty URL\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name: "service error",
			body: `{"url":"https://example.com/path"}`,
			setup: func(urlService *mocks.MockURLService) {
				urlService.EXPECT().SetShortURL("https://example.com/path").Return("", errHandlerService)
			},
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "Internal Server Error\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name:            "body read error",
			readError:       true,
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "Internal Server Error\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name:            "invalid JSON",
			body:            `{`,
			wantStatus:      http.StatusBadRequest,
			wantBody:        "Bad Request\n",
			wantContentType: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			urlService := mocks.NewMockURLService(controller)
			if tt.setup != nil {
				tt.setup(urlService)
			}
			handler := NewApiShortenURLHandler(urlService, "http://localhost:8080")
			var body io.Reader = strings.NewReader(tt.body)
			if tt.readError {
				body = errorReader{}
			}
			request := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			if tt.wantJSON {
				assert.JSONEq(t, tt.wantBody, response.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, response.Body.String())
			}
			assert.Equal(t, tt.wantContentType, response.Header().Get("Content-Type"))
		})
	}
}
