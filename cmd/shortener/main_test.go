package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/contextkeys"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/app/utils"
	"github.com/issafronov/shortener/internal/middleware/compress"
	"github.com/issafronov/shortener/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	storage.Urls = map[string]storage.ShortenerURL{}
}

func TestCreateLinkHandle(t *testing.T) {
	conf := &config.Config{}
	conf.FileStoragePath = "testStorage.json"
	s, _ := storage.NewFileStorage(conf)
	h, err := handlers.NewHandler(conf, s)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = testutils.WithTestUserContext(r, utils.CreateShortKey(10))
		h.CreateLinkHandle(w, r)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	tests := []struct {
		name         string
		method       string
		body         string
		expectedCode int
		contentType  string
	}{
		{
			name:         "Generate short link (plain/text)",
			method:       http.MethodPost,
			body:         "https://www.google.com",
			expectedCode: http.StatusCreated,
			contentType:  "text/plain",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := resty.New()

			res, err := client.R().
				SetHeader("Content-Type", "text/plain").
				SetBody(tc.body).
				Post(srv.URL)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedCode, res.StatusCode())
			assert.Equal(t, tc.contentType, res.Header().Get("Content-Type"))
			assert.NotEmpty(t, strings.TrimSpace(string(res.Body())), "Expected non-empty short URL")
		})
	}
}

func TestCreateJSONLinkHandle(t *testing.T) {
	conf := &config.Config{}
	conf.FileStoragePath = "testStorage.json"
	s, _ := storage.NewFileStorage(conf)
	h, err := handlers.NewHandler(conf, s)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = testutils.WithTestUserContext(r, utils.CreateShortKey(10))
		h.CreateJSONLinkHandle(w, r)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	testCases := []struct {
		name         string
		method       string
		body         string
		expectedCode int
		contentType  string
	}{
		{
			name:         "method_post_success",
			method:       http.MethodPost,
			body:         `{"url": "https://www.google.com"}`,
			expectedCode: http.StatusCreated,
			contentType:  "application/json",
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = test.method
			req.URL = srv.URL

			if len(test.body) > 0 {
				req.SetHeader("Content-Type", "application/json")
				req.SetBody(test.body)
			}

			res, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")
			assert.Equal(t, test.expectedCode, res.StatusCode(), "Response code didn't match expected")
			assert.Equal(t, test.contentType, res.Header().Get("Content-Type"))
		})
	}
}

func TestGzipCompression(t *testing.T) {
	conf := &config.Config{}
	conf.FileStoragePath = "testStorage.json"
	s, _ := storage.NewFileStorage(conf)
	h, err := handlers.NewHandler(conf, s)
	if err != nil {
		t.Fatal(err)
	}
	handler := compress.GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = testutils.WithTestUserContext(r, utils.CreateShortKey(10))
		h.CreateJSONLinkHandle(w, r)
	}))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	requestBody := `{"url": "https://www.google.com"}`

	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)
		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)
		err = zb.Close()
		require.NoError(t, err)

		r := httptest.NewRequest("POST", srv.URL, buf)
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, "testUserID")
		r = r.WithContext(ctx)
		r.RequestURI = ""
		r.Header.Set("Content-Encoding", "gzip")
		r.Header.Set("Accept-Encoding", "")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		_, err = io.ReadAll(resp.Body)
		defer resp.Body.Close()

		require.NoError(t, err)
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)
		r := httptest.NewRequest("POST", srv.URL, buf)
		ctx := r.Context()
		ctx = context.WithValue(ctx, contextkeys.UserIDKey, "testUserID")
		r = r.WithContext(ctx)
		r.RequestURI = ""
		r.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		defer resp.Body.Close()

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		_, err = io.ReadAll(zr)
		require.NoError(t, err)
	})
}

// helper для создания временного файла с данными
func createTestStorageFile(t *testing.T, data []storage.ShortenerURL) string {
	tmpFile, err := os.CreateTemp("", "test_storage_*.json")
	require.NoError(t, err)
	for _, d := range data {
		line, _ := json.Marshal(d)
		tmpFile.Write(append(line, '\n'))
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func Test_restoreStorage(t *testing.T) {
	// подготовка: запишем пару ссылок в файл
	testData := []storage.ShortenerURL{
		{ShortURL: "abc123", OriginalURL: "https://example.com"},
		{ShortURL: "xyz789", OriginalURL: "https://test.com"},
	}
	path := createTestStorageFile(t, testData)
	defer os.Remove(path)

	cfg := &config.Config{FileStoragePath: path}
	storage.Urls = make(map[string]storage.ShortenerURL)

	err := restoreStorage(cfg)
	require.NoError(t, err)
	assert.Len(t, storage.Urls, 2)
	assert.Equal(t, "https://example.com", storage.Urls["abc123"].OriginalURL)
}

func Test_getOrNA(t *testing.T) {
	assert.Equal(t, "N/A", getOrNA(""))
	assert.Equal(t, "value", getOrNA("value"))
}

func Test_printBuildInfo(t *testing.T) {
	buildVersion = "v1.0.0"
	buildDate = "2025-01-01"
	buildCommit = "abcdef"

	// Просто убедимся, что функция не падает
	printBuildInfo()
}

func Test_Router(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "storage-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	cfg := &config.Config{
		LoggerLevel:     "info",
		FileStoragePath: tmpFile.Name(),
	}

	s, err := storage.NewFileStorage(cfg)
	require.NoError(t, err)

	router := Router(cfg, s)
	assert.NotNil(t, router)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func Test_runServer_FileStorage(t *testing.T) {
	cfg := &config.Config{
		FileStoragePath: "test_runserver.json",
		ServerAddress:   "127.0.0.1:9999",
	}
	defer os.Remove(cfg.FileStoragePath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	serverErr := make(chan error, 1)

	go func() {
		err := runServer(cfg, ctx, serverErr)
		assert.NoError(t, err)
	}()

	time.Sleep(500 * time.Millisecond) // ждем сервер
	resp, err := http.Get("http://" + cfg.ServerAddress + "/ping")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func Test_runServer_InvalidDB(t *testing.T) {
	cfg := &config.Config{
		DatabaseDSN:   "invalid-dsn",
		ServerAddress: "127.0.0.1:0", // any available port
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	serverErr := make(chan error, 1)
	err := runServer(cfg, ctx, serverErr)
	assert.Error(t, err)
}
