package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"
)

type KnowledgeBaseRepo struct {
	dbPath string
}

func NewKnowledgeBaseRepo(dbPath string) *KnowledgeBaseRepo {
	return &KnowledgeBaseRepo{dbPath: dbPath}
}

func (r *KnowledgeBaseRepo) Create(ctx context.Context, kb domain.KnowledgeBase) error {
	query := fmt.Sprintf(`
INSERT INTO knowledge_bases (id, name, description, namespace, created_at, updated_at, archived_at)
VALUES (%s, %s, %s, %s, %s, %s, NULL);`,
		sqlQuote(kb.ID),
		sqlQuote(kb.Name),
		sqlQuote(kb.Description),
		sqlQuote(kb.Namespace),
		sqlQuote(kb.CreatedAt.Format(time.RFC3339Nano)),
		sqlQuote(kb.UpdatedAt.Format(time.RFC3339Nano)),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *KnowledgeBaseRepo) ListActive(ctx context.Context) ([]domain.KnowledgeBase, error) {
	query := `
SELECT id, name, description, namespace, created_at, updated_at, archived_at
FROM knowledge_bases
WHERE archived_at IS NULL
ORDER BY updated_at DESC;`
	return r.queryMany(ctx, query)
}

func (r *KnowledgeBaseRepo) GetByID(ctx context.Context, id string) (*domain.KnowledgeBase, error) {
	query := fmt.Sprintf(`
SELECT id, name, description, namespace, created_at, updated_at, archived_at
FROM knowledge_bases
WHERE id = %s
  AND archived_at IS NULL
LIMIT 1;`, sqlQuote(id))

	items, err := r.queryMany(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (r *KnowledgeBaseRepo) Update(ctx context.Context, kb domain.KnowledgeBase) error {
	query := fmt.Sprintf(`
UPDATE knowledge_bases
SET name = %s,
    description = %s,
    updated_at = %s
WHERE id = %s
  AND archived_at IS NULL;`,
		sqlQuote(kb.Name),
		sqlQuote(kb.Description),
		sqlQuote(kb.UpdatedAt.Format(time.RFC3339Nano)),
		sqlQuote(kb.ID),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *KnowledgeBaseRepo) Archive(ctx context.Context, id string, archivedAt time.Time) error {
	query := fmt.Sprintf(`
UPDATE knowledge_bases
SET archived_at = %s,
    updated_at = %s
WHERE id = %s
  AND archived_at IS NULL;`,
		sqlQuote(archivedAt.Format(time.RFC3339Nano)),
		sqlQuote(archivedAt.Format(time.RFC3339Nano)),
		sqlQuote(id),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *KnowledgeBaseRepo) queryMany(ctx context.Context, query string) ([]domain.KnowledgeBase, error) {
	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return []domain.KnowledgeBase{}, nil
	}

	var items []domain.KnowledgeBase
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal knowledge bases: %w", err)
	}
	return items, nil
}
