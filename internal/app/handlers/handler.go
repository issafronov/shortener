package handlers

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/app/utils"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
)

const (
	shortKeyLength = 8
)

type Handler struct {
	config *config.Config
}

func NewHandler(config *config.Config) *Handler {
	return &Handler{config: config}
}

func (h *Handler) CreateLinkHandle(res http.ResponseWriter, req *http.Request) {
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
	storage.Urls[shortKey] = originalURL
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

func (h *Handler) GetLinkHandle(res http.ResponseWriter, req *http.Request) {
	logger.Log.Info("GetLinkHandle", zap.String("url", req.URL.String()))
	key := chi.URLParam(req, "key")
	link, ok := storage.Urls[key]

	if !ok {
		logger.Log.Info("Link not found", zap.String("key", key))
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	logger.Log.Info("Link found", zap.String("key", key))
	http.Redirect(res, req, link, http.StatusTemporaryRedirect)
}

func (h *Handler) CreateJSONLinkHandle(res http.ResponseWriter, req *http.Request) {
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
	storage.Urls[shortKey] = originalURL
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
