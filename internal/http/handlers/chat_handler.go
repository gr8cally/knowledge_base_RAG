package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/domain"
)

type chatService interface {
	GetConversation(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error)
	ListMessages(ctx context.Context, kbID, conversationID string) ([]app.ChatMessageView, error)
	AddUserMessage(ctx context.Context, kbID, conversationID, content string) (domain.Message, error)
	StreamAssistant(ctx context.Context, kbID, conversationID, userMessageID string, stream func(app.ChatStreamEvent) error) error
}

type kbLookup interface {
	Get(ctx context.Context, id string) (*domain.KnowledgeBase, error)
}

type ChatHandler struct {
	service   chatService
	kbService kbLookup
}

func NewChatHandler(service chatService, kbService kbLookup) *ChatHandler {
	return &ChatHandler{service: service, kbService: kbService}
}

func (h *ChatHandler) Page(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("kbID")
	conversationID := r.PathValue("conversationID")

	kb, err := h.kbService.Get(r.Context(), kbID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "get_kb_failed", err.Error())
		return
	}
	if kb == nil {
		http.NotFound(w, r)
		return
	}

	conv, err := h.service.GetConversation(r.Context(), kbID, conversationID)
	if err != nil {
		if errors.Is(err, app.ErrConversationNotFound) || errors.Is(err, app.ErrKnowledgeBaseNotFound) {
			http.NotFound(w, r)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "get_conversation_failed", err.Error())
		return
	}

	http.Redirect(w, r, "/?kb="+kb.ID+"&conversation="+conv.ID, http.StatusSeeOther)
}

func (h *ChatHandler) MessagesAPI(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListMessages(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"))
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound), errors.Is(err, app.ErrConversationNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "list_messages_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ChatHandler) PostMessageAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	msg, err := h.service.AddUserMessage(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"), req.Content)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound), errors.Is(err, app.ErrConversationNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusBadRequest, "post_message_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"message":    msg,
		"status":     "accepted",
		"stream_url": fmt.Sprintf("/api/kbs/%s/conversations/%s/stream?message_id=%s", r.PathValue("kbID"), r.PathValue("conversationID"), msg.ID),
	})
}

func (h *ChatHandler) StreamAPI(w http.ResponseWriter, r *http.Request) {
	userMessageID := r.URL.Query().Get("message_id")
	if userMessageID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_message_id", "message_id is required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "sse_not_supported", "response writer does not support streaming")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	_ = h.service.StreamAssistant(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"), userMessageID, func(event app.ChatStreamEvent) error {
		if err := writeChatSSE(w, event); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
}

func writeChatSSE(w http.ResponseWriter, event app.ChatStreamEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: assistant\ndata: %s\n\n", payload)
	return err
}
