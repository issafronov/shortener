package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Тестирует, что мидлвар корректно сжимает ответ, если клиент поддерживает gzip
func TestGzipMiddleware_CompressesResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(handler).ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.Header.Get("Content-Encoding") != "gzip" {
		t.Errorf("expected gzip content-encoding, got %q", res.Header.Get("Content-Encoding"))
	}

	// Распаковываем тело
	gr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gzipped body: %v", err)
	}

	if string(body) != "Hello, world!" {
		t.Errorf("unexpected response body: got %q", string(body))
	}
}

// Тестирует, что мидлвар не сжимает ответ, если клиент не поддерживает gzip
func TestGzipMiddleware_SkipsCompression(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, plain world!"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil) // без Accept-Encoding
	rr := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if encoding := res.Header.Get("Content-Encoding"); encoding != "" {
		t.Errorf("expected no content-encoding, got %q", encoding)
	}

	body, _ := io.ReadAll(res.Body)
	if string(body) != "Hello, plain world!" {
		t.Errorf("unexpected response body: got %q", string(body))
	}
}

// Тестирует, что мидлвар корректно распаковывает входящий gzip-запрос
func TestGzipMiddleware_DecompressesRequest(t *testing.T) {
	var received string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		received = string(data)
	})

	// Сжимаем тело запроса
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("compressed input"))
	_ = zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	GzipMiddleware(handler).ServeHTTP(rr, req)

	if received != "compressed input" {
		t.Errorf("unexpected request body: got %q", received)
	}
}

// Проверка: если входящий gzip невалиден, сервер возвращает 500
func TestGzipMiddleware_BadGzipInput(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called on gzip error")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not really gzipped"))
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()

	GzipMiddleware(handler).ServeHTTP(rr, req)

	res := rr.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", res.StatusCode)
	}
}
