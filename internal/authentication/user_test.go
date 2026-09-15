package authentication

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CimaCha/go-url-shortener/internal/authentication/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestAuthMiddlewareCreatesUserCookie(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantSecure bool
	}{
		{name: "HTTP cookie", target: "http://example.com/", wantSecure: false},
		{name: "HTTPS cookie", target: "https://example.com/", wantSecure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			builder := mocks.NewMockTokenBuilder(controller)
			parser := mocks.NewMockUserIDParser(controller)
			var gotUserID string
			var builtUserID string
			builder.EXPECT().BuildJWTString(gomock.Any(), TokenExp).DoAndReturn(func(id string, exp time.Duration) (string, error) {
				require.Len(t, id, 32)
				builtUserID = id
				return "new-token", nil
			})
			next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				gotUserID = getUserID(request.Context())
				writer.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			request.Header.Set("userID", "forged-user")
			response := httptest.NewRecorder()

			AuthMiddleware(zap.NewNop(), builder, parser)(next).ServeHTTP(response, request)

			require.Equal(t, http.StatusNoContent, response.Code)
			require.NotEmpty(t, gotUserID)
			cookies := response.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, "jwt", cookies[0].Name)
			require.Equal(t, tt.wantSecure, cookies[0].Secure)
			require.WithinDuration(t, time.Now().Add(TokenExp), cookies[0].Expires, 2*time.Second)
			require.Equal(t, "new-token", cookies[0].Value)
			require.Equal(t, builtUserID, gotUserID)
			require.Equal(t, "/", cookies[0].Path)
			require.True(t, cookies[0].HttpOnly)
			require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		})
	}
}

func TestAuthMiddlewareUsesValidCookie(t *testing.T) {
	controller := gomock.NewController(t)
	builder := mocks.NewMockTokenBuilder(controller)
	parser := mocks.NewMockUserIDParser(controller)
	token := "existing-token"
	parser.EXPECT().GetUserID(token).Return("existing-user", nil)
	request := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	request.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	request.Header.Set("userID", "forged-user")
	response := httptest.NewRecorder()
	called := false

	AuthMiddleware(zap.NewNop(), builder, parser)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		called = true
		require.Equal(t, "existing-user", getUserID(request.Context()))
	})).ServeHTTP(response, request)

	require.True(t, called)
}

func TestAuthMiddlewareRejectsInvalidCookie(t *testing.T) {
	controller := gomock.NewController(t)
	builder := mocks.NewMockTokenBuilder(controller)
	parser := mocks.NewMockUserIDParser(controller)
	request := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	request.AddCookie(&http.Cookie{Name: "jwt", Value: "invalid"})
	parser.EXPECT().GetUserID("invalid").Return("", ErrTokenIsInvalid)
	response := httptest.NewRecorder()
	called := false

	AuthMiddleware(zap.NewNop(), builder, parser)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})).ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.False(t, called)
}

func TestAuthMiddlewareHandlesBuilderError(t *testing.T) {
	controller := gomock.NewController(t)
	builder := mocks.NewMockTokenBuilder(controller)
	parser := mocks.NewMockUserIDParser(controller)
	builder.EXPECT().BuildJWTString(gomock.Any(), TokenExp).Return("", errors.New("signing failed"))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	called := false
	AuthMiddleware(zap.NewNop(), builder, parser)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})).ServeHTTP(response, request)
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.False(t, called)
	require.Empty(t, response.Result().Cookies())
}

func TestGetUserID(t *testing.T) {
	require.Empty(t, getUserID(context.Background()))
	type unrelatedContextKey string
	ctx := context.WithValue(context.Background(), unrelatedContextKey("UserID"), "forged-user")
	require.Empty(t, getUserID(ctx))
	ctx = context.WithValue(ctx, contextKey{}, "user-id")
	require.Equal(t, "user-id", getUserID(ctx))
	require.Equal(t, "user-id", UserID(ctx))
}
