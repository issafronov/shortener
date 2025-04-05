package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/issafronov/shortener/internal/middleware/compress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func init() {
	storage.Urls = map[string]string{}
}

func TestCreateLinkHandle(t *testing.T) {
	conf := &config.Config{}
	conf.FileStoragePath = "testStorage.json"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, _ := storage.NewPostgresStorage(ctx, conf.DatabaseDSN)
	h, err := handlers.NewHandler(conf, s)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(h.CreateLinkHandle)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	type want struct {
		code        int
		contentType string
	}

	var key string

	tests := []struct {
		name   string
		method string
		want   want
	}{
		{
			name:   "Generate link",
			method: http.MethodPost,
			want: want{
				code:        201,
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := resty.New().R()
			req.Method = test.method
			if test.method == http.MethodPost {
				req.Body = strings.NewReader("https://www.google.com")
			}

			req.URL = srv.URL

			if test.method == http.MethodGet {
				req.URL = key
			}

			fmt.Println(req.URL)
			res, err := req.Send()
			require.NoError(t, err, "error making HTTP request")
			assert.Equal(t, test.want.code, res.StatusCode())
			key = string(res.Body())
			fmt.Println("after test ", key)
			assert.Equal(t, test.want.contentType, res.Header().Get("Content-Type"))
		})
	}
}

func TestCreateJSONLinkHandle(t *testing.T) {
	conf := &config.Config{}
	conf.FileStoragePath = "testStorage.json"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, _ := storage.NewPostgresStorage(ctx, conf.DatabaseDSN)
	h, err := handlers.NewHandler(conf, s)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(h.CreateJSONLinkHandle)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, _ := storage.NewPostgresStorage(ctx, conf.DatabaseDSN)
	h, err := handlers.NewHandler(conf, s)
	if err != nil {
		t.Fatal(err)
	}
	handler := compress.GzipMiddleware(http.HandlerFunc(h.CreateJSONLinkHandle))
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
