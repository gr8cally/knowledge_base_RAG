package handlers

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/domain"
	"knowledge_base_RAG/internal/ingest"
	"knowledge_base_RAG/internal/web/templates"
	components "knowledge_base_RAG/internal/web/templates/components"
)

type workspaceConversationService interface {
	List(ctx context.Context, kbID string) ([]domain.Conversation, error)
	Create(ctx context.Context, kbID, title string) (domain.Conversation, error)
}

type workspaceDocumentService interface {
	ListDocuments(ctx context.Context, kbID string) ([]domain.Document, error)
	UploadFile(ctx context.Context, kbID string, header *multipart.FileHeader) (ingest.UploadResult, error)
	RefreshDocument(ctx context.Context, kbID, documentID string) (ingest.UploadResult, error)
	DeleteDocument(ctx context.Context, kbID, documentID string) error
	ReindexAll(ctx context.Context, kbID string) (*domain.IngestionJob, error)
}

type WorkspaceHandler struct {
	kbService           *app.KnowledgeBaseService
	conversationService workspaceConversationService
	documentService     workspaceDocumentService
	maxUploadMB         int
}

func NewWorkspaceHandler(kbService *app.KnowledgeBaseService, conversationService workspaceConversationService, documentService workspaceDocumentService, maxUploadMB int) *WorkspaceHandler {
	return &WorkspaceHandler{
		kbService:           kbService,
		conversationService: conversationService,
		documentService:     documentService,
		maxUploadMB:         maxUploadMB,
	}
}

func (h *WorkspaceHandler) Page(w http.ResponseWriter, r *http.Request) {
	data, err := h.loadWorkspaceData(r.Context(), r.URL.Query().Get("kb"), r.URL.Query().Get("conversation"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "workspace_load_failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		if err := templates.WorkspaceShell(data).Render(r.Context(), w); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "render_workspace_shell_failed", err.Error())
		}
		return
	}

	if err := templates.WorkspacePage(data).Render(r.Context(), w); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_workspace_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_form", "invalid conversation form")
		return
	}

	kbID := r.FormValue("kb_id")
	if kbID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_kb", "kb_id is required")
		return
	}

	conversation, err := h.conversationService.Create(r.Context(), kbID, r.FormValue("title"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "create_conversation_failed", err.Error())
		return
	}

	pushURL := "/?kb=" + kbID + "&conversation=" + conversation.ID
	w.Header().Set("HX-Push-Url", pushURL)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data, err := h.loadWorkspaceData(r.Context(), kbID, conversation.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "workspace_load_failed", err.Error())
		return
	}
	if err := templates.WorkspaceShell(data).Render(r.Context(), w); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_workspace_shell_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) SourcesPanel(w http.ResponseWriter, r *http.Request) {
	kbID := r.URL.Query().Get("kb")
	if kbID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_kb", "kb query parameter is required")
		return
	}
	if err := h.renderSourcesPanel(r.Context(), w, kbID, ""); err != nil {
		if errors.Is(err, app.ErrKnowledgeBaseNotFound) {
			http.NotFound(w, r)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "render_sources_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(h.maxUploadMB)<<20)
	if err := r.ParseMultipartForm(int64(h.maxUploadMB) << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_multipart", err.Error())
		return
	}
	kbID := r.FormValue("kb_id")
	if kbID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_kb", "kb_id is required")
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeAPIError(w, http.StatusBadRequest, "missing_files", "no files uploaded")
		return
	}

	notice := "Files queued for ingestion."
	for _, header := range files {
		result, err := h.documentService.UploadFile(r.Context(), kbID, header)
		if err != nil {
			switch {
			case errors.Is(err, ingest.ErrKnowledgeBaseNotFound):
				http.NotFound(w, r)
			case errors.Is(err, ingest.ErrIngestionQueueFull):
				writeAPIError(w, http.StatusServiceUnavailable, "ingestion_queue_full", err.Error())
			default:
				writeAPIError(w, http.StatusBadRequest, "upload_failed", err.Error())
			}
			return
		}
		if result.Notice != "" {
			notice = result.Notice
		}
	}

	if err := h.renderSourcesPanel(r.Context(), w, kbID, notice); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_sources_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) RefreshDocument(w http.ResponseWriter, r *http.Request) {
	kbID := r.URL.Query().Get("kb")
	if kbID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_kb", "kb query parameter is required")
		return
	}
	result, err := h.documentService.RefreshDocument(r.Context(), kbID, r.PathValue("documentID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound), errors.Is(err, ingest.ErrDocumentNotFound):
			http.NotFound(w, r)
		case errors.Is(err, ingest.ErrIngestionQueueFull):
			writeAPIError(w, http.StatusServiceUnavailable, "ingestion_queue_full", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "refresh_failed", err.Error())
		}
		return
	}
	notice := "Document refresh queued."
	if result.Notice != "" {
		notice = result.Notice
	}
	if err := h.renderSourcesPanel(r.Context(), w, kbID, notice); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_sources_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	kbID := r.URL.Query().Get("kb")
	if kbID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_kb", "kb query parameter is required")
		return
	}
	if err := h.documentService.DeleteDocument(r.Context(), kbID, r.PathValue("documentID")); err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound), errors.Is(err, ingest.ErrDocumentNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusBadRequest, "delete_failed", err.Error())
		}
		return
	}
	if err := h.renderSourcesPanel(r.Context(), w, kbID, "Document deleted."); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_sources_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) ReindexAll(w http.ResponseWriter, r *http.Request) {
	kbID := r.URL.Query().Get("kb")
	if kbID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_kb", "kb query parameter is required")
		return
	}
	if _, err := h.documentService.ReindexAll(r.Context(), kbID); err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound):
			http.NotFound(w, r)
		case errors.Is(err, ingest.ErrReindexInProgress):
			writeAPIError(w, http.StatusConflict, "reindex_in_progress", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "reindex_failed", err.Error())
		}
		return
	}
	if err := h.renderSourcesPanel(r.Context(), w, kbID, "Re-index all queued."); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_sources_failed", err.Error())
		return
	}
}

func (h *WorkspaceHandler) renderSourcesPanel(ctx context.Context, w http.ResponseWriter, kbID, notice string) error {
	kb, err := h.kbService.Get(ctx, kbID)
	if err != nil {
		return err
	}
	if kb == nil {
		return app.ErrKnowledgeBaseNotFound
	}
	documents, err := h.documentService.ListDocuments(ctx, kbID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return components.DocumentList(kb, documents, notice).Render(ctx, w)
}

func (h *WorkspaceHandler) loadWorkspaceData(ctx context.Context, selectedKBID string, selectedConversationID string) (templates.WorkspacePageData, error) {
	kbs, err := h.kbService.List(ctx)
	if err != nil {
		return templates.WorkspacePageData{}, err
	}

	var activeKB *domain.KnowledgeBase
	if selectedKBID != "" {
		for i := range kbs {
			if kbs[i].ID == selectedKBID {
				activeKB = &kbs[i]
				break
			}
		}
	}
	if activeKB == nil && len(kbs) > 0 {
		activeKB = &kbs[0]
		selectedKBID = activeKB.ID
	}

	conversations := []domain.Conversation{}
	if activeKB != nil {
		items, err := h.conversationService.List(ctx, activeKB.ID)
		if err != nil {
			return templates.WorkspacePageData{}, err
		}
		conversations = items
	}

	activeConversationID := selectedConversationID
	if activeConversationID != "" {
		found := false
		for _, conversation := range conversations {
			if conversation.ID == activeConversationID {
				found = true
				break
			}
		}
		if !found {
			activeConversationID = ""
		}
	}
	if activeConversationID == "" && len(conversations) > 0 {
		activeConversationID = conversations[0].ID
	}

	var activeConversation *domain.Conversation
	if activeConversationID != "" {
		for i := range conversations {
			if conversations[i].ID == activeConversationID {
				activeConversation = &conversations[i]
				break
			}
		}
	}

	documents := []domain.Document{}
	if activeKB != nil {
		items, err := h.documentService.ListDocuments(ctx, activeKB.ID)
		if err != nil {
			return templates.WorkspacePageData{}, err
		}
		documents = items
	}

	return templates.WorkspacePageData{
		KnowledgeBases:       kbs,
		ActiveKBID:           selectedKBID,
		ActiveKB:             activeKB,
		Conversations:        conversations,
		ActiveConversationID: activeConversationID,
		ActiveConversation:   activeConversation,
		Documents:            documents,
	}, nil
}
