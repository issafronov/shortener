package main

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func init() {
	storage.Urls = map[string]string{}
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
