package authentication

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrTokenIsInvalid = errors.New("token is not valid")
	ErrNoUserIDExists = errors.New("user id is not exists in token claims")
)

//go:generate mockgen -source=jwt.go -destination=mocks/mock_jwt.gen.go -package=mocks

type TokenBuilder interface {
	BuildJWTString(userID string, tokenExp time.Duration) (string, error)
}

type JWTBuilder struct {
	SecretKey []byte
}

func NewJWTBuilder(secretKey []byte) *JWTBuilder {
	return &JWTBuilder{
		SecretKey: secretKey,
	}
}

// Claims — структура утверждений, которая включает стандартные утверждения
// и одно пользовательское — UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// BuildJWTString создаёт токен и возвращает его в виде строки.
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
