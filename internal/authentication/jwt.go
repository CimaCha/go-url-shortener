package authentication

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	// ErrTokenIsInvalid indicates an invalid or unverifiable JWT.
	ErrTokenIsInvalid = errors.New("token is not valid")
	// ErrNoUserIDExists indicates a valid JWT without a user ID.
	ErrNoUserIDExists = errors.New("user id is not exists in token claims")
)

//go:generate mockgen -source=jwt.go -destination=mocks/mock_jwt.gen.go -package=mocks

// TokenBuilder creates signed JWTs for user IDs.
type TokenBuilder interface {
	BuildJWTString(userID string, tokenExp time.Duration) (string, error)
}

// JWTBuilder signs user tokens with SecretKey.
type JWTBuilder struct {
	SecretKey []byte
}

// NewJWTBuilder creates a JWT signer with the given secret key.
func NewJWTBuilder(secretKey []byte) *JWTBuilder {
	return &JWTBuilder{
		SecretKey: secretKey,
	}
}

// Claims contains standard JWT claims and the application user ID.
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// BuildJWTString signs a token for userID that expires after tokenExp.
func (b JWTBuilder) BuildJWTString(userID string, tokenExp time.Duration) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString(b.SecretKey)
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}
