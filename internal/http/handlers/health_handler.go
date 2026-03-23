package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

type HealthHandler struct {
	sqlitePath string
}

func NewHealthHandler(sqlitePath string) *HealthHandler {
	return &HealthHandler{sqlitePath: sqlitePath}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	if err := os.MkdirAll(filepath.Dir(h.sqlitePath), 0o755); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  err.Error(),
		})
		return
	}

	f, err := os.OpenFile(h.sqlitePath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  err.Error(),
		})
		return
	}
	_ = f.Close()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
