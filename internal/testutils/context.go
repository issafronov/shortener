package testutils

import (
	"context"
	"net/http"

	"github.com/issafronov/shortener/internal/app/contextkeys"
)

// WithTestUserContext хэлпер для тестов
func WithTestUserContext(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), contextkeys.UserIDKey, userID)
	return r.WithContext(ctx)
}
