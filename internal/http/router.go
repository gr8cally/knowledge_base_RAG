package http

import (
	"context"
	"log/slog"
	"net/http"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/http/handlers"
	"knowledge_base_RAG/internal/http/middleware"
)

func NewRouter(logger *slog.Logger, sqlitePath string, chromaPing func(context.Context) error, kbService *app.KnowledgeBaseService) http.Handler {
	healthHandler := handlers.NewHealthHandler(sqlitePath, chromaPing)
	kbHandler := handlers.NewKBHandler(kbService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler.Health)
	mux.HandleFunc("GET /readyz", healthHandler.Ready)
	mux.HandleFunc("GET /", kbHandler.Index)
	mux.HandleFunc("GET /kbs/{kbID}", kbHandler.Detail)
	mux.HandleFunc("GET /api/kbs", kbHandler.ListAPI)
	mux.HandleFunc("POST /api/kbs", kbHandler.CreateAPI)
	mux.HandleFunc("PATCH /api/kbs/{kbID}", kbHandler.UpdateAPI)
	mux.HandleFunc("DELETE /api/kbs/{kbID}", kbHandler.ArchiveAPI)

	handler := middleware.Recover(logger)(mux)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}
