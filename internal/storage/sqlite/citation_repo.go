package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"knowledge_base_RAG/internal/domain"
)

type CitationRepo struct {
	dbPath string
}

func NewCitationRepo(dbPath string) *CitationRepo {
	return &CitationRepo{dbPath: dbPath}
}

func (r *CitationRepo) CreateBatch(ctx context.Context, citations []domain.Citation) error {
	if len(citations) == 0 {
		return nil
	}

	values := make([]string, 0, len(citations))
	for _, citation := range citations {
		values = append(values, fmt.Sprintf("(%s, %s, %d, %s, %s, %s, %d, %.8f, %s)",
			sqlQuote(citation.ID),
			sqlQuote(citation.MessageID),
			citation.CitationIndex,
			sqlQuote(citation.DocumentID),
			sqlQuote(citation.SourceLabel),
			sqlQuote(citation.Excerpt),
			citation.ChunkIndex,
			citation.Score,
			sqlQuote(citation.CreatedAt.Format(timestampLayout)),
		))
	}

	query := `
INSERT INTO message_citations (
  id, message_id, citation_index, document_id, source_label, excerpt, chunk_index, score, created_at
) VALUES ` + strings.Join(values, ",\n") + ";"

	return execSQL(ctx, r.dbPath, query)
}

func (r *CitationRepo) ListByMessageIDs(ctx context.Context, messageIDs []string) ([]domain.Citation, error) {
	if len(messageIDs) == 0 {
		return []domain.Citation{}, nil
	}

	quotedIDs := make([]string, 0, len(messageIDs))
	for _, messageID := range messageIDs {
		quotedIDs = append(quotedIDs, sqlQuote(messageID))
	}

	query := `
SELECT id, message_id, citation_index, document_id, source_label, excerpt, chunk_index, score, created_at
FROM message_citations
WHERE message_id IN (` + strings.Join(quotedIDs, ", ") + `)
ORDER BY message_id ASC, citation_index ASC;`

	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return []domain.Citation{}, nil
	}

	var items []domain.Citation
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal citations: %w", err)
	}
	return items, nil
}
