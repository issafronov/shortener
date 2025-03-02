package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/issafronov/shortener/internal/app/handlers"
	"net/http"
)

func main() {
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
	return http.ListenAndServe(`localhost:8080`, Router())
}
