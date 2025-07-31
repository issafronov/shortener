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

type mockService struct {
	CreateURLFunc      func(ctx context.Context, originalURL, userID string) (string, error)
	CreateURLBatchFunc func(ctx context.Context, batch []models.BatchURLData, userID string) ([]models.BatchURLDataResponse, error)
	GetOriginalURLFunc func(ctx context.Context, shortKey string) (string, error)
	GetUserURLsFunc    func(ctx context.Context, userID, host string) ([]models.ShortURLResponse, error)
	DeleteUserURLsFunc func(ctx context.Context, userID string, ids []string) error
	GetStatsFunc       func(ctx context.Context) (int64, int64, error)
	PingFunc           func(ctx context.Context) error
}

func (m *mockService) CreateURL(ctx context.Context, originalURL, userID string) (string, error) {
	if m.CreateURLFunc != nil {
		return m.CreateURLFunc(ctx, originalURL, userID)
	}
	return "", nil
}

func (m *mockService) CreateURLBatch(ctx context.Context, batch []models.BatchURLData, userID string) ([]models.BatchURLDataResponse, error) {
	if m.CreateURLBatchFunc != nil {
		return m.CreateURLBatchFunc(ctx, batch, userID)
	}
	return nil, nil
}

func (m *mockService) GetOriginalURL(ctx context.Context, shortKey string) (string, error) {
	if m.GetOriginalURLFunc != nil {
		return m.GetOriginalURLFunc(ctx, shortKey)
	}
	return "", nil
}

func (m *mockService) GetUserURLs(ctx context.Context, userID, host string) ([]models.ShortURLResponse, error) {
	if m.GetUserURLsFunc != nil {
		return m.GetUserURLsFunc(ctx, userID, host)
	}
	return nil, nil
}

func (m *mockService) DeleteUserURLs(ctx context.Context, userID string, ids []string) error {
	if m.DeleteUserURLsFunc != nil {
		return m.DeleteUserURLsFunc(ctx, userID, ids)
	}
	return nil
}

func (m *mockService) GetStats(ctx context.Context) (int64, int64, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(ctx)
	}
	return 0, 0, nil
}

func (m *mockService) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

type mockStorage struct {
	GetFunc        func(ctx context.Context, key string) (string, error)
	GetByUserFunc  func(ctx context.Context, userID string) ([]models.ShortURLResponse, error)
	CreateFunc     func(ctx context.Context, url storage.ShortenerURL) (string, error)
	DeleteURLsFunc func(ctx context.Context, userID string, ids []string) error
	CountURLsFunc  func(ctx context.Context) (int64, error)
	CountUsersFunc func(ctx context.Context) (int64, error)
}

func (m *mockStorage) Get(ctx context.Context, key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return "", nil
}

func (m *mockStorage) GetByUser(ctx context.Context, userID string) ([]models.ShortURLResponse, error) {
	if m.GetByUserFunc != nil {
		return m.GetByUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockStorage) Create(ctx context.Context, url storage.ShortenerURL) (string, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, url)
	}
	return "", nil
}

func (m *mockStorage) DeleteURLs(ctx context.Context, userID string, ids []string) error {
	if m.DeleteURLsFunc != nil {
		return m.DeleteURLsFunc(ctx, userID, ids)
	}
	return nil
}

func (m *mockStorage) CountURLs(ctx context.Context) (int64, error) {
	if m.CountURLsFunc != nil {
		return m.CountURLsFunc(ctx)
	}
	return 0, nil
}

func (m *mockStorage) CountUsers(ctx context.Context) (int64, error) {
	if m.CountUsersFunc != nil {
		return m.CountUsersFunc(ctx)
	}
	return 0, nil
}

func TestCreateJSONLinkHandle(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost"}
	svc := &mockService{
		GetStatsFunc: func(ctx context.Context) (int64, int64, error) {
			return 42, 10, nil
		},
		CreateURLFunc: func(ctx context.Context, originalURL, userID string) (string, error) {
			return "shortKey", nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		GetStatsFunc: func(ctx context.Context) (int64, int64, error) {
			return 42, 10, nil
		},
		CreateURLFunc: func(ctx context.Context, originalURL, userID string) (string, error) {
			return "shortKey", nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	h.CreateJSONLinkHandle(w, req)
	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestCreateJSONLinkHandle_InvalidBody(t *testing.T) {
	cfg := &config.Config{}
	svc := &mockService{
		GetStatsFunc: func(ctx context.Context) (int64, int64, error) {
			return 42, 10, nil
		},
		CreateURLFunc: func(ctx context.Context, originalURL, userID string) (string, error) {
			return "shortKey", nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		GetOriginalURLFunc: func(ctx context.Context, key string) (string, error) {
			return "https://example.com", nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		GetUserURLsFunc: func(ctx context.Context, userID, host string) ([]models.ShortURLResponse, error) {
			return []models.ShortURLResponse{}, nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		DeleteUserURLsFunc: func(ctx context.Context, userID string, ids []string) error {
			return nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		CreateURLFunc: func(ctx context.Context, originalURL, userID string) (string, error) {
			return "shortKey", nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		CreateURLBatchFunc: func(ctx context.Context, batch []models.BatchURLData, userID string) ([]models.BatchURLDataResponse, error) {
			resp := make([]models.BatchURLDataResponse, len(batch))
			for i, item := range batch {
				resp[i] = models.BatchURLDataResponse{
					CorrelationID: item.CorrelationID,
					ShortURL:      "shortKey" + item.CorrelationID,
				}
			}
			return resp, nil
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
	svc := &mockService{
		GetOriginalURLFunc: func(ctx context.Context, key string) (string, error) {
			return "", errors.New("url not found")
		},
	}

	h, _ := handlers.NewHandler(cfg, svc)

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
