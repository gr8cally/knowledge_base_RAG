package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"
)

type DocumentRepo struct {
	dbPath string
}

func NewDocumentRepo(dbPath string) *DocumentRepo {
	return &DocumentRepo{dbPath: dbPath}
}

func (r *DocumentRepo) Create(ctx context.Context, doc domain.Document) error {
	query := fmt.Sprintf(`
INSERT INTO documents (
  id, kb_id, source_type, display_name, normalized_name, source_uri,
  sha256, storage_path, mime_type, size_bytes, parser_used, chunk_count,
  status, error_message, created_at, updated_at
) VALUES (
  %s, %s, %s, %s, %s, %s,
  %s, %s, %s, %d, %s, %d,
  %s, %s, %s, %s
);`,
		sqlQuote(doc.ID),
		sqlQuote(doc.KBID),
		sqlQuote(doc.SourceType),
		sqlQuote(doc.DisplayName),
		sqlQuote(doc.NormalizedName),
		sqlQuote(doc.SourceURI),
		sqlQuote(doc.SHA256),
		sqlQuote(doc.StoragePath),
		sqlQuote(doc.MimeType),
		doc.SizeBytes,
		sqlQuote(doc.ParserUsed),
		doc.ChunkCount,
		sqlQuote(doc.Status),
		sqlQuote(doc.ErrorMessage),
		sqlQuote(doc.CreatedAt.Format(time.RFC3339Nano)),
		sqlQuote(doc.UpdatedAt.Format(time.RFC3339Nano)),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *DocumentRepo) Update(ctx context.Context, doc domain.Document) error {
	query := fmt.Sprintf(`
UPDATE documents
SET source_type = %s,
    display_name = %s,
    normalized_name = %s,
    source_uri = %s,
    sha256 = %s,
    storage_path = %s,
    mime_type = %s,
    size_bytes = %d,
    parser_used = %s,
    chunk_count = %d,
    status = %s,
    error_message = %s,
    updated_at = %s
WHERE id = %s;`,
		sqlQuote(doc.SourceType),
		sqlQuote(doc.DisplayName),
		sqlQuote(doc.NormalizedName),
		sqlQuote(doc.SourceURI),
		sqlQuote(doc.SHA256),
		sqlQuote(doc.StoragePath),
		sqlQuote(doc.MimeType),
		doc.SizeBytes,
		sqlQuote(doc.ParserUsed),
		doc.ChunkCount,
		sqlQuote(doc.Status),
		sqlQuote(doc.ErrorMessage),
		sqlQuote(doc.UpdatedAt.Format(time.RFC3339Nano)),
		sqlQuote(doc.ID),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *DocumentRepo) ListByKB(ctx context.Context, kbID string) ([]domain.Document, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, source_type, display_name, normalized_name, source_uri,
       sha256, storage_path, mime_type, size_bytes, parser_used, chunk_count,
       status, error_message, created_at, updated_at
FROM documents
WHERE kb_id = %s
ORDER BY updated_at DESC;`, sqlQuote(kbID))
	return r.queryMany(ctx, query)
}

func (r *DocumentRepo) GetByID(ctx context.Context, kbID, documentID string) (*domain.Document, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, source_type, display_name, normalized_name, source_uri,
       sha256, storage_path, mime_type, size_bytes, parser_used, chunk_count,
       status, error_message, created_at, updated_at
FROM documents
WHERE kb_id = %s
  AND id = %s
LIMIT 1;`, sqlQuote(kbID), sqlQuote(documentID))

	items, err := r.queryMany(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (r *DocumentRepo) FindByKBAndNormalizedName(ctx context.Context, kbID, normalizedName string) (*domain.Document, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, source_type, display_name, normalized_name, source_uri,
       sha256, storage_path, mime_type, size_bytes, parser_used, chunk_count,
       status, error_message, created_at, updated_at
FROM documents
WHERE kb_id = %s
  AND normalized_name = %s
LIMIT 1;`, sqlQuote(kbID), sqlQuote(normalizedName))

	items, err := r.queryMany(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (r *DocumentRepo) Delete(ctx context.Context, kbID, documentID string) error {
	query := fmt.Sprintf(`
DELETE FROM documents
WHERE kb_id = %s
  AND id = %s;`, sqlQuote(kbID), sqlQuote(documentID))
	return execSQL(ctx, r.dbPath, query)
}

func (r *DocumentRepo) queryMany(ctx context.Context, query string) ([]domain.Document, error) {
	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return []domain.Document{}, nil
	}

	var items []domain.Document
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal documents: %w", err)
	}
	return items, nil
}
