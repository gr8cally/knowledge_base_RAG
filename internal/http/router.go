package http

import (
	"log/slog"
	"net/http"

	"github.com/gr8cally/knowledge_base_RAG/internal/http/handlers"
	"github.com/gr8cally/knowledge_base_RAG/internal/http/middleware"
)

func NewRouter(logger *slog.Logger, sqlitePath string) http.Handler {
	healthHandler := handlers.NewHealthHandler(sqlitePath)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler.Health)
	mux.HandleFunc("GET /readyz", healthHandler.Ready)

	handler := middleware.Recover(logger)(mux)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}
