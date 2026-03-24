package http

import (
	"context"
	"log/slog"
	"net/http"

	"knowledge_base_RAG/internal/http/handlers"
	"knowledge_base_RAG/internal/http/middleware"
)

func NewRouter(logger *slog.Logger, sqlitePath string, chromaPing func(context.Context) error) http.Handler {
	healthHandler := handlers.NewHealthHandler(sqlitePath, chromaPing)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler.Health)
	mux.HandleFunc("GET /readyz", healthHandler.Ready)

	handler := middleware.Recover(logger)(mux)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}
