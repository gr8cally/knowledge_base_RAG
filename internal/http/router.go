package http

import (
	"context"
	"log/slog"
	"net/http"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/http/handlers"
	"knowledge_base_RAG/internal/http/middleware"
	"knowledge_base_RAG/internal/ingest"
)

func NewRouter(logger *slog.Logger, sqlitePath string, chromaPing func(context.Context) error, kbService *app.KnowledgeBaseService, documentService *ingest.Service, conversationService *app.ConversationService, chatService *app.ChatService, maxUploadMB int) http.Handler {
	healthHandler := handlers.NewHealthHandler(sqlitePath, chromaPing)
	kbHandler := handlers.NewKBHandler(kbService)
	workspaceHandler := handlers.NewWorkspaceHandler(kbService, conversationService, documentService, maxUploadMB)
	documentHandler := handlers.NewDocumentHandler(documentService, maxUploadMB)
	ingestHandler := handlers.NewIngestHandler(documentService)
	conversationHandler := handlers.NewConversationHandler(conversationService)
	chatHandler := handlers.NewChatHandler(chatService, kbService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler.Health)
	mux.HandleFunc("GET /readyz", healthHandler.Ready)
	mux.HandleFunc("GET /", workspaceHandler.Page)
	mux.HandleFunc("POST /workspace/kbs", workspaceHandler.CreateKnowledgeBase)
	mux.HandleFunc("POST /workspace/conversations", workspaceHandler.CreateConversation)
	mux.HandleFunc("GET /workspace/sources", workspaceHandler.SourcesPanel)
	mux.HandleFunc("POST /workspace/sources/upload", workspaceHandler.UploadDocument)
	mux.HandleFunc("POST /workspace/sources/{documentID}/refresh", workspaceHandler.RefreshDocument)
	mux.HandleFunc("DELETE /workspace/sources/{documentID}", workspaceHandler.DeleteDocument)
	mux.HandleFunc("POST /workspace/sources/reindex", workspaceHandler.ReindexAll)
	mux.HandleFunc("GET /kbs/{kbID}", kbHandler.Detail)
	mux.HandleFunc("GET /kbs/{kbID}/conversations/{conversationID}", chatHandler.Page)
	mux.HandleFunc("GET /api/kbs", kbHandler.ListAPI)
	mux.HandleFunc("POST /api/kbs", kbHandler.CreateAPI)
	mux.HandleFunc("PATCH /api/kbs/{kbID}", kbHandler.UpdateAPI)
	mux.HandleFunc("DELETE /api/kbs/{kbID}", kbHandler.ArchiveAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/documents", documentHandler.ListAPI)
	mux.HandleFunc("POST /api/kbs/{kbID}/documents/upload", documentHandler.UploadAPI)
	mux.HandleFunc("POST /api/kbs/{kbID}/documents/{documentID}/refresh", documentHandler.RefreshAPI)
	mux.HandleFunc("DELETE /api/kbs/{kbID}/documents/{documentID}", documentHandler.DeleteAPI)
	mux.HandleFunc("POST /api/kbs/{kbID}/reindex-all", documentHandler.ReindexAllAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/conversations", conversationHandler.ListAPI)
	mux.HandleFunc("POST /api/kbs/{kbID}/conversations", conversationHandler.CreateAPI)
	mux.HandleFunc("PATCH /api/kbs/{kbID}/conversations/{conversationID}", conversationHandler.UpdateAPI)
	mux.HandleFunc("DELETE /api/kbs/{kbID}/conversations/{conversationID}", conversationHandler.ArchiveAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/conversations/{conversationID}/messages", chatHandler.MessagesAPI)
	mux.HandleFunc("POST /api/kbs/{kbID}/conversations/{conversationID}/messages", chatHandler.PostMessageAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/conversations/{conversationID}/stream", chatHandler.StreamAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/ingestion-jobs", ingestHandler.ListAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/ingestion-jobs/{jobID}", ingestHandler.GetAPI)
	mux.HandleFunc("GET /api/kbs/{kbID}/ingestion-jobs/{jobID}/events", ingestHandler.Events)

	handler := middleware.Recover(logger)(mux)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.RequestID(handler)

	return handler
}
