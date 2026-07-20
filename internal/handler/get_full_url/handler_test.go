package get_full_url

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/repository/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errHandlerStorage = errors.New("storage error")

func TestGetFullUrlHandler(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		shortURL     string
		setup        func(*mocks.MockUrlStorage)
		wantStatus   int
		wantLocation string
	}{
		{name: "method not allowed", method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed},
		{
			name:     "stored URL",
			method:   http.MethodGet,
			shortURL: "short",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("short").Return("https://example.com/path", nil)
			},
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: "https://example.com/path",
		},
		{
			name:     "missing URL",
			method:   http.MethodGet,
			shortURL: "missing",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("missing").Return("", repository.ErrURLNotFound)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "invalid escaped URL",
			method:   http.MethodGet,
			shortURL: "short",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("short").Return("%", nil)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "storage error",
			method:   http.MethodGet,
			shortURL: "short",
			setup: func(storage *mocks.MockUrlStorage) {
				storage.EXPECT().GetFullUrl("short").Return("", errHandlerStorage)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockUrlStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}
			handler := NewGetFullUrlHandler(service.NewService(storage))
			request := httptest.NewRequest(tt.method, "/"+tt.shortURL, nil)
			request.SetPathValue("id", tt.shortURL)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantLocation, response.Header().Get("Location"))
		})
	}
}
