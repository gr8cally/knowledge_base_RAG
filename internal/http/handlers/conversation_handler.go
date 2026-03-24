package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/domain"
)

type conversationService interface {
	List(ctx context.Context, kbID string) ([]domain.Conversation, error)
	Create(ctx context.Context, kbID, title string) (domain.Conversation, error)
	Get(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error)
	Update(ctx context.Context, kbID, conversationID, title string) (*domain.Conversation, error)
	Archive(ctx context.Context, kbID, conversationID string) error
}

type ConversationHandler struct {
	service conversationService
}

func NewConversationHandler(service conversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

func (h *ConversationHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.PathValue("kbID"))
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "list_conversations_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ConversationHandler) CreateAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	conv, err := h.service.Create(r.Context(), r.PathValue("kbID"), req.Title)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusBadRequest, "create_conversation_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

func (h *ConversationHandler) UpdateAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	conv, err := h.service.Update(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"), req.Title)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound), errors.Is(err, app.ErrConversationNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusBadRequest, "update_conversation_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

func (h *ConversationHandler) ArchiveAPI(w http.ResponseWriter, r *http.Request) {
	err := h.service.Archive(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"))
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound), errors.Is(err, app.ErrConversationNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "archive_conversation_failed", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
