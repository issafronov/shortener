package pprof_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/issafronov/shortener/internal/pprof"
)

func TestStart(t *testing.T) {
	pprof.Start()

	// Подождём немного, чтобы сервер успел подняться
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:6060/debug/pprof/")
	if err != nil {
		t.Fatalf("Failed to GET pprof endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Response body is empty")
	}
}
