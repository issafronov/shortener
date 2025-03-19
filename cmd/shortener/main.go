package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/logger"
	"net/http"
)

func main() {
	conf := config.LoadConfig()

	if err := runServer(conf); err != nil {
		panic(err)
	}
}

func Router(config *config.Config) chi.Router {
	router := chi.NewRouter()
	handler := handlers.NewHandler(config)

	if err := logger.Initialize(config.LoggerLevel); err != nil {
		panic(err)
	}

	router.Use(logger.RequestLogger)
	router.Get("/{key}", handler.GetLinkHandle)
	router.Post("/", handler.CreateLinkHandle)
	router.Post("/api/shorten", handler.CreateJSONLinkHandle)
	return router
}

func runServer(config *config.Config) error {
	fmt.Println("Running server on", config.ServerAddress)
	return http.ListenAndServe(config.ServerAddress, Router(config))
}
