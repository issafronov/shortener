package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/security"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizationMiddleware_NoCookie(t *testing.T) {
	_ = os.Unsetenv("SECRET_KEY")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	var capturedUserID string
	handler := AuthorizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(contextkeys.UserIDKey)
		userID, ok := val.(string)
		assert.True(t, ok)
		assert.NotEmpty(t, userID)
		capturedUserID = userID
	}))

	handler.ServeHTTP(rr, req)
	res := rr.Result()
	defer res.Body.Close()

	cookies := res.Cookies()
	var jwtToken string
	for _, c := range cookies {
		if c.Name == "JWT_TOKEN" {
			jwtToken = c.Value
			break
		}
	}
	assert.NotEmpty(t, jwtToken)

	parsed, err := jwt.ParseWithClaims(jwtToken, &security.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims := parsed.Claims.(*security.Claims)
	assert.Equal(t, capturedUserID, claims.UserID)
}

func TestAuthorizationMiddleware_ValidCookie(t *testing.T) {
	_ = os.Setenv("SECRET_KEY", "mytestsecret")
	userID := "existingUser123"
	token, err := security.GenerateJWT(userID)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JWT_TOKEN",
		Value: token,
	})
	rr := httptest.NewRecorder()

	handler := AuthorizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(contextkeys.UserIDKey)
		assert.Equal(t, userID, val)
	}))

	handler.ServeHTTP(rr, req)
}

func TestAuthorizationMiddleware_InvalidCookie(t *testing.T) {
	_ = os.Setenv("SECRET_KEY", "mytestsecret")

	token, _ := security.GenerateJWT("user1")
	tampered := token + "abc"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JWT_TOKEN",
		Value: tampered,
	})
	rr := httptest.NewRecorder()

	handler := AuthorizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Handler should not be called for invalid token")
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAuthorizationMiddleware_InvalidSignature(t *testing.T) {
	os.Setenv("SECRET_KEY", "goodsecret")
	token, err := security.GenerateJWT("userX")
	assert.NoError(t, err)

	os.Setenv("SECRET_KEY", "wrongsecret")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JWT_TOKEN",
		Value: token,
	})
	rr := httptest.NewRecorder()

	handler := AuthorizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Should not enter handler for invalid signature")
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAuthorizationMiddleware_ExpiredToken(t *testing.T) {
	_ = os.Setenv("SECRET_KEY", "expiresecret")

	expiredClaims := &security.Claims{
		UserID: "expiredUser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenStr, err := token.SignedString([]byte("expiresecret"))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "JWT_TOKEN",
		Value: tokenStr,
	})
	rr := httptest.NewRecorder()

	handler := AuthorizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Should not process request with expired token")
	}))

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
