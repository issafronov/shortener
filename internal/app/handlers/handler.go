// Package handlers содержит HTTP-хендлеры для обработки запросов к сервису сокращения URL
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/service"
)

// Handler обрабатывает входящие HTTP-запросы и взаимодействует с сервисным слоем.
type Handler struct {
	service service.Service
	config  *config.Config
}

// NewHandler создает новый экземпляр Handler.
func NewHandler(cfg *config.Config, svc service.Service) (*Handler, error) {
	return &Handler{
		config:  cfg,
		service: svc,
	}, nil
}

// CreateLinkHandle обрабатывает POST-запрос с обычной строкой URL и создает сокращённую ссылку.
func (h *Handler) CreateLinkHandle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkeys.UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	originalURL := string(body)

	shortKey, err := h.service.CreateURL(r.Context(), originalURL, userID)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			h.respondWithText(w, r, shortKey, http.StatusConflict)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.respondWithText(w, r, shortKey, http.StatusCreated)
}

// CreateJSONLinkHandle обрабатывает POST-запрос с JSON-телом и создает сокращённую ссылку.
func (h *Handler) CreateJSONLinkHandle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkeys.UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var urlData models.URLData
	if err := json.NewDecoder(r.Body).Decode(&urlData); err != nil || urlData.URL == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	shortKey, err := h.service.CreateURL(r.Context(), urlData.URL, userID)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			h.respondWithJSON(w, r, shortKey, http.StatusConflict)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.respondWithJSON(w, r, shortKey, http.StatusCreated)
}

// CreateBatchJSONLinkHandle обрабатывает пакетный POST-запрос с JSON и возвращает множество сокращённых ссылок.
func (h *Handler) CreateBatchJSONLinkHandle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkeys.UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var batch []models.BatchURLData
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	result, err := h.service.CreateURLBatch(r.Context(), batch, userID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for i := range result {
		result[i].ShortURL = h.buildFullURL(r, result[i].ShortURL)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

// GetLinkHandle обрабатывает GET-запрос и перенаправляет на оригинальный URL по короткому ключу.
func (h *Handler) GetLinkHandle(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	originalURL, err := h.service.GetOriginalURL(r.Context(), key)
	if err != nil {
		if errors.Is(err, service.ErrDeleted) {
			http.Error(w, http.StatusText(http.StatusGone), http.StatusGone)
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
		return
	}
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

// GetUserLinksHandle возвращает список сокращённых ссылок пользователя.
func (h *Handler) GetUserLinksHandle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkeys.UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	links, err := h.service.GetUserURLs(r.Context(), userID, h.getBaseURL(r))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(links) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(links)
}

// DeleteLinksHandle обрабатывает запрос на удаление нескольких ссылок пользователя.
func (h *Handler) DeleteLinksHandle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkeys.UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	go func() {
		_ = h.service.DeleteUserURLs(r.Context(), userID, ids)
	}()

	w.WriteHeader(http.StatusAccepted)
}

// InternalStats возвращает статистику по числу пользователей и URL-адресов.
func (h *Handler) InternalStats(w http.ResponseWriter, r *http.Request) {
	urls, users, err := h.service.GetStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	stats := struct {
		URLs  int64 `json:"urls"`
		Users int64 `json:"users"`
	}{
		URLs:  urls,
		Users: users,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// getBaseURL возвращает базовый URL сервиса.
func (h *Handler) getBaseURL(r *http.Request) string {
	if h.config.BaseURL != "" {
		return h.config.BaseURL
	}
	return "http://" + r.Host
}

// respondWithText записывает ответ в виде обычного текста.
func (h *Handler) respondWithText(w http.ResponseWriter, r *http.Request, shortKey string, status int) {
	fullURL := h.buildFullURL(r, shortKey)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fullURL))
}

// respondWithJSON записывает JSON-ответ с сокращённой ссылкой.
func (h *Handler) respondWithJSON(w http.ResponseWriter, r *http.Request, shortKey string, status int) {
	fullURL := h.buildFullURL(r, shortKey)
	response := models.ShortURLData{Result: fullURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// buildFullURL формирует полный URL из базового и короткого ключа.
func (h *Handler) buildFullURL(r *http.Request, shortKey string) string {
	base := h.getBaseURL(r)
	return base + "/" + shortKey
}
