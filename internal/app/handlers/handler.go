// Package handlers содержит HTTP-хендлеры для обработки запросов к сервису сокращения URL
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/app/utils"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
)

const (
	shortKeyLength = 8
)

// Handler реализует HTTP-хендлеры для работы с сокращёнными ссылками
type Handler struct {
	config  *config.Config
	storage storage.Storage
}

// NewHandler создаёт и возвращает новый экземпляр Handler
func NewHandler(config *config.Config, s storage.Storage) (*Handler, error) {
	return &Handler{
		config:  config,
		storage: s,
	}, nil
}

// WriteURL сохраняет переданную структуру ShortenerURL в хранилище и возвращает сгенерированную ссылку
func (h *Handler) WriteURL(ctx context.Context, url storage.ShortenerURL) (string, error) {
	logger.Log.Info("Writing URL", zap.String("url", url.ShortURL))
	return h.storage.Create(ctx, url)
}

// CreateLinkHandle обрабатывает POST-запрос с text/plain и создаёт сокращённый URL.
// Возвращает 201 Created и короткий URL в теле ответа.
func (h *Handler) CreateLinkHandle(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(contextkeys.UserIDKey).(string)
	if !ok {
		logger.Log.Info("Failed to get user ID")
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Log.Info("Error reading body", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	originalURL := string(body)
	logger.Log.Info("CreateLinkHandle", zap.String("originalURL", originalURL))

	if originalURL == "" {
		logger.Log.Info("Empty originalURL")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	shortKey := utils.CreateShortKey(shortKeyLength)
	uuid := len(storage.Urls) + 1

	shortenerURL := storage.ShortenerURL{
		UUID:        uuid,
		ShortURL:    shortKey,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	storage.Urls[shortKey] = shortenerURL
	storage.UsersUrls[userID] = []string{shortKey, originalURL}

	if key, err := h.WriteURL(req.Context(), shortenerURL); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			res.Header().Set("Content-Type", "text/plain")
			res.WriteHeader(http.StatusConflict)
			resultHostAddr := "http://" + req.Host
			if h.config.BaseURL != "" {
				resultHostAddr = h.config.BaseURL
			}
			_, _ = res.Write([]byte(resultHostAddr + "/" + key))
			return
		}
		logger.Log.Info("Failed to write shortener URL", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusCreated)
	resultHostAddr := "http://" + req.Host

	if h.config.BaseURL != "" {
		resultHostAddr = h.config.BaseURL
	}

	_, err = res.Write([]byte(resultHostAddr + "/" + shortKey))

	if err != nil {
		logger.Log.Info("Error writing response", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// GetLinkHandle обрабатывает GET-запрос для получения оригинального URL по ключу.
// Перенаправляет пользователя на исходный адрес или возвращает 404/410
func (h *Handler) GetLinkHandle(res http.ResponseWriter, req *http.Request) {
	logger.Log.Info("GetLinkHandle", zap.String("url", req.URL.String()))
	key := chi.URLParam(req, "key")
	link, err := h.storage.Get(req.Context(), key)

	if err != nil {
		if err.Error() == "url gone" {
			logger.Log.Info("Link is deleted", zap.String("key", key))
			http.Error(res, http.StatusText(http.StatusGone), http.StatusGone)
			return
		}
		logger.Log.Info("Error getting link", zap.String("key", key), zap.Error(err))
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	logger.Log.Info("Link found", zap.String("key", key))
	http.Redirect(res, req, link, http.StatusTemporaryRedirect)
}

// CreateJSONLinkHandle обрабатывает POST-запрос с JSON-телом и создаёт сокращённый URL.
// Возвращает 201 Created и JSON-ответ с результатом
func (h *Handler) CreateJSONLinkHandle(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(contextkeys.UserIDKey).(string)

	if !ok {
		logger.Log.Info("Failed to get user ID")
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	logger.Log.Debug("decoding request")
	var urlData models.URLData
	dec := json.NewDecoder(req.Body)

	if err := dec.Decode(&urlData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	originalURL := urlData.URL
	logger.Log.Info("CreateLinkHandle", zap.String("originalURL", originalURL))

	if originalURL == "" {
		logger.Log.Info("Empty originalURL")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	shortKey := utils.CreateShortKey(shortKeyLength)
	uuid := len(storage.Urls) + 1
	shortenerURL := storage.ShortenerURL{
		UUID:        uuid,
		ShortURL:    shortKey,
		OriginalURL: originalURL,
		UserID:      userID,
	}
	storage.Urls[shortKey] = shortenerURL
	storage.UsersUrls[userID] = []string{shortKey, originalURL}

	if key, err := h.WriteURL(req.Context(), shortenerURL); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			res.Header().Set("Content-Type", "application/json")
			res.WriteHeader(http.StatusConflict)
			resultHostAddr := "http://" + req.Host
			if h.config.BaseURL != "" {
				resultHostAddr = h.config.BaseURL
			}
			shortURLData := models.ShortURLData{
				Result: resultHostAddr + "/" + key,
			}
			enc := json.NewEncoder(res)

			if err := enc.Encode(shortURLData); err != nil {
				logger.Log.Debug("error encoding response", zap.Error(err))
				return
			}
			return
		}
		logger.Log.Info("Failed to write shortener URL", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resultHostAddr := "http://" + req.Host

	if h.config.BaseURL != "" {
		resultHostAddr = h.config.BaseURL
	}

	shortURLData := models.ShortURLData{
		Result: resultHostAddr + "/" + shortKey,
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(res)

	if err := enc.Encode(shortURLData); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

// CreateBatchJSONLinkHandle обрабатывает POST-запрос с массивом URL в формате JSON.
// Создаёт сокращённые ссылки пакетно и возвращает массив результатов
func (h *Handler) CreateBatchJSONLinkHandle(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(contextkeys.UserIDKey).(string)

	if !ok {
		logger.Log.Info("Failed to get user ID")
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	logger.Log.Debug("CreateBatchJSONLinkHandle: decoding request")
	var batchURLData []models.BatchURLData
	dec := json.NewDecoder(req.Body)

	if err := dec.Decode(&batchURLData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var BatchURLDataResponse []models.BatchURLDataResponse

	for _, batch := range batchURLData {
		if batch.OriginalURL == "" || batch.CorrelationID == "" {
			logger.Log.Info("Empty originalURL or correlationID, skipping")
			continue
		}
		shortKey := utils.CreateShortKey(shortKeyLength)
		shortenerURL := &storage.ShortenerURL{
			ShortURL:      shortKey,
			OriginalURL:   batch.OriginalURL,
			CorrelationID: batch.CorrelationID,
			UserID:        userID,
		}

		if key, err := h.WriteURL(req.Context(), *shortenerURL); err != nil {
			logger.Log.Info("Failed to write shortener URL", zap.Error(err))
			resultHostAddr := "http://" + req.Host

			if h.config.BaseURL != "" {
				resultHostAddr = h.config.BaseURL
			}

			BatchURLDataResponse = append(BatchURLDataResponse, models.BatchURLDataResponse{
				ShortURL:      resultHostAddr + "/" + key,
				CorrelationID: batch.CorrelationID,
			})
			continue
		}

		resultHostAddr := "http://" + req.Host

		if h.config.BaseURL != "" {
			resultHostAddr = h.config.BaseURL
		}

		BatchURLDataResponse = append(BatchURLDataResponse, models.BatchURLDataResponse{
			ShortURL:      resultHostAddr + "/" + shortKey,
			CorrelationID: batch.CorrelationID,
		})
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(res)

	if err := enc.Encode(BatchURLDataResponse); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

// GetUserLinksHandle возвращает список всех сокращённых URL, созданных авторизованным пользователем.
// Возвращает 200 OK и JSON-ответ или 204, если список пуст
func (h *Handler) GetUserLinksHandle(res http.ResponseWriter, req *http.Request) {
	userID, ok := req.Context().Value(contextkeys.UserIDKey).(string)

	if !ok {
		logger.Log.Info("Failed to get user ID")
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	resultHostAddr := "http://" + req.Host

	if h.config.BaseURL != "" {
		resultHostAddr = h.config.BaseURL
	}
	ctx := context.WithValue(req.Context(), contextkeys.HostKey, resultHostAddr)
	ShortURLResponse, err := h.storage.GetByUser(ctx, userID)
	if err != nil {
		logger.Log.Info("Failed to get shortener URL")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(ShortURLResponse) == 0 {
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusNoContent)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(res)
	if err := enc.Encode(ShortURLResponse); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

// DeleteLinksHandle обрабатывает асинхронное удаление ссылок по списку ID для авторизованного пользователя.
// Возвращает 202 Accepted
func (h *Handler) DeleteLinksHandle(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextkeys.UserIDKey).(string)

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Используем паттерн fan-in
	chunkSize := 10 // Размер каждой "порции" URL, которую будем обрабатывать параллельно
	totalChunks := len(ids) / chunkSize
	if len(ids)%chunkSize != 0 {
		totalChunks++
	}

	resultChan := make(chan error, totalChunks)

	// Разбиваем список на чанки и запускаем горутины
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}

		go func(chunk []string) {
			err := h.storage.DeleteURLs(r.Context(), userID, chunk)
			resultChan <- err
		}(ids[start:end])
	}

	// Собираем ошибки из горутин
	var firstError error
	for i := 0; i < totalChunks; i++ {
		if err := <-resultChan; err != nil && firstError == nil {
			firstError = err
		}
	}

	if firstError != nil {
		logger.Log.Error("Failed to delete some URLs", zap.Error(firstError))
	}

	w.WriteHeader(http.StatusAccepted)
}
