package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
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
	router.Get("/{key}", handler.MainPage)
	router.Post("/", handler.MainPage)
	return router
}

func runServer(config *config.Config) error {
	fmt.Println("Starting server...", config)
	fmt.Println("Running server on", config.ServerAddress)
	return http.ListenAndServe(config.ServerAddress, Router(config))
}
