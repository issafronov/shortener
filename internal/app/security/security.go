package security

import (
	"github.com/issafronov/shortener/internal/middleware/logger"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

const exp = time.Hour * 24

func GenerateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
		},
		UserID: userID,
	})

	key := os.Getenv("SECRET_KEY")
	if key == "" {
		logger.Log.Info("SECRET_KEY environment variable not set")
		key = "secret"
	}

	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
