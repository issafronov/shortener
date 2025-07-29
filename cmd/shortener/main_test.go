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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/service"
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

func createTempFile(t *testing.T, data []byte) string {
	tmpFile, err := os.CreateTemp("", "test_storage_*.json")
	require.NoError(t, err)

	if data != nil {
		_, err = tmpFile.Write(data)
		require.NoError(t, err)
	}

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func TestCreateLinkHandle(t *testing.T) {
	conf := &config.Config{}
	tmpFile := createTempFile(t, nil)
	defer os.Remove(tmpFile)

	conf.FileStoragePath = tmpFile
	s, err := storage.NewFileStorage(conf)
	require.NoError(t, err)
	svc := service.NewService(s)
	h, err := handlers.NewHandler(conf, svc)
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
			assert.NotEmpty(t, strings.TrimSpace(string(res.Body())))
		})
	}
}

func TestCreateJSONLinkHandle(t *testing.T) {
	conf := &config.Config{}
	tmpFile := createTempFile(t, nil)
	defer os.Remove(tmpFile)

	conf.FileStoragePath = tmpFile
	s, err := storage.NewFileStorage(conf)
	require.NoError(t, err)
	svc := service.NewService(s)
	h, err := handlers.NewHandler(conf, svc)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = testutils.WithTestUserContext(r, utils.CreateShortKey(10))
		h.CreateJSONLinkHandle(w, r)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	t.Run("method_post_success", func(t *testing.T) {
		res, err := resty.New().R().
			SetHeader("Content-Type", "application/json").
			SetBody(`{"url": "https://www.google.com"}`).
			Post(srv.URL)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, res.StatusCode())
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
	})
}

func TestGzipCompression(t *testing.T) {
	conf := &config.Config{}
	tmpFile := createTempFile(t, nil)
	defer os.Remove(tmpFile)

	conf.FileStoragePath = tmpFile
	s, err := storage.NewFileStorage(conf)
	require.NoError(t, err)
	svc := service.NewService(s)
	h, err := handlers.NewHandler(conf, svc)
	require.NoError(t, err)

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
		require.NoError(t, zb.Close())

		req, err := http.NewRequest("POST", srv.URL, buf)
		require.NoError(t, err)

		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)

		req, err := http.NewRequest("POST", srv.URL, buf)
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)

		_, err = io.ReadAll(zr)
		require.NoError(t, err)
	})
}

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

	printBuildInfo()
}

func Test_Router(t *testing.T) {
	tmpFile := createTempFile(t, nil)
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		LoggerLevel:     "info",
		FileStoragePath: tmpFile,
	}

	s, err := storage.NewFileStorage(cfg)
	require.NoError(t, err)
	svc := service.NewService(s)

	router := Router(cfg, svc)
	assert.NotNil(t, router)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func Test_runServer_FileStorage(t *testing.T) {
	tmpFile := createTempFile(t, nil)
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		FileStoragePath: tmpFile,
		ServerAddress:   "127.0.0.1:8085",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	go func() {
		err := runServer(cfg, ctx, &wg)
		assert.NoError(t, err)
	}()

	time.Sleep(500 * time.Millisecond)

	resp, err := http.Get("http://" + cfg.ServerAddress + "/ping")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	cancel()
	wg.Wait()
}

func Test_runServer_InvalidDB(t *testing.T) {
	cfg := &config.Config{
		DatabaseDSN:   "invalid-dsn",
		ServerAddress: "127.0.0.1:0",
	}
	ctx := context.Background()
	var wg sync.WaitGroup

	err := runServer(cfg, ctx, &wg)
	assert.Error(t, err)
}
