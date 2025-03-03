package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"net/http"
	"os"
)

func main() {
	// первый аргумент — имя запущенного файла
	fmt.Printf("Command: %v\n", os.Args[0])
	// выведем остальные параметры
	for i, v := range os.Args[1:] {
		fmt.Println(i+1, v)
	}

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
	fmt.Println("Running server on", config.Address)
	return http.ListenAndServe(config.Address, Router(config))
}
