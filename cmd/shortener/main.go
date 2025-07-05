package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/middleware/auth"
	"github.com/issafronov/shortener/internal/middleware/compress"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"github.com/issafronov/shortener/internal/pprof"
	_ "github.com/jackc/pgx/stdlib"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	pprof.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conf := config.LoadConfig()

	if err := restoreStorage(conf); err != nil {
		panic(err)
	}
	if err := runServer(conf, ctx); err != nil {
		panic(err)
	}
}

// Router возвращает настроенный маршрутизатор chi с подключёнными middleware и обработчиками
func Router(config *config.Config, s storage.Storage) chi.Router {
	router := chi.NewRouter()

	if err := logger.Initialize(config.LoggerLevel); err != nil {
		panic(err)
	}

	handler, err := handlers.NewHandler(config, s)

	if err != nil {
		logger.Log.Info("Failed to initialize handler")
	}

	router.Use(logger.RequestLogger)
	router.Use(compress.GzipMiddleware)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(auth.AuthorizationMiddleware)
	router.Get("/{key}", handler.GetLinkHandle)
	router.Post("/", handler.CreateLinkHandle)
	router.Post("/api/shorten", handler.CreateJSONLinkHandle)
	router.Post("/api/shorten/batch", handler.CreateBatchJSONLinkHandle)
	router.Get("/ping", handler.Ping)
	router.Get("/api/user/urls", handler.GetUserLinksHandle)
	router.Delete("/api/user/urls", handler.DeleteLinksHandle)
	return router
}

func runServer(cfg *config.Config, ctx context.Context) error {
	fmt.Println("Running server on", cfg.ServerAddress)
	var s storage.Storage
	var err error
	if cfg.DatabaseDSN != "" {
		pgStorage, err := storage.NewPostgresStorage(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		s = pgStorage
	} else {
		s, err = storage.NewFileStorage(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize file storage: %w", err)
		}
	}
	return http.ListenAndServe(cfg.ServerAddress, Router(cfg, s))
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
		storage.Urls[shortenerURL.ShortURL] = shortenerURL
	}
	return file.Close()
}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", getOrNA(buildVersion))
	fmt.Printf("Build date: %s\n", getOrNA(buildDate))
	fmt.Printf("Build commit: %s\n", getOrNA(buildCommit))
}

func getOrNA(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}
