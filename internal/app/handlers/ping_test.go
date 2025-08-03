package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/stretchr/testify/assert"
)

func TestPing_Success(t *testing.T) {
	svc := &mockService{
		PingFunc: func(ctx context.Context) error {
			return nil
		},
	}

	cfg := &config.Config{}
	h, _ := handlers.NewHandler(cfg, svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.Ping(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestPing_Failure(t *testing.T) {
	svc := &mockService{
		PingFunc: func(ctx context.Context) error {
			return errors.New("db connection failed")
		},
	}

	cfg := &config.Config{}
	h, _ := handlers.NewHandler(cfg, svc)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.Ping(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
}
