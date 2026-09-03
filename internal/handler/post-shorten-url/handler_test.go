package shortenurl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

var errHandlerService = errors.New("service error")

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("read error") }

func TestShortenURLHandler(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		readError       bool
		setup           func(*mocks.MockShortener, context.Context)
		wantStatus      int
		wantBody        string
		wantContentType string
	}{
		{
			name: "successful request",
			body: "https://example.com/path",
			setup: func(urlService *mocks.MockShortener, ctx context.Context) {
				urlService.EXPECT().
					Shorten(ctx, "https://example.com/path", "user-id").
					Return("short", nil)
			},
			wantStatus:      http.StatusCreated,
			wantBody:        "http://localhost:8080/short",
			wantContentType: "text/plain",
		},
		{
			name: "empty URL",
			setup: func(urlService *mocks.MockShortener, ctx context.Context) {
				urlService.EXPECT().Shorten(ctx, "", "user-id").Return("", service.ErrEmptyURL)
			},
			wantStatus:      http.StatusBadRequest,
			wantBody:        "empty URL\n",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name: "service error",
			body: "https://example.com/path",
			setup: func(urlService *mocks.MockShortener, ctx context.Context) {
				urlService.EXPECT().Shorten(ctx, "https://example.com/path", "user-id").Return("", errHandlerService)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			urlService := mocks.NewMockShortener(controller)
			var body io.Reader = strings.NewReader(tt.body)
			if tt.readError {
				body = errorReader{}
			}
			request := httptest.NewRequest(http.MethodPost, "/", body)
			request.Header.Set("userID", "user-id")
			if tt.setup != nil {
				tt.setup(urlService, request.Context())
			}
			handler := NewShortenURLHandler(*zap.NewNop(), urlService, "http://localhost:8080")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantBody, response.Body.String())
			assert.Equal(t, tt.wantContentType, response.Header().Get("Content-Type"))
		})
	}
}
