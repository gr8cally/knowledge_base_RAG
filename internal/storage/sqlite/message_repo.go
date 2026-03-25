package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"
)

type MessageRepo struct {
	dbPath string
}

const timestampLayout = time.RFC3339Nano

func NewMessageRepo(dbPath string) *MessageRepo {
	return &MessageRepo{dbPath: dbPath}
}

func (r *MessageRepo) Create(ctx context.Context, msg domain.Message) error {
	query := fmt.Sprintf(`
INSERT INTO messages (
  id, conversation_id, role, content, created_at
) VALUES (
  %s, %s, %s, %s, %s
);`,
		sqlQuote(msg.ID),
		sqlQuote(msg.ConversationID),
		sqlQuote(msg.Role),
		sqlQuote(msg.Content),
		sqlQuote(msg.CreatedAt.Format(timestampLayout)),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *MessageRepo) GetByID(ctx context.Context, conversationID, messageID string) (*domain.Message, error) {
	query := fmt.Sprintf(`
SELECT id, conversation_id, role, content, created_at
FROM messages
WHERE conversation_id = %s
  AND id = %s
LIMIT 1;`, sqlQuote(conversationID), sqlQuote(messageID))

	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var items []domain.Message
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (r *MessageRepo) ListByConversation(ctx context.Context, conversationID string) ([]domain.Message, error) {
	query := fmt.Sprintf(`
SELECT id, conversation_id, role, content, created_at
FROM messages
WHERE conversation_id = %s
ORDER BY created_at ASC;`, sqlQuote(conversationID))
	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return []domain.Message{}, nil
	}
	var items []domain.Message
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}
	return items, nil
}
