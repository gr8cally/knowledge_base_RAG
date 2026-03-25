package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"knowledge_base_RAG/internal/domain"
	"knowledge_base_RAG/internal/rag"
	"knowledge_base_RAG/internal/vector"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

var ErrAssistantResponseInProgress = errors.New("assistant response already in progress")

type CitationRepository interface {
	CreateBatch(ctx context.Context, citations []domain.Citation) error
	ListByMessageIDs(ctx context.Context, messageIDs []string) ([]domain.Citation, error)
}

type ChatMessageView struct {
	Message   domain.Message    `json:"message"`
	Citations []domain.Citation `json:"citations"`
}

type ChatStreamEvent struct {
	Type        string            `json:"type"`
	UserMessage string            `json:"user_message_id,omitempty"`
	MessageID   string            `json:"message_id,omitempty"`
	Content     string            `json:"content,omitempty"`
	Citations   []domain.Citation `json:"citations,omitempty"`
	Error       string            `json:"error,omitempty"`
	At          time.Time         `json:"at"`
}

type ChatService struct {
	kbService            *KnowledgeBaseService
	conversationRepo     ConversationRepository
	messageRepo          MessageRepository
	citationRepo         CitationRepository
	vectorStore          *vector.Store
	llm                  llms.Model
	ragTopK              int
	ragScoreThreshold    float32
	chatHistoryMaxTurns  int
	mu                   sync.Mutex
	inFlightUserMessages map[string]struct{}
}

func NewChatService(
	kbService *KnowledgeBaseService,
	conversationRepo ConversationRepository,
	messageRepo MessageRepository,
	citationRepo CitationRepository,
	vectorStore *vector.Store,
	llm llms.Model,
	ragTopK int,
	ragScoreThreshold float32,
	chatHistoryMaxTurns int,
) *ChatService {
	return &ChatService{
		kbService:            kbService,
		conversationRepo:     conversationRepo,
		messageRepo:          messageRepo,
		citationRepo:         citationRepo,
		vectorStore:          vectorStore,
		llm:                  llm,
		ragTopK:              ragTopK,
		ragScoreThreshold:    ragScoreThreshold,
		chatHistoryMaxTurns:  chatHistoryMaxTurns,
		inFlightUserMessages: make(map[string]struct{}),
	}
}

func (s *ChatService) GetConversation(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error) {
	kb, err := s.kbService.Get(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, ErrKnowledgeBaseNotFound
	}

	conv, err := s.conversationRepo.GetByID(ctx, kbID, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, ErrConversationNotFound
	}
	return conv, nil
}

func (s *ChatService) ListMessages(ctx context.Context, kbID, conversationID string) ([]ChatMessageView, error) {
	if _, err := s.GetConversation(ctx, kbID, conversationID); err != nil {
		return nil, err
	}

	messages, err := s.messageRepo.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return []ChatMessageView{}, nil
	}

	messageIDs := make([]string, 0, len(messages))
	for _, message := range messages {
		messageIDs = append(messageIDs, message.ID)
	}

	citations, err := s.citationRepo.ListByMessageIDs(ctx, messageIDs)
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]domain.Citation, len(messages))
	for _, citation := range citations {
		grouped[citation.MessageID] = append(grouped[citation.MessageID], citation)
	}

	items := make([]ChatMessageView, 0, len(messages))
	for _, message := range messages {
		items = append(items, ChatMessageView{
			Message:   message,
			Citations: grouped[message.ID],
		})
	}
	return items, nil
}

func (s *ChatService) AddUserMessage(ctx context.Context, kbID, conversationID, content string) (domain.Message, error) {
	if _, err := s.GetConversation(ctx, kbID, conversationID); err != nil {
		return domain.Message{}, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.Message{}, fmt.Errorf("content is required")
	}

	now := time.Now().UTC()
	msg := domain.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		Role:           "user",
		Content:        content,
		CreatedAt:      now,
	}
	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return domain.Message{}, err
	}
	if err := s.conversationRepo.TouchLastMessage(ctx, kbID, conversationID, now); err != nil {
		return domain.Message{}, err
	}
	return msg, nil
}

func (s *ChatService) StreamAssistant(
	ctx context.Context,
	kbID, conversationID, userMessageID string,
	stream func(ChatStreamEvent) error,
) error {
	kb, err := s.kbService.Get(ctx, kbID)
	if err != nil {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: err.Error(), At: time.Now().UTC()})
		return err
	}
	if kb == nil {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: ErrKnowledgeBaseNotFound.Error(), At: time.Now().UTC()})
		return ErrKnowledgeBaseNotFound
	}
	if _, err := s.GetConversation(ctx, kbID, conversationID); err != nil {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: err.Error(), At: time.Now().UTC()})
		return err
	}

	userMessage, err := s.messageRepo.GetByID(ctx, conversationID, userMessageID)
	if err != nil {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: err.Error(), At: time.Now().UTC()})
		return err
	}
	if userMessage == nil || userMessage.Role != "user" {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: ErrConversationNotFound.Error(), At: time.Now().UTC()})
		return ErrConversationNotFound
	}

	existingReply, existingCitations, err := s.findExistingReply(ctx, conversationID, userMessageID)
	if err != nil {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: err.Error(), At: time.Now().UTC()})
		return err
	}
	if existingReply != nil {
		return stream(ChatStreamEvent{
			Type:        "completed",
			UserMessage: userMessageID,
			MessageID:   existingReply.ID,
			Content:     existingReply.Content,
			Citations:   existingCitations,
			At:          time.Now().UTC(),
		})
	}

	if !s.beginGeneration(userMessageID) {
		_ = stream(ChatStreamEvent{Type: "error", UserMessage: userMessageID, Error: ErrAssistantResponseInProgress.Error(), At: time.Now().UTC()})
		return ErrAssistantResponseInProgress
	}
	defer s.endGeneration(userMessageID)

	if err := stream(ChatStreamEvent{
		Type:        "snapshot",
		UserMessage: userMessageID,
		At:          time.Now().UTC(),
	}); err != nil {
		return err
	}

	retriever := rag.NewRetriever(s.vectorStore, kb.Namespace, s.ragTopK, s.ragScoreThreshold)
	history := rag.NewReadOnlyHistory(conversationID, s.messageRepo)
	chain := rag.NewConversationChain(s.llm, retriever, history, s.chatHistoryMaxTurns)

	var answerBuilder strings.Builder
	output, err := chains.Call(ctx, chain, map[string]any{
		"question": userMessage.Content,
	}, chains.WithStreamingFunc(func(streamCtx context.Context, chunk []byte) error {
		text := string(chunk)
		answerBuilder.WriteString(text)
		return stream(ChatStreamEvent{
			Type:        "token",
			UserMessage: userMessageID,
			Content:     text,
			At:          time.Now().UTC(),
		})
	}))
	if err != nil {
		_ = stream(ChatStreamEvent{
			Type:        "error",
			UserMessage: userMessageID,
			Error:       err.Error(),
			At:          time.Now().UTC(),
		})
		return err
	}

	answer, _ := output["text"].(string)
	if answer == "" {
		answer = answerBuilder.String()
	}

	sourceDocs, _ := output["source_documents"].([]schema.Document)
	citations := rag.BuildCitations(sourceDocs)
	if len(sourceDocs) == 0 {
		answer = "I don't have enough evidence in the selected knowledge base to answer that."
	}

	assistantMessage := domain.Message{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        rag.AppendCitationMarkers(answer, citations),
		CreatedAt:      time.Now().UTC(),
	}
	for idx := range citations {
		citations[idx].MessageID = assistantMessage.ID
	}

	if err := s.messageRepo.Create(ctx, assistantMessage); err != nil {
		_ = stream(ChatStreamEvent{
			Type:        "error",
			UserMessage: userMessageID,
			Error:       err.Error(),
			At:          time.Now().UTC(),
		})
		return err
	}
	if err := s.citationRepo.CreateBatch(ctx, citations); err != nil {
		_ = stream(ChatStreamEvent{
			Type:        "error",
			UserMessage: userMessageID,
			Error:       err.Error(),
			At:          time.Now().UTC(),
		})
		return err
	}
	if err := s.conversationRepo.TouchLastMessage(ctx, kbID, conversationID, assistantMessage.CreatedAt); err != nil {
		_ = stream(ChatStreamEvent{
			Type:        "error",
			UserMessage: userMessageID,
			Error:       err.Error(),
			At:          time.Now().UTC(),
		})
		return err
	}

	return stream(ChatStreamEvent{
		Type:        "completed",
		UserMessage: userMessageID,
		MessageID:   assistantMessage.ID,
		Content:     assistantMessage.Content,
		Citations:   citations,
		At:          time.Now().UTC(),
	})
}

func (s *ChatService) findExistingReply(ctx context.Context, conversationID, userMessageID string) (*domain.Message, []domain.Citation, error) {
	messages, err := s.messageRepo.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, nil, err
	}

	for idx, message := range messages {
		if message.ID != userMessageID {
			continue
		}
		if idx+1 >= len(messages) || messages[idx+1].Role != "assistant" {
			return nil, nil, nil
		}

		reply := messages[idx+1]
		citations, err := s.citationRepo.ListByMessageIDs(ctx, []string{reply.ID})
		if err != nil {
			return nil, nil, err
		}
		return &reply, citations, nil
	}

	return nil, nil, nil
}

func (s *ChatService) beginGeneration(userMessageID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.inFlightUserMessages[userMessageID]; exists {
		return false
	}
	s.inFlightUserMessages[userMessageID] = struct{}{}
	return true
}

func (s *ChatService) endGeneration(userMessageID string) {
	s.mu.Lock()
	delete(s.inFlightUserMessages, userMessageID)
	s.mu.Unlock()
}
