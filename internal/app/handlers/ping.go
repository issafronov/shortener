package handlers

import (
	"fmt"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
	"net/http"
)

func (h *Handler) Ping(res http.ResponseWriter, req *http.Request) {
	logger.Log.Info("PingHandle", zap.String("url", req.URL.String()))
	if err := h.db.Ping(); err != nil {
		res.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(res, err.Error())
		fmt.Println(err.Error())
	}
	res.WriteHeader(http.StatusOK)
}
