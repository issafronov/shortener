package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"net/http"
)

func main() {
	config.ParseFlags()

	if err := runServer(); err != nil {
		panic(err)
	}
}

func Router() chi.Router {
	router := chi.NewRouter()
	router.Get("/{key}", handlers.MainPage)
	router.Post("/", handlers.MainPage)
	return router
}

func runServer() error {
	fmt.Println("Running server on", config.FlagRunAddr)
	return http.ListenAndServe(config.FlagRunAddr, Router())
}
