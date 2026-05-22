package health

import (
	"encoding/json"
	"net/http"
	"time"
)

var startTime = time.Now()

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"version":        "1.0.0",
		"uptime_seconds": int64(time.Since(startTime).Seconds()),
	})
}
