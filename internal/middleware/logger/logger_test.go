package logger

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer

	encoderCfg := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zap.DebugLevel)
	Log = zap.New(core)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusTeapot) // 418
		w.Write([]byte("short"))
	})

	reqLogger := RequestLogger(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-url", nil)
	rr := httptest.NewRecorder()

	reqLogger.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal("Handler was not called")
	}

	if rr.Code != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	if string(body) != "short" {
		t.Errorf("Expected body 'short', got '%s'", string(body))
	}

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte(`"status":418`)) {
		t.Errorf("Expected log to contain status 418, got log: %s", logOutput)
	}
	if !bytes.Contains([]byte(logOutput), []byte(`"uri":"/test-url"`)) {
		t.Errorf("Expected log to contain uri /test-url, got log: %s", logOutput)
	}
}

func TestInitialize(t *testing.T) {
	// Проверка корректного уровня
	err := Initialize("debug")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Проверка ошибки при неверном уровне
	err = Initialize("invalid-level")
	if err == nil {
		t.Fatal("Expected error for invalid level, got nil")
	}
}

type dummyResponseWriter struct {
	headerWritten bool
	statusCode    int
	body          bytes.Buffer
}

func (d *dummyResponseWriter) Header() http.Header {
	return http.Header{}
}

func (d *dummyResponseWriter) Write(b []byte) (int, error) {
	return d.body.Write(b)
}

func (d *dummyResponseWriter) WriteHeader(statusCode int) {
	d.headerWritten = true
	d.statusCode = statusCode
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	drw := &dummyResponseWriter{}
	rd := &responseData{}
	lrw := &loggingResponseWriter{
		ResponseWriter: drw,
		responseData:   rd,
	}

	data := []byte("hello")

	n, err := lrw.Write(data)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}

	if lrw.responseData.size != len(data) {
		t.Errorf("responseData.size = %d, want %d", lrw.responseData.size, len(data))
	}

	if drw.body.String() != "hello" {
		t.Errorf("Underlying ResponseWriter body = %q, want %q", drw.body.String(), "hello")
	}
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	drw := &dummyResponseWriter{}
	rd := &responseData{}
	lrw := &loggingResponseWriter{
		ResponseWriter: drw,
		responseData:   rd,
	}

	status := 418
	lrw.WriteHeader(status)

	if !drw.headerWritten {
		t.Errorf("Underlying ResponseWriter WriteHeader not called")
	}
	if drw.statusCode != status {
		t.Errorf("Underlying ResponseWriter statusCode = %d, want %d", drw.statusCode, status)
	}
	if lrw.responseData.status != status {
		t.Errorf("responseData.status = %d, want %d", lrw.responseData.status, status)
	}
}
