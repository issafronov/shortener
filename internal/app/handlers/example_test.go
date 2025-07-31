package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/service"
	"github.com/issafronov/shortener/internal/app/storage"
	"golang.org/x/net/context"
)

// Example of creating a short link via CreateJSONLinkHandle.
func ExampleHandler_CreateJSONLinkHandle() {
	cfg := &config.Config{
		BaseURL: "http://localhost",
	}

	svc := &mockService{}

	h, _ := handlers.NewHandler(cfg, svc)

	urlData := models.URLData{URL: "https://example.com"}
	body, _ := json.Marshal(urlData)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.CreateJSONLinkHandle(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	// Output:
	// Status: 201
}

// Example of getting a short link via GetLinkHandle.
func ExampleHandler_GetLinkHandle() {
	cfg := &config.Config{
		BaseURL:         "http://localhost",
		FileStoragePath: "../storage/test_storage.json",
	}
	store, _ := storage.NewFileStorage(cfg)
	svc := service.NewService(store)
	h, _ := handlers.NewHandler(cfg, svc)

	// Вставка ссылки через сервис (корректно)
	shortURL := "abc123"
	originalURL := "https://example.com"
	userID := "user1"

	// Предварительно добавим ссылку в хранилище
	storage.Urls["abc123"] = storage.ShortenerURL{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "user1",
	}

	_, _ = store.Create(context.Background(), storage.ShortenerURL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		UserID:      userID,
	})

	// Создаём запрос
	req := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, userID))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.HostKey, "http://localhost"))

	// Мокаем chi params
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("key", shortURL)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	w := httptest.NewRecorder()

	h.GetLinkHandle(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Location:", resp.Header.Get("Location"))

	// Output:
	// Status: 307
	// Location: https://example.com
}
