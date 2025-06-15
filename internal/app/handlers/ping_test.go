package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"
)

type mockStorageWithPing struct {
	PingFunc func(ctx context.Context) error
}

func (m *mockStorageWithPing) Ping(ctx context.Context) error {
	return m.PingFunc(ctx)
}

func (m *mockStorageWithPing) Create(ctx context.Context, url storage.ShortenerURL) (string, error) {
	return "", nil
}
func (m *mockStorageWithPing) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (m *mockStorageWithPing) GetByUser(ctx context.Context, userID string) ([]models.ShortURLResponse, error) {
	return nil, nil
}
func (m *mockStorageWithPing) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	return nil
}

func TestPing_Success(t *testing.T) {
	storage := &mockStorageWithPing{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	cfg := &config.Config{}
	h, _ := handlers.NewHandler(cfg, storage)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.Ping(w, req)

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestPing_Failure(t *testing.T) {
	storage := &mockStorageWithPing{
		PingFunc: func(ctx context.Context) error {
			return errors.New("db connection failed")
		},
	}

	cfg := &config.Config{}
	h, _ := handlers.NewHandler(cfg, storage)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.Ping(w, req)

	res := w.Result()
	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}
