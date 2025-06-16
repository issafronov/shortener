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
	"github.com/issafronov/shortener/internal/app/storage"
	"golang.org/x/net/context"
)

// Example of creating a short link via CreateJSONLinkHandle.
func ExampleHandler_CreateJSONLinkHandle() {
	cfg := &config.Config{
		BaseURL:         "http://localhost",
		FileStoragePath: "../storage/test_storage.json",
	}
	store, _ := storage.NewFileStorage(cfg)
	h, _ := handlers.NewHandler(cfg, store)

	// Создаём JSON-запрос
	urlData := models.URLData{URL: "https://example.com"}
	body, _ := json.Marshal(urlData)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	w := httptest.NewRecorder()

	h.CreateJSONLinkHandle(w, req)
	resp := w.Result()

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
	h, _ := handlers.NewHandler(cfg, store)

	// Предварительно добавим ссылку в хранилище
	storage.Urls["abc123"] = storage.ShortenerURL{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "user1",
	}

	// Создаём запрос
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.UserIDKey, "user1"))
	req = req.WithContext(context.WithValue(req.Context(), contextkeys.HostKey, "http://localhost"))

	// Мокаем параметр маршрута chi
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("key", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	w := httptest.NewRecorder()

	h.GetLinkHandle(w, req)
	resp := w.Result()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Location:", resp.Header.Get("Location"))
	// Output:
	// Status: 307
	// Location: https://example.com
}
