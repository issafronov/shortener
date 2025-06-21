package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/security"
	"github.com/issafronov/shortener/internal/app/utils"
	"github.com/issafronov/shortener/internal/middleware/logger"
)

// AuthorizationMiddleware — middleware для аутентификации пользователя с помощью JWT токена
func AuthorizationMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := r.Cookie("JWT_TOKEN")
		var userID string
		if err != nil {
			logger.Log.Debug("AuthorizationMiddleware: no JWT_TOKEN cookie")
			userID = utils.CreateShortKey(10)
			signedCookie, err := security.GenerateJWT(userID)
			if err != nil {
				logger.Log.Info("AuthorizationMiddleware: error generating JWT token")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			cookieValue := &http.Cookie{
				Name:    "JWT_TOKEN",
				Value:   signedCookie,
				Expires: time.Now().Add(time.Hour * 24),
			}
			http.SetCookie(w, cookieValue)
		} else {
			claims := &security.Claims{}
			secret := os.Getenv("SECRET_KEY")
			if secret == "" {
				secret = "secret"
			}
			token, err := jwt.ParseWithClaims(tokenString.Value, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					logger.Log.Debug("AuthorizationMiddleware: token expired")
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				logger.Log.Debug("AuthorizationMiddleware: error parsing JWT token")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if !token.Valid {
				logger.Log.Debug("AuthorizationMiddleware: invalid JWT token")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			userID = claims.UserID
		}
		ctx := context.WithValue(r.Context(), contextkeys.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
