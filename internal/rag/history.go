package rag

import (
	"context"

	"knowledge_base_RAG/internal/domain"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type MessageLister interface {
	ListByConversation(ctx context.Context, conversationID string) ([]domain.Message, error)
}

type ReadOnlyHistory struct {
	conversationID string
	repo           MessageLister
}

var _ schema.ChatMessageHistory = (*ReadOnlyHistory)(nil)

func NewReadOnlyHistory(conversationID string, repo MessageLister) *ReadOnlyHistory {
	return &ReadOnlyHistory{
		conversationID: conversationID,
		repo:           repo,
	}
}

func (h *ReadOnlyHistory) Messages(ctx context.Context) ([]llms.ChatMessage, error) {
	items, err := h.repo.ListByConversation(ctx, h.conversationID)
	if err != nil {
		return nil, err
	}

	messages := make([]llms.ChatMessage, 0, len(items))
	for _, item := range items {
		switch item.Role {
		case "assistant":
			messages = append(messages, llms.AIChatMessage{Content: item.Content})
		default:
			messages = append(messages, llms.HumanChatMessage{Content: item.Content})
		}
	}
	return messages, nil
}

func (h *ReadOnlyHistory) AddMessage(context.Context, llms.ChatMessage) error { return nil }
func (h *ReadOnlyHistory) AddAIMessage(context.Context, string) error         { return nil }
func (h *ReadOnlyHistory) AddUserMessage(context.Context, string) error       { return nil }
func (h *ReadOnlyHistory) Clear(context.Context) error                        { return nil }
func (h *ReadOnlyHistory) SetMessages(context.Context, []llms.ChatMessage) error {
	return nil
}
