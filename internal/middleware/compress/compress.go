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
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(w)
	return &compressWriter{
		w:  w,
		zw: zw,
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if !c.wroteHeader {
		if statusCode >= 200 && statusCode < 300 {
			c.w.Header().Set("Content-Encoding", "gzip")
			c.w.Header().Del("Content-Length")
		}
		c.w.WriteHeader(statusCode)
		c.wroteHeader = true
	}
}

func (c *compressWriter) Close() error {
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

func (c *compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newCompressWriter(w)
			defer cw.Close()
			next.ServeHTTP(cw, r)
			return
		}

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to read gzip body", http.StatusInternalServerError)
				fmt.Println("Error creating gzip reader:", err)
				return
			}
			defer cr.Close()
			r.Body = cr
		}

		next.ServeHTTP(w, r)
	})
}
