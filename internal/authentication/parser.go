package authentication

import "github.com/golang-jwt/jwt/v4"

//go:generate mockgen -source=parser.go -destination=mocks/mock_parser.gen.go -package=mocks

// UserIDParser extracts a user ID from a signed token.
type UserIDParser interface {
	GetUserID(tokenString string) (string, error)
}

// Parser verifies JWTs with SecretKey.
type Parser struct {
	SecretKey []byte
}

// NewUserIDParser creates a JWT parser with the given secret key.
func NewUserIDParser(secretKey []byte) *Parser {
	return &Parser{
		SecretKey: secretKey,
	}
}

// GetUserID verifies a token and returns its nonempty user ID.
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
