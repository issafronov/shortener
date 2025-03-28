package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/middleware/compress"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"net/http"
	"os"
)

func main() {
	conf := config.LoadConfig()

	if err := restoreStorage(conf); err != nil {
		panic(err)
	}
	if err := runServer(conf); err != nil {
		panic(err)
	}
}

func Router(config *config.Config) chi.Router {
	router := chi.NewRouter()

	if err := logger.Initialize(config.LoggerLevel); err != nil {
		panic(err)
	}

	handler, err := handlers.NewHandler(config)

	if err != nil {
		logger.Log.Info("Failed to initialize handler")
	}

	router.Use(logger.RequestLogger)
	router.Use(compress.GzipMiddleware)
	router.Get("/{key}", handler.GetLinkHandle)
	router.Post("/", handler.CreateLinkHandle)
	router.Post("/api/shorten", handler.CreateJSONLinkHandle)
	return router
}

func runServer(config *config.Config) error {
	fmt.Println("Running server on", config.ServerAddress)
	return http.ListenAndServe(config.ServerAddress, Router(config))
}

func restoreStorage(config *config.Config) error {
	fmt.Println("Restoring storage")

	file, err := os.OpenFile(config.FileStoragePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := scanner.Bytes()
		shortenerURL := storage.ShortenerURL{}
		err = json.Unmarshal(data, &shortenerURL)
		if err != nil {
			return err
		}
		storage.Urls[shortenerURL.ShortURL] = shortenerURL.OriginalURL
	}
	return file.Close()
}
