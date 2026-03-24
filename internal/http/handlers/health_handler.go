package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	sqlitestore "knowledge_base_RAG/internal/storage/sqlite"
)

type HealthHandler struct {
	sqlitePath string
	chromaPing func(context.Context) error
}

func NewHealthHandler(sqlitePath string, chromaPing func(context.Context) error) *HealthHandler {
	return &HealthHandler{
		sqlitePath: sqlitePath,
		chromaPing: chromaPing,
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	sqliteCtx, sqliteCancel := context.WithTimeout(context.Background(), time.Second)
	defer sqliteCancel()

	if err := sqlitestore.Ping(sqliteCtx, h.sqlitePath); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  err.Error(),
		})
		return
	}

	chromaCtx, chromaCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer chromaCancel()

	if err := h.chromaPing(chromaCtx); err != nil {
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
