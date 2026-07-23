package get_full_url

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/repository"
	"github.com/CimaCha/go-url-shortener/internal/repository/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errHandlerStorage = errors.New("storage error")

func TestGetFullUrlHandler(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		shortURL     string
		setup        func(*mocks.MockURLStorage)
		wantStatus   int
		wantLocation string
	}{
		{name: "method not allowed", method: http.MethodPost, shortURL: "short", wantStatus: http.StatusMethodNotAllowed},
		{
			name:     "stored URL",
			method:   http.MethodGet,
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("short").Return("https://example.com/path", nil)
			},
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: "https://example.com/path",
		},
		{
			name:     "missing URL",
			method:   http.MethodGet,
			shortURL: "missing",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("missing").Return("", repository.ErrURLNotFound)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "invalid escaped URL",
			method:   http.MethodGet,
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("short").Return("%", nil)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "storage error",
			method:   http.MethodGet,
			shortURL: "short",
			setup: func(storage *mocks.MockURLStorage) {
				storage.EXPECT().GetFullURL("short").Return("", errHandlerStorage)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			storage := mocks.NewMockURLStorage(controller)
			if tt.setup != nil {
				tt.setup(storage)
			}
			handler := NewGetFullURLHandler(service.NewService(storage))
			router := chi.NewRouter()
			router.Method(http.MethodGet, "/{id}", handler)
			request := httptest.NewRequest(tt.method, "/"+tt.shortURL, nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantLocation, response.Header().Get("Location"))
		})
	}
}
