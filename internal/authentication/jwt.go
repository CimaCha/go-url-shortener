package authentication

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

var (
	ErrTokenIsInvalid = errors.New("token is not valid")
	ErrNoUserIdExists = errors.New("user id is not exists in token claims")
)

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
func (b JWTBuilder) BuildJWTString(userId string, tokenExp time.Duration) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		// собственное утверждение
		UserID: userId,
	})

	// создаём строку токена
	tokenString, err := token.SignedString(b.SecretKey)
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func (b JWTBuilder) GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return b.SecretKey, nil
		})
	var validationError = jwt.ValidationError{}
	if errors.As(err, &validationError) {
		if validationError.Errors == jwt.ValidationErrorClaimsInvalid {
			return "", ErrNoUserIdExists
		}
	}

	if !token.Valid {
		return "", ErrTokenIsInvalid
	}

	fmt.Println("Token is valid")
	return claims.UserID, nil
}
