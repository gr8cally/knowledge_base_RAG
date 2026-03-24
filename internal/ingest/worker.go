package ingest

import (
	"context"
	"fmt"
	"log/slog"

	"knowledge_base_RAG/internal/domain"
)

type Worker struct {
	logger       *slog.Logger
	chunkSize    int
	chunkOverlap int
	ocrEnabled   bool
	ocrLang      string
}

func NewWorker(logger *slog.Logger, chunkSize, chunkOverlap int, ocrEnabled bool, ocrLang string) *Worker {
	return &Worker{
		logger:       logger,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
		ocrEnabled:   ocrEnabled,
		ocrLang:      ocrLang,
	}
}

func (w *Worker) Process(ctx context.Context, doc *domain.Document) error {
	result, err := LoadDocument(ctx, doc.StoragePath, doc.MimeType, w.ocrEnabled, w.ocrLang)
	if err != nil {
		return err
	}

	chunks := ChunkText(result.Text, w.chunkSize, w.chunkOverlap)
	doc.ParserUsed = result.ParserUsed
	doc.ChunkCount = len(chunks)
	doc.Status = "ready"
	doc.ErrorMessage = ""

	w.logger.Info("dry_run_chunks",
		"document_id", doc.ID,
		"display_name", doc.DisplayName,
		"chunk_count", len(chunks),
	)
	for idx, chunk := range chunks {
		if idx >= 3 {
			break
		}
		w.logger.Debug("dry_run_chunk_preview",
			"document_id", doc.ID,
			"chunk_index", idx,
			"preview", previewChunk(chunk),
		)
	}

	return nil
}

func previewChunk(chunk string) string {
	if len(chunk) <= 120 {
		return chunk
	}
	return fmt.Sprintf("%s...", chunk[:120])
}
