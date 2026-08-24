package fullurl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/get-full-url/mocks"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var errHandlerService = errors.New("service error")

func TestGetFullURLHandler(t *testing.T) {
	tests := []struct {
		name         string
		shortURL     string
		setup        func(*mocks.MockGetFullURLService)
		wantStatus   int
		wantBody     string
		wantLocation string
	}{
		{
			name:     "stored URL",
			shortURL: "short",
			setup: func(urlService *mocks.MockGetFullURLService) {
				urlService.EXPECT().Resolve("short").Return("https://example.com/path", nil)
			},
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: "https://example.com/path",
		},
		{
			name:     "missing URL",
			shortURL: "missing",
			setup: func(urlService *mocks.MockGetFullURLService) {
				urlService.EXPECT().Resolve("missing").Return("", service.ErrURLNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "URL not found\n",
		},
		{
			name: "empty URL",
			setup: func(urlService *mocks.MockGetFullURLService) {
				urlService.EXPECT().Resolve("").Return("", service.ErrEmptyURL)
			},
			wantStatus: http.StatusBadRequest,
			wantBody:   "empty URL\n",
		},
		{
			name:     "service error",
			shortURL: "short",
			setup: func(urlService *mocks.MockGetFullURLService) {
				urlService.EXPECT().Resolve("short").Return("", errHandlerService)
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			urlService := mocks.NewMockGetFullURLService(controller)
			tt.setup(urlService)
			handler := NewGetFullURLHandler(urlService)
			request := httptest.NewRequest(http.MethodGet, "/"+tt.shortURL, nil)
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("id", tt.shortURL)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantBody, response.Body.String())
			assert.Equal(t, tt.wantLocation, response.Header().Get("Location"))
		})
	}
}
