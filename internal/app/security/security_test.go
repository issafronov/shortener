package security

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestGenerateJWT(t *testing.T) {
	_ = os.Setenv("SECRET_KEY", "testsecret")

	userID := "testuser123"
	tokenStr, err := GenerateJWT(userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("testsecret"), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(*Claims)
	assert.True(t, ok)
	assert.Equal(t, userID, claims.UserID)
	assert.WithinDuration(t, time.Now().Add(exp), claims.ExpiresAt.Time, time.Minute)
}

func TestGenerateJWT_DefaultSecret(t *testing.T) {
	_ = os.Unsetenv("SECRET_KEY") // No env var, should use default

	userID := "defaultuser"
	tokenStr, err := GenerateJWT(userID)

	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil // default fallback
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(*Claims)
	assert.True(t, ok)
	assert.Equal(t, userID, claims.UserID)
}
