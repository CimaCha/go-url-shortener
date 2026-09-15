package authentication

import "github.com/golang-jwt/jwt/v4"

//go:generate mockgen -source=parser.go -destination=mocks/mock_parser.gen.go -package=mocks

type UserIDParser interface {
	GetUserID(tokenString string) (string, error)
}

type Parser struct {
	SecretKey []byte
}

func NewUserIDParser(secretKey []byte) *Parser {
	return &Parser{
		SecretKey: secretKey,
	}
}

func (p Parser) GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, ErrTokenIsInvalid
			}
			return p.SecretKey, nil
		})
	if err != nil || token == nil || !token.Valid {
		return "", ErrTokenIsInvalid
	}
	if claims.UserID == "" {
		return "", ErrNoUserIDExists
	}
	return claims.UserID, nil
}
