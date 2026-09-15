package userurls

import (
	"errors"
	"github.com/CimaCha/go-url-shortener/internal/authentication"
	authmocks "github.com/CimaCha/go-url-shortener/internal/authentication/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CimaCha/go-url-shortener/internal/handler/get-api-user-urls/mocks"
	"github.com/CimaCha/go-url-shortener/internal/model"
	"github.com/CimaCha/go-url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name            string
		result          []*model.UserRecord
		serviceErr      error
		wantStatus      int
		wantBody        string
		wantContentType string
	}{
		{
			name: "returns user URLs",
			result: []*model.UserRecord{
				{ShortURL: "short", OriginalURL: "https://example.com"},
			},
			wantStatus:      http.StatusOK,
			wantBody:        `[{"short_url":"http://localhost:8080/short","original_url":"https://example.com"}]`,
			wantContentType: "application/json",
		},
		{name: "user has no URLs", serviceErr: service.ErrUserNotFound, wantStatus: http.StatusNoContent},
		{name: "empty URL list", result: []*model.UserRecord{}, wantStatus: http.StatusNoContent},
		{name: "service error", serviceErr: errors.New("service error"), wantStatus: http.StatusInternalServerError, wantBody: "Internal Server Error\n", wantContentType: "text/plain; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			urlService := mocks.NewMockUserURLsGetter(controller)
			request := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
			request.Header.Set("userID", "forged-user")
			request.AddCookie(&http.Cookie{Name: "jwt", Value: "test-token"})
			parser := authmocks.NewMockUserIDParser(controller)
			parser.EXPECT().GetUserID("test-token").Return("user-id", nil)
			builder := authmocks.NewMockTokenBuilder(controller)
			authentication.AuthMiddleware(zap.NewNop(), builder, parser)(http.HandlerFunc(func(_ http.ResponseWriter, authenticated *http.Request) {
				request = authenticated
			})).ServeHTTP(httptest.NewRecorder(), request)
			urlService.EXPECT().GetUserURLs(request.Context(), "user-id").Return(tt.result, tt.serviceErr)
			handler := NewHandler(*zap.NewNop(), urlService, "http://localhost:8080")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, tt.wantStatus, response.Code)
			if tt.wantContentType == "application/json" {
				assert.JSONEq(t, tt.wantBody, response.Body.String())
			} else {
				assert.Equal(t, tt.wantBody, response.Body.String())
			}
			assert.Equal(t, tt.wantContentType, response.Header().Get("Content-Type"))
			if len(tt.result) > 0 {
				assert.Equal(t, "short", tt.result[0].ShortURL)
			}
		})
	}
}
