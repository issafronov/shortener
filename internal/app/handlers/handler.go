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

func MainPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
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

		if config.FlagResultHostAddr != "" {
			resultHostAddr = config.FlagResultHostAddr
		}

		_, err = res.Write([]byte(resultHostAddr + "/" + shortKey))

		if err != nil {
			panic(err)
		}
		return
	}

	if req.Method != http.MethodGet {
		http.Error(res, "Only GET and POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	key := chi.URLParam(req, "key")
	link, ok := storage.Urls[key]

	if !ok {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	http.Redirect(res, req, link, http.StatusTemporaryRedirect)
}
