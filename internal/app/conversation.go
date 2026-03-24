package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrKnowledgeBaseNotFound = errors.New("knowledge base not found")
	ErrConversationNotFound  = errors.New("conversation not found")
)

type ConversationRepository interface {
	Create(ctx context.Context, conv domain.Conversation) error
	ListActiveByKB(ctx context.Context, kbID string) ([]domain.Conversation, error)
	GetByID(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error)
	Update(ctx context.Context, conv domain.Conversation) error
	Archive(ctx context.Context, kbID, conversationID string, archivedAt time.Time) error
	TouchLastMessage(ctx context.Context, kbID, conversationID string, at time.Time) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg domain.Message) error
	ListByConversation(ctx context.Context, conversationID string) ([]domain.Message, error)
}

type ConversationService struct {
	kbService   *KnowledgeBaseService
	convRepo    ConversationRepository
	messageRepo MessageRepository
}

func NewConversationService(kbService *KnowledgeBaseService, convRepo ConversationRepository, messageRepo MessageRepository) *ConversationService {
	return &ConversationService{
		kbService:   kbService,
		convRepo:    convRepo,
		messageRepo: messageRepo,
	}
}

func (s *ConversationService) List(ctx context.Context, kbID string) ([]domain.Conversation, error) {
	kb, err := s.kbService.Get(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, ErrKnowledgeBaseNotFound
	}
	return s.convRepo.ListActiveByKB(ctx, kbID)
}

func (s *ConversationService) Get(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error) {
	kb, err := s.kbService.Get(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, ErrKnowledgeBaseNotFound
	}
	conv, err := s.convRepo.GetByID(ctx, kbID, conversationID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, ErrConversationNotFound
	}
	return conv, nil
}

func (s *ConversationService) Create(ctx context.Context, kbID, title string) (domain.Conversation, error) {
	kb, err := s.kbService.Get(ctx, kbID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if kb == nil {
		return domain.Conversation{}, ErrKnowledgeBaseNotFound
	}
	if strings.TrimSpace(title) == "" {
		title = "New Conversation"
	}
	now := time.Now().UTC()
	conv := domain.Conversation{
		ID:        uuid.NewString(),
		KBID:      kbID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return domain.Conversation{}, err
	}
	return conv, nil
}

func (s *ConversationService) Update(ctx context.Context, kbID, conversationID, title string) (*domain.Conversation, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	conv, err := s.Get(ctx, kbID, conversationID)
	if err != nil {
		return nil, err
	}
	conv.Title = title
	conv.UpdatedAt = time.Now().UTC()
	if err := s.convRepo.Update(ctx, *conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *ConversationService) Archive(ctx context.Context, kbID, conversationID string) error {
	if _, err := s.Get(ctx, kbID, conversationID); err != nil {
		return err
	}
	return s.convRepo.Archive(ctx, kbID, conversationID, time.Now().UTC())
}

func (s *ConversationService) ListMessages(ctx context.Context, kbID, conversationID string) ([]domain.Message, error) {
	if _, err := s.Get(ctx, kbID, conversationID); err != nil {
		return nil, err
	}
	return s.messageRepo.ListByConversation(ctx, conversationID)
}

func (s *ConversationService) AddUserMessage(ctx context.Context, kbID, conversationID, content string) (domain.Message, error) {
	if strings.TrimSpace(content) == "" {
		return domain.Message{}, fmt.Errorf("content is required")
	}
	if _, err := s.Get(ctx, kbID, conversationID); err != nil {
		return domain.Message{}, err
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
	if err := s.convRepo.TouchLastMessage(ctx, kbID, conversationID, now); err != nil {
		return domain.Message{}, err
	}
	return msg, nil
}
