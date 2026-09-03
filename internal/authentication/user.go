package authentication

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const TokenExp = time.Hour * 3

func AuthMiddleware(log *zap.Logger, jwtBuilder JWTBuilder) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			jwtCookie, err := request.Cookie("jwt")
			if errors.Is(err, http.ErrNoCookie) {
				log.Info("user need auth")
				// если нет куки, генерируем id и устанавливаем куку
				id, err := generateUserID()
				if err != nil {
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
				jwtString, err := jwtBuilder.BuildJWTString(id, TokenExp)
				if err != nil {
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}

				request.Header.Set("userID", id)

				cookie := &http.Cookie{
					Name:     "jwt",
					Value:    jwtString,
					Path:     "/",                            // Available across the whole domain
					Expires:  time.Now().Add(24 * time.Hour), // Set to expire in 24 hours
					HttpOnly: true,                           // Prevents JavaScript access (XSS protection)
					Secure:   true,                           // Ensures cookie is only sent over HTTPS
					SameSite: http.SameSiteLaxMode,           // Controls cross-site request behavior
				}

				http.SetCookie(writer, cookie)

				handler.ServeHTTP(writer, request)
			} else {
				userID, err := jwtBuilder.GetUserID(jwtCookie.Value)
				if errors.Is(err, ErrNoUserIDExists) {
					http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				}
				request.Header.Set("userID", userID)
				handler.ServeHTTP(writer, request)
			}
		})
	}
}

func generateUserID() (string, error) {
	// определяем слайс байт нужной длины
	b := make([]byte, 16)
	_, err := rand.Read(b) // записываем байты в слайс b
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return "", err
	}
	return hex.EncodeToString(b), nil
}
