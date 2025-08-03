package handlers

import (
	"net/http"

	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
)

// Ping - handler для проверки работоспособности сервиса.
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("PingHandle", zap.String("url", r.URL.String()))

	if err := h.service.Ping(r.Context()); err != nil {
		logger.Log.Error("service ping failed", zap.Error(err))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
