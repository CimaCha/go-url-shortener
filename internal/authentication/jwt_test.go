package authentication

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestJWTBuilder(t *testing.T) {
	builder := JWTBuilder{SecretKey: []byte("secret")}

	t.Run("round trip", func(t *testing.T) {
		token, err := builder.BuildJWTString("user-id", time.Hour)
		require.NoError(t, err)

		userID, err := builder.GetUserID(token)
		require.NoError(t, err)
		require.Equal(t, "user-id", userID)
	})

	tests := []struct {
		name  string
		token func(*testing.T) string
		err   error
	}{
		{name: "malformed token", token: func(*testing.T) string { return "not-a-token" }, err: ErrTokenIsInvalid},
		{
			name: "wrong secret",
			token: func(t *testing.T) string {
				token, err := (JWTBuilder{SecretKey: []byte("other")}).BuildJWTString("user-id", time.Hour)
				require.NoError(t, err)
				return token
			},
			err: ErrTokenIsInvalid,
		},
		{
			name: "expired token",
			token: func(t *testing.T) string {
				token, err := builder.BuildJWTString("user-id", -time.Hour)
				require.NoError(t, err)
				return token
			},
			err: ErrTokenIsInvalid,
		},
		{
			name: "empty user id",
			token: func(t *testing.T) string {
				token, err := builder.BuildJWTString("", time.Hour)
				require.NoError(t, err)
				return token
			},
			err: ErrNoUserIDExists,
		},
		{
			name: "different HMAC algorithm",
			token: func(t *testing.T) string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
					RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
					UserID:           "user-id",
				})
				signed, err := token.SignedString(builder.SecretKey)
				require.NoError(t, err)
				return signed
			},
			err: ErrTokenIsInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := builder.GetUserID(tt.token(t))
			require.Empty(t, userID)
			require.True(t, errors.Is(err, tt.err), "error = %v, want %v", err, tt.err)
		})
	}
}
