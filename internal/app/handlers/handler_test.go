package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"
)

type mockStorage struct {
	CreateFunc     func(ctx context.Context, url storage.ShortenerURL) (string, error)
	GetFunc        func(ctx context.Context, key string) (string, error)
	GetByUserFunc  func(ctx context.Context, userID string) ([]models.ShortURLResponse, error)
	DeleteURLsFunc func(ctx context.Context, userID string, ids []string) error
	PingFunc       func(ctx context.Context) error
}

func (m *mockStorage) Create(ctx context.Context, url storage.ShortenerURL) (string, error) {
	return m.CreateFunc(ctx, url)
}

func (m *mockStorage) Get(ctx context.Context, key string) (string, error) {
	return m.GetFunc(ctx, key)
}

func (m *mockStorage) GetByUser(ctx context.Context, userID string) ([]models.ShortURLResponse, error) {
	return m.GetByUserFunc(ctx, userID)
}

func (m *mockStorage) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	return m.DeleteURLsFunc(ctx, userID, ids)
}

func (m *mockStorage) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func TestCreateJSONLinkHandle(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost"}
	storage := &mockStorage{
		CreateFunc: func(ctx context.Context, url storage.ShortenerURL) (string, error) {
			return url.ShortURL, nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	data := models.URLData{URL: "https://example.com"}
	body, _ := json.Marshal(data)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.CreateJSONLinkHandle(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode)
}

func TestCreateJSONLinkHandle_Unauthorized(t *testing.T) {
	cfg := &config.Config{}
	h, _ := handlers.NewHandler(cfg, &mockStorage{})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	h.CreateJSONLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestCreateJSONLinkHandle_InvalidBody(t *testing.T) {
	cfg := &config.Config{}
	h, _ := handlers.NewHandler(cfg, &mockStorage{})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("invalid-json")))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.CreateJSONLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestGetLinkHandle_Found(t *testing.T) {
	cfg := &config.Config{}
	storage := &mockStorage{
		GetFunc: func(ctx context.Context, key string) (string, error) {
			return "https://example.com", nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	r := chi.NewRouter()
	r.Get("/{key}", h.GetLinkHandle)

	ts := httptest.NewServer(r)
	defer ts.Close()

	req := httptest.NewRequest(http.MethodGet, ts.URL+"/abc123", nil)
	w := httptest.NewRecorder()

	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("key", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))

	h.GetLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
}

func TestGetUserLinksHandle_NoContent(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost"}
	storage := &mockStorage{
		GetByUserFunc: func(ctx context.Context, userID string) ([]models.ShortURLResponse, error) {
			return []models.ShortURLResponse{}, nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	req := httptest.NewRequest(http.MethodGet, "/user/urls", nil)
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.GetUserLinksHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestDeleteLinksHandle(t *testing.T) {
	cfg := &config.Config{}
	storage := &mockStorage{
		DeleteURLsFunc: func(ctx context.Context, userID string, ids []string) error {
			return nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	body, _ := json.Marshal([]string{"abc123"})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.DeleteLinksHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
}

func TestCreateLinkHandle(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost"}
	storage := &mockStorage{
		CreateFunc: func(ctx context.Context, url storage.ShortenerURL) (string, error) {
			return url.ShortURL, nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	body := []byte("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.CreateLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode)
}

func TestCreateBatchJSONLinkHandle(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost"}
	storage := &mockStorage{
		CreateFunc: func(ctx context.Context, url storage.ShortenerURL) (string, error) {
			return url.ShortURL, nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	batch := []models.BatchURLData{
		{CorrelationID: "1", OriginalURL: "https://example1.com"},
		{CorrelationID: "2", OriginalURL: "https://example2.com"},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.CreateBatchJSONLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode)
}

func TestGetLinkHandle_NotFound(t *testing.T) {
	cfg := &config.Config{}
	storage := &mockStorage{
		GetFunc: func(ctx context.Context, key string) (string, error) {
			return "", errors.New("url not found")
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("key", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	w := httptest.NewRecorder()

	h.GetLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestGetLinkHandle_Deleted(t *testing.T) {
	cfg := &config.Config{}
	storage := &mockStorage{
		GetFunc: func(ctx context.Context, key string) (string, error) {
			return "", errors.New("url gone")
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("key", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	w := httptest.NewRecorder()

	h.GetLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusGone, res.StatusCode)
}

func TestGetUserLinksHandle_OK(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost"}
	storage := &mockStorage{
		GetByUserFunc: func(ctx context.Context, userID string) ([]models.ShortURLResponse, error) {
			return []models.ShortURLResponse{
				{
					ShortURL:    "http://localhost/abc123",
					OriginalURL: "https://example.com",
				},
			}, nil
		},
	}
	h, _ := handlers.NewHandler(cfg, storage)

	req := httptest.NewRequest(http.MethodGet, "/user/urls", nil)
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.GetUserLinksHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
