package compress

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type compressWriter struct {
	w           http.ResponseWriter
	zw          *gzip.Writer
	wroteHeader bool
	compress    bool
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(w)
	return &compressWriter{
		w:        w,
		zw:       zw,
		compress: false,
	}
}

// Header возвращает заголовки ответа
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает тела ответа, при необходимости сжимает
func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.compress {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

// WriteHeader добавляет заголовок
func (c *compressWriter) WriteHeader(statusCode int) {
	if c.wroteHeader {
		return
	}
	// Сжимаем тело только для 2xx ответов
	if statusCode >= 200 && statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
		c.w.Header().Del("Content-Length") // размер меняется при сжатии
		c.compress = true
	} else {
		// Для редиректов (3xx) и ошибок не сжимаем
		c.compress = false
	}
	c.w.WriteHeader(statusCode)
	c.wroteHeader = true
}

// Close закрывает и возвращает в пул
func (c *compressWriter) Close() error {
	if !c.compress {
		// Если gzip не был включен, то просто ничего не делаем
		return nil
	}
	err := c.zw.Close()
	c.zw.Reset(io.Discard) // очистка, чтобы избежать утечек
	gzipWriterPool.Put(c.zw)
	return err
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &compressReader{r: r, zr: zr}, nil
}

// Read чтение данных
func (c *compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

// Close закрвает ресурсы
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// GzipMiddleware — middleware для обработки gzip-сжатия входящих и исходящих HTTP-сообщений
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newCompressWriter(w)
			ow = cw
			defer func() {
				if err := cw.Close(); err != nil {
					fmt.Println("gzip close error:", err)
				}
			}()
		}

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to read gzip body", http.StatusInternalServerError)
				fmt.Println("Error creating gzip reader:", err)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(ow, r)
	})
}
