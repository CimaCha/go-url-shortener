package authentication

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const TokenExp = time.Hour * 3

type contextKey struct{}

func getUserID(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// UserID returns the authenticated user ID, or an empty string if absent.
func UserID(ctx context.Context) string {
	return getUserID(ctx)
}

func AuthMiddleware(log *zap.Logger, jwtBuilder TokenBuilder, userIDParser UserIDParser) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			jwtCookie, err := request.Cookie("jwt")
			if errors.Is(err, http.ErrNoCookie) {
				log.Info("user need auth")
				id, err := generateUserID()
				if err != nil {
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				jwtString, err := jwtBuilder.BuildJWTString(id, TokenExp)
				if err != nil {
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				request = request.WithContext(context.WithValue(request.Context(), contextKey{}, id))

				cookie := &http.Cookie{
					Name:     "jwt",
					Value:    jwtString,
					Path:     "/",
					Expires:  time.Now().Add(TokenExp),
					HttpOnly: true,
					Secure:   request.TLS != nil,
					SameSite: http.SameSiteLaxMode,
				}

				http.SetCookie(writer, cookie)

				handler.ServeHTTP(writer, request)
				return
			}
			if err != nil {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID, err := userIDParser.GetUserID(jwtCookie.Value)
			if err != nil {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			request = request.WithContext(context.WithValue(request.Context(), contextKey{}, userID))
			handler.ServeHTTP(writer, request)
		})
	}
}

func generateUserID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
