package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	sqlitestore "github.com/gr8cally/knowledge_base_RAG/internal/storage/sqlite"
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqlitestore.Ping(ctx, h.sqlitePath); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
