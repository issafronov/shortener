package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conf := config.LoadConfig()

	if err := restoreStorage(conf); err != nil {
		panic(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	serverErr := make(chan error, 1)
	go func() {
		if err := runServer(conf, ctx, serverErr); err != nil {
			serverErr <- err
		}
	}()

	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v\n", sig)
		cancel()

		_, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		fmt.Println("Shutting down server gracefully...")

	case err := <-serverErr:
		fmt.Printf("Server error: %v\n", err)
		cancel()
	}

	fmt.Println("Server shutdown completed")
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

func runServer(cfg *config.Config, ctx context.Context, serverErr chan<- error) error {
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
	router := Router(cfg, s)
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	go func() {
		var err error
		if cfg.EnableHTTPS {
			certFile := "cert.pem"
			keyFile := "key.pem"

			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				serverErr <- fmt.Errorf("certificate file %s not found", certFile)
				return
			}
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				serverErr <- fmt.Errorf("key file %s not found", keyFile)
				return
			}

			fmt.Println("Starting HTTPS server on", cfg.ServerAddress)
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			fmt.Println("Starting HTTP server on", cfg.ServerAddress)
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return nil
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
