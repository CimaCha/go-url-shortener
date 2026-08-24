package get_ping

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/get-ping/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type contextKey struct{}

func TestDBConnectionPingHandler(t *testing.T) {
	tests := []struct {
		name       string
		pingError  error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "database is available",
			wantStatus: http.StatusOK,
		},
		{
			name:       "database is unavailable",
			pingError:  errors.New("ping database"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			dbPool := mocks.NewMockGetPingService(controller)
			request := httptest.NewRequest(http.MethodGet, "/ping", nil)
			request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "request"))
			dbPool.EXPECT().Ping(request.Context()).Return(tt.pingError)
			handler := NewDBConnectionPingHandler(dbPool)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			assert.Equal(t, tt.wantBody, response.Body.String())
		})
	}
}
