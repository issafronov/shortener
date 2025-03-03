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

func TestMainPage(t *testing.T) {
	conf := config.LoadConfig()
	h := handlers.NewHandler(conf)
	handler := http.HandlerFunc(h.MainPage)
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
				contentType: "text/plain; charset=utf-8",
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
