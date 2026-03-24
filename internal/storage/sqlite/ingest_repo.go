package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"
)

type IngestRepo struct {
	dbPath string
}

func NewIngestRepo(dbPath string) *IngestRepo {
	return &IngestRepo{dbPath: dbPath}
}

func (r *IngestRepo) Create(ctx context.Context, job domain.IngestionJob) error {
	started := "NULL"
	finished := "NULL"
	if job.StartedAt != nil {
		started = sqlQuote(job.StartedAt.Format(time.RFC3339Nano))
	}
	if job.FinishedAt != nil {
		finished = sqlQuote(job.FinishedAt.Format(time.RFC3339Nano))
	}

	query := fmt.Sprintf(`
INSERT INTO ingestion_jobs (
  id, kb_id, trigger_type, status, total_items, processed_items, skipped_items,
  failed_items, error_message, created_at, started_at, finished_at
) VALUES (
  %s, %s, %s, %s, %d, %d, %d,
  %d, %s, %s, %s, %s
);`,
		sqlQuote(job.ID),
		sqlQuote(job.KBID),
		sqlQuote(job.TriggerType),
		sqlQuote(job.Status),
		job.TotalItems,
		job.ProcessedItems,
		job.SkippedItems,
		job.FailedItems,
		sqlQuote(job.ErrorMessage),
		sqlQuote(job.CreatedAt.Format(time.RFC3339Nano)),
		started,
		finished,
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *IngestRepo) Update(ctx context.Context, job domain.IngestionJob) error {
	started := "NULL"
	finished := "NULL"
	if job.StartedAt != nil {
		started = sqlQuote(job.StartedAt.Format(time.RFC3339Nano))
	}
	if job.FinishedAt != nil {
		finished = sqlQuote(job.FinishedAt.Format(time.RFC3339Nano))
	}

	query := fmt.Sprintf(`
UPDATE ingestion_jobs
SET status = %s,
    total_items = %d,
    processed_items = %d,
    skipped_items = %d,
    failed_items = %d,
    error_message = %s,
    started_at = %s,
    finished_at = %s
WHERE id = %s;`,
		sqlQuote(job.Status),
		job.TotalItems,
		job.ProcessedItems,
		job.SkippedItems,
		job.FailedItems,
		sqlQuote(job.ErrorMessage),
		started,
		finished,
		sqlQuote(job.ID),
	)
	return execSQL(ctx, r.dbPath, query)
}

func (r *IngestRepo) ListByKB(ctx context.Context, kbID string) ([]domain.IngestionJob, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, trigger_type, status, total_items, processed_items, skipped_items,
       failed_items, error_message, created_at, started_at, finished_at
FROM ingestion_jobs
WHERE kb_id = %s
ORDER BY created_at DESC;`, sqlQuote(kbID))
	return r.queryMany(ctx, query)
}

func (r *IngestRepo) GetByID(ctx context.Context, kbID, jobID string) (*domain.IngestionJob, error) {
	query := fmt.Sprintf(`
SELECT id, kb_id, trigger_type, status, total_items, processed_items, skipped_items,
       failed_items, error_message, created_at, started_at, finished_at
FROM ingestion_jobs
WHERE kb_id = %s
  AND id = %s
LIMIT 1;`, sqlQuote(kbID), sqlQuote(jobID))

	items, err := r.queryMany(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (r *IngestRepo) queryMany(ctx context.Context, query string) ([]domain.IngestionJob, error) {
	out, err := execSQLiteJSON(ctx, r.dbPath, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return []domain.IngestionJob{}, nil
	}

	var items []domain.IngestionJob
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("unmarshal ingestion jobs: %w", err)
	}
	return items, nil
}
