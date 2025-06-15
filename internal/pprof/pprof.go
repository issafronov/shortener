package pprof

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

// Start запускает pprof-сервер
func Start() {
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Println("pprof server error:", err)
		}
	}()
}
