package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"

	"github.com/google/uuid"
)

type KnowledgeBaseRepository interface {
	Create(ctx context.Context, kb domain.KnowledgeBase) error
	ListActive(ctx context.Context) ([]domain.KnowledgeBase, error)
	GetByID(ctx context.Context, id string) (*domain.KnowledgeBase, error)
	Update(ctx context.Context, kb domain.KnowledgeBase) error
	Archive(ctx context.Context, id string, archivedAt time.Time) error
}

type KnowledgeBaseService struct {
	repo KnowledgeBaseRepository
}

func NewKnowledgeBaseService(repo KnowledgeBaseRepository) *KnowledgeBaseService {
	return &KnowledgeBaseService{repo: repo}
}

func (s *KnowledgeBaseService) List(ctx context.Context) ([]domain.KnowledgeBase, error) {
	return s.repo.ListActive(ctx)
}

func (s *KnowledgeBaseService) Get(ctx context.Context, id string) (*domain.KnowledgeBase, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *KnowledgeBaseService) Create(ctx context.Context, name, description string) (domain.KnowledgeBase, error) {
	now := time.Now().UTC()
	kb := domain.KnowledgeBase{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		Namespace:   "kb_" + uuid.NewString(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// Use the persistent id as the namespace suffix expected by the spec.
	kb.Namespace = "kb_" + kb.ID

	if err := validateKB(kb.Name); err != nil {
		return domain.KnowledgeBase{}, err
	}
	if err := s.repo.Create(ctx, kb); err != nil {
		return domain.KnowledgeBase{}, err
	}
	return kb, nil
}

func (s *KnowledgeBaseService) Update(ctx context.Context, id, name, description string) (*domain.KnowledgeBase, error) {
	if err := validateKB(name); err != nil {
		return nil, err
	}

	kb, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, nil
	}

	kb.Name = name
	kb.Description = description
	kb.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, *kb); err != nil {
		return nil, err
	}
	return kb, nil
}

func (s *KnowledgeBaseService) Archive(ctx context.Context, id string) error {
	return s.repo.Archive(ctx, id, time.Now().UTC())
}

func validateKB(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
