package handlers

import (
	"fmt"
	"net/http"

	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
)

// Ping - handler для проверки работоспособности сервиса
func (h *Handler) Ping(res http.ResponseWriter, req *http.Request) {
	logger.Log.Info("PingHandle", zap.String("url", req.URL.String()))
	if err := h.storage.Ping(req.Context()); err != nil {
		res.WriteHeader(http.StatusServiceUnavailable)
		fmt.Println(err.Error())
	}
	res.WriteHeader(http.StatusOK)
}
