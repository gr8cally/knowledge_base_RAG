package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"
)

type ConversationRepo struct {
	dbPath string
}

func NewConversationRepo(dbPath string) *ConversationRepo {
	return &ConversationRepo{dbPath: dbPath}
}

func (r *ConversationRepo) Create(ctx context.Context, conv domain.Conversation) error {
	lastMessageAt := "NULL"
	archivedAt := "NULL"
	if conv.LastMessageAt != nil {
		lastMessageAt = sqlQuote(conv.LastMessageAt.Format(time.RFC3339Nano))
	}
	if conv.ArchivedAt != nil {
		archivedAt = sqlQuote(conv.ArchivedAt.Format(time.RFC3339Nano))
	}

	query := fmt.Sprintf(`
INSERT INTO conversations (
  id, kb_id, title, created_at, updated_at, last_message_at, archived_at
) VALUES (
  %s, %s, %s, %s, %s, %s, %s
);`,
		sqlQuote(conv.ID),
		sqlQuote(conv.KBID),
		sqlQuote(conv.Title),
		sqlQuote(conv.CreatedAt.Format(time.RFC3339Nano)),
		sqlQuote(conv.UpdatedAt.Format(time.RFC3339Nano)),
		lastMessageAt,
		archivedAt,
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *ConversationRepo) ListActiveByKB(ctx context.Context, kbID string) ([]domain.Conversation, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, title, created_at, updated_at, last_message_at, archived_at
FROM conversations
WHERE kb_id = %s
  AND archived_at IS NULL
ORDER BY COALESCE(last_message_at, created_at) DESC;`, sqlQuote(kbID))
	return r.queryMany(ctx, query)
}

func (r *ConversationRepo) GetByID(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, title, created_at, updated_at, last_message_at, archived_at
FROM conversations
WHERE kb_id = %s
  AND id = %s
  AND archived_at IS NULL
LIMIT 1;`, sqlQuote(kbID), sqlQuote(conversationID))
	items, err := r.queryMany(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (r *ConversationRepo) Update(ctx context.Context, conv domain.Conversation) error {
	lastMessageAt := "NULL"
	if conv.LastMessageAt != nil {
		lastMessageAt = sqlQuote(conv.LastMessageAt.Format(time.RFC3339Nano))
	}

	query := fmt.Sprintf(`
UPDATE conversations
SET title = %s,
    updated_at = %s
WHERE id = %s
  AND kb_id = %s
  AND archived_at IS NULL;`,
		sqlQuote(conv.Title),
		sqlQuote(conv.UpdatedAt.Format(time.RFC3339Nano)),
		sqlQuote(conv.ID),
		sqlQuote(conv.KBID),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *ConversationRepo) Archive(ctx context.Context, kbID, conversationID string, archivedAt time.Time) error {
	query := fmt.Sprintf(`
UPDATE conversations
SET archived_at = %s,
    updated_at = %s
WHERE id = %s
  AND kb_id = %s
  AND archived_at IS NULL;`,
		sqlQuote(archivedAt.Format(time.RFC3339Nano)),
		sqlQuote(archivedAt.Format(time.RFC3339Nano)),
		sqlQuote(conversationID),
		sqlQuote(kbID),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *ConversationRepo) TouchLastMessage(ctx context.Context, kbID, conversationID string, at time.Time) error {
	query := fmt.Sprintf(`
UPDATE conversations
SET last_message_at = %s,
    updated_at = %s
WHERE id = %s
  AND kb_id = %s
  AND archived_at IS NULL;`,
		sqlQuote(at.Format(time.RFC3339Nano)),
		sqlQuote(at.Format(time.RFC3339Nano)),
		sqlQuote(conversationID),
		sqlQuote(kbID),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *ConversationRepo) queryMany(ctx context.Context, query string) ([]domain.Conversation, error) {
	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return []domain.Conversation{}, nil
	}
	var items []domain.Conversation
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal conversations: %w", err)
	}
	return items, nil
}
