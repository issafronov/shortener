package handlers

import (
	"github.com/issafronov/shortener/internal/app/utils"
	"io"
	"net/http"
)

const (
	shortKeyLength = 8
)

var urls = make(map[string]string)

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
		urls[shortKey] = originalURL
		res.WriteHeader(http.StatusCreated)
		res.Header().Set("content-type", "text/plain")
		_, err = res.Write([]byte("http://localhost:8080/" + shortKey))

		if err != nil {
			panic(err)
		}
		return
	}

	if req.Method != http.MethodGet {
		http.Error(res, "Only GET and POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	link, ok := urls[req.URL.Path[1:]]

	if !ok {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	http.Redirect(res, req, link, http.StatusTemporaryRedirect)
}
