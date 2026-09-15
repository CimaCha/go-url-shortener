package authentication

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestJWTBuilder(t *testing.T) {
	builder := NewJWTBuilder([]byte("secret"))
	before := time.Now()
	tokenString, err := builder.BuildJWTString("user-id", time.Hour)
	require.NoError(t, err)
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		require.Equal(t, jwt.SigningMethodHS256, token.Method)
		return builder.SecretKey, nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, "user-id", claims.UserID)
	require.WithinDuration(t, before.Add(time.Hour), claims.ExpiresAt.Time, 2*time.Second)
}
