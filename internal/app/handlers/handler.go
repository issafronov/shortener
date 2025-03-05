package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/app/utils"
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
		panic(err)
	}

	originalURL := string(body)

	if originalURL == "" {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	
	shortKey := utils.CreateShortKey(shortKeyLength)
	storage.Urls[shortKey] = originalURL
	res.WriteHeader(http.StatusCreated)
	res.Header().Set("content-type", "text/plain")
	resultHostAddr := "http://" + req.Host

	if h.config.BaseURL != "" {
		resultHostAddr = h.config.BaseURL
	}

	_, err = res.Write([]byte(resultHostAddr + "/" + shortKey))

	if err != nil {
		panic(err)
	}
}

func (h *Handler) GetLinkHandle(res http.ResponseWriter, req *http.Request) {
	key := chi.URLParam(req, "key")
	link, ok := storage.Urls[key]

	if !ok {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	http.Redirect(res, req, link, http.StatusTemporaryRedirect)
}
