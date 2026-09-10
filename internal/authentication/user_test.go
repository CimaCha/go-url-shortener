package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
			builder := JWTBuilder{SecretKey: []byte("secret")}
			var gotUserID string
			next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				gotUserID = request.Header.Get("userID")
				writer.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			response := httptest.NewRecorder()

			AuthMiddleware(zap.NewNop(), builder)(next).ServeHTTP(response, request)

			require.Equal(t, http.StatusNoContent, response.Code)
			require.NotEmpty(t, gotUserID)
			cookies := response.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, "jwt", cookies[0].Name)
			require.Equal(t, tt.wantSecure, cookies[0].Secure)
			require.WithinDuration(t, time.Now().Add(TokenExp), cookies[0].Expires, 2*time.Second)
			userID, err := builder.GetUserID(cookies[0].Value)
			require.NoError(t, err)
			require.Equal(t, gotUserID, userID)
		})
	}
}

func TestAuthMiddlewareUsesValidCookie(t *testing.T) {
	builder := JWTBuilder{SecretKey: []byte("secret")}
	token, err := builder.BuildJWTString("existing-user", time.Hour)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	request.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	response := httptest.NewRecorder()
	called := false

	AuthMiddleware(zap.NewNop(), builder)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		called = true
		require.Equal(t, "existing-user", request.Header.Get("userID"))
	})).ServeHTTP(response, request)

	require.True(t, called)
}

func TestAuthMiddlewareRejectsInvalidCookie(t *testing.T) {
	builder := JWTBuilder{SecretKey: []byte("secret")}
	request := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	request.AddCookie(&http.Cookie{Name: "jwt", Value: "invalid"})
	response := httptest.NewRecorder()
	called := false

	AuthMiddleware(zap.NewNop(), builder)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})).ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.False(t, called)
}
