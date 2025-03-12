package main

import (
	"io"
	"math/rand"
	"net/http"
)

var urls = make(map[string]string)

func main() {
	if err := runServer(); err != nil {
		panic(err)
	}
}

func runServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, mainPage)
	return http.ListenAndServe(`localhost:8080`, mux)
}

func mainPage(res http.ResponseWriter, req *http.Request) {
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

		shortKey := createShortKey()
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

func createShortKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const keyLength = 6

	shortKey := make([]byte, keyLength)
	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortKey)
}
