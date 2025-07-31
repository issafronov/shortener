package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/grpcserver"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/service"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/middleware/auth"
	"github.com/issafronov/shortener/internal/middleware/compress"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"github.com/issafronov/shortener/internal/middleware/trustedsubnet"
	"github.com/issafronov/shortener/internal/pprof"
	"github.com/issafronov/shortener/proto"
	_ "github.com/jackc/pgx/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	pprof.Start()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	conf := config.LoadConfig()

	if err := restoreStorage(conf); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	if err := runServer(conf, ctx, &wg); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	wg.Wait()

	fmt.Println("Server shutdown completed")
}

// Router возвращает настроенный маршрутизатор chi с подключёнными middleware и обработчиками
func Router(config *config.Config, s service.Service) chi.Router {
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

	router.Group(func(r chi.Router) {
		subnet := parseSubnet(config.TrustedSubnet)
		r.Use(trustedsubnet.TrustedSubnetMiddleware(subnet))
		r.Get("/api/internal/stats", handler.InternalStats)
	})

	return router
}

func runServer(cfg *config.Config, parentCtx context.Context, wg *sync.WaitGroup) error {
	fmt.Println("Running server on", cfg.ServerAddress)

	serverCtx, stop := context.WithCancel(parentCtx)
	defer stop()

	var st storage.Storage
	var srv service.Service
	var err error

	if cfg.DatabaseDSN != "" {
		pgStorage, err := storage.NewPostgresStorage(serverCtx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		st = pgStorage
	} else {
		st, err = storage.NewFileStorage(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize file storage: %w", err)
		}
	}
	srv = service.NewService(st)
	router := Router(cfg, srv)
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startHTTPServer(cfg, server); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			stop()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runGRPCServer(cfg, srv, serverCtx); err != nil {
			fmt.Printf("gRPC server error: %v\n", err)
			stop()
		}
	}()

	<-serverCtx.Done()

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

func startHTTPServer(cfg *config.Config, server *http.Server) error {
	if cfg.EnableHTTPS {
		certFile := "cert.pem"
		keyFile := "key.pem"

		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file %s not found", certFile)
		}
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			return fmt.Errorf("key file %s not found", keyFile)
		}

		fmt.Println("Starting HTTPS server on", cfg.ServerAddress)
		return server.ListenAndServeTLS(certFile, keyFile)
	}

	fmt.Println("Starting HTTP server on", cfg.ServerAddress)
	return server.ListenAndServe()
}

// runGRPCServer запускает grpc
func runGRPCServer(cfg *config.Config, srv service.Service, ctx context.Context) error {
	lis, err := net.Listen("tcp", cfg.GRPCServerAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC: %w", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterShortenerServer(grpcServer, grpcserver.NewGRPCHandler(srv, cfg))
	reflection.Register(grpcServer)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	fmt.Printf("Starting gRPC server on %s\n", cfg.GRPCServerAddress)
	return grpcServer.Serve(lis)
}

func parseSubnet(cidr string) *net.IPNet {
	if cidr == "" {
		return nil
	}
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Printf("invalid trusted subnet: %v", err)
		return nil
	}
	return subnet
}
