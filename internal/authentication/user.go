package authentication

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const TokenExp = time.Hour * 3

func AuthMiddleware(log *zap.Logger, jwtBuilder JWTBuilder) func(http.Handler) http.Handler {
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

				request.Header.Set("userID", id)

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

			userID, err := jwtBuilder.GetUserID(jwtCookie.Value)
			if err != nil {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			request.Header.Set("userID", userID)
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
