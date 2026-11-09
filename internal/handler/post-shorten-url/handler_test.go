package shortenurl

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/audit"
	"github.com/CimaCha/go-url-shortener/internal/authentication"
	authmocks "github.com/CimaCha/go-url-shortener/internal/authentication/mocks"

	"github.com/CimaCha/go-url-shortener/internal/handler/post-shorten-url/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"gopkg.in/h2non/gentleman.v2"
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
			request.Header.Set("userID", "forged-user")
			request.AddCookie(&http.Cookie{Name: "jwt", Value: "test-token"})
			parser := authmocks.NewMockUserIDParser(controller)
			parser.EXPECT().GetUserID("test-token").Return("user-id", nil)
			builder := authmocks.NewMockTokenBuilder(controller)
			authentication.AuthMiddleware(zap.NewNop(), builder, parser)(http.HandlerFunc(func(_ http.ResponseWriter, authenticated *http.Request) {
				request = authenticated
			})).ServeHTTP(httptest.NewRecorder(), request)
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

func TestShortenURLHandlerReportsOriginalURL(t *testing.T) {
	const originalURL = "https://example.com/path"
	events := make(chan audit.Exposure, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event audit.Exposure
		if err := json.NewDecoder(r.Body).Decode(&event); err == nil {
			events <- event
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	producer := audit.Producer{}
	producer.Register(audit.NewURLClientSubscriber("url", server.URL, gentleman.New()))
	urlService := mocks.NewMockShortener(gomock.NewController(t))
	urlService.EXPECT().Shorten(gomock.Any(), originalURL, "").Return("short", nil)
	handler := audit.AuditMiddleware(zap.NewNop(), producer, audit.Shorten)(NewShortenURLHandler(*zap.NewNop(), urlService, "http://localhost:8080"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(originalURL)))
	producer.Close()

	require.Equal(t, http.StatusCreated, response.Code)
	select {
	case event := <-events:
		require.Equal(t, originalURL, event.URL)
	default:
		t.Fatal("audit event was not sent")
	}
}
