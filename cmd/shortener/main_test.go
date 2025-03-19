package main

import (
	"fmt"
	"github.com/issafronov/shortener/internal/app/config"
	"github.com/issafronov/shortener/internal/app/handlers"
	"github.com/issafronov/shortener/internal/app/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func init() {
	storage.Urls = map[string]string{}
}

func TestCreateLinkHandle(t *testing.T) {
	conf := &config.Config{}
	h := handlers.NewHandler(conf)
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
	h := handlers.NewHandler(conf)
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
