package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/models"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/app/utils"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
)

const (
	shortKeyLength = 8
)

type Handler struct {
	config *config.Config
	file   *os.File
	writer *bufio.Writer
	reader *bufio.Reader
}

type ShortenerURL struct {
	UUID        int    `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewHandler(config *config.Config) (*Handler, error) {
	file, err := os.OpenFile(config.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &Handler{
		config: config,
		file:   file,
		writer: bufio.NewWriter(file),
		reader: bufio.NewReader(file),
	}, nil
}

func (h *Handler) WriteURL(url ShortenerURL) error {
	logger.Log.Info("Writing URL", zap.String("url", url.ShortURL))
	data, err := json.Marshal(url)

	if err != nil {
		logger.Log.Info("Failed to marshal shortener URL", zap.Error(err))
		return err
	}

	if _, err := h.writer.Write(data); err != nil {
		logger.Log.Info("Failed to write URL", zap.String("url", url.ShortURL), zap.Error(err))
		return err
	}

	if err := h.writer.WriteByte('\n'); err != nil {
		logger.Log.Info("Error writing data new line", zap.Error(err))
		return err
	}

	return h.writer.Flush()
}

func (h *Handler) GetUUID() int {
	counter := 0
	for i := 1; ; i++ {
		line, err := h.reader.ReadBytes('\n')
		fmt.Printf("[line:%d pos:%d] %q\n", i, counter, line)
		if err != nil {
			break
		}
		counter += len(line)
	}

	return counter + 1
}

func (h *Handler) Close() error {
	return h.file.Close()
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

	uuid := h.GetUUID()

	shortenerURL := &ShortenerURL{
		UUID:        uuid,
		ShortURL:    shortKey,
		OriginalURL: originalURL,
	}

	if err := h.WriteURL(*shortenerURL); err != nil {
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
	uuid := h.GetUUID()

	shortenerURL := &ShortenerURL{
		UUID:        uuid,
		ShortURL:    shortKey,
		OriginalURL: originalURL,
	}

	if err := h.WriteURL(*shortenerURL); err != nil {
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
