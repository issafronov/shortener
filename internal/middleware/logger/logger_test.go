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
