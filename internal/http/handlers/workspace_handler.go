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
	kbs, err := h.kbService.List(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "list_kbs_failed", err.Error())
		return
	}

	selectedKBID := r.URL.Query().Get("kb")
	selectedConversationID := r.URL.Query().Get("conversation")

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
		items, err := h.conversationService.List(r.Context(), activeKB.ID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "list_conversations_failed", err.Error())
			return
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

	data := templates.WorkspacePageData{
		KnowledgeBases:       kbs,
		ActiveKBID:           selectedKBID,
		ActiveKB:             activeKB,
		Conversations:        conversations,
		ActiveConversationID: activeConversationID,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.WorkspacePage(data).Render(r.Context(), w); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "render_workspace_failed", err.Error())
		return
	}
}
