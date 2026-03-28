package handlers

import (
	"context"
	"net/http"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/domain"
	"knowledge_base_RAG/internal/web/templates"
)

type workspaceConversationService interface {
	List(ctx context.Context, kbID string) ([]domain.Conversation, error)
	Create(ctx context.Context, kbID, title string) (domain.Conversation, error)
}

type WorkspaceHandler struct {
	kbService           *app.KnowledgeBaseService
	conversationService workspaceConversationService
}

func NewWorkspaceHandler(kbService *app.KnowledgeBaseService, conversationService workspaceConversationService) *WorkspaceHandler {
	return &WorkspaceHandler{
		kbService:           kbService,
		conversationService: conversationService,
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

	return templates.WorkspacePageData{
		KnowledgeBases:       kbs,
		ActiveKBID:           selectedKBID,
		ActiveKB:             activeKB,
		Conversations:        conversations,
		ActiveConversationID: activeConversationID,
		ActiveConversation:   activeConversation,
	}, nil
}
