package main

import (
	"github.com/issafronov/shortener/internal/app/handlers"
	"net/http"
)

func main() {
	if err := runServer(); err != nil {
		panic(err)
	}
}

func runServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, handlers.MainPage)
	return http.ListenAndServe(`localhost:8080`, mux)
}
