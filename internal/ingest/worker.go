package ingest

import (
	"context"
	"fmt"
	"log/slog"

	"knowledge_base_RAG/internal/domain"

	"github.com/tmc/langchaingo/schema"
)

type VectorIndexer interface {
	AddDocumentChunks(ctx context.Context, kbNamespace string, docs []schema.Document) error
	DeleteDocument(ctx context.Context, kbNamespace, documentID string) error
	DeleteNamespace(ctx context.Context, kbNamespace string) error
}

type Worker struct {
	logger       *slog.Logger
	chunkSize    int
	chunkOverlap int
	ocrEnabled   bool
	ocrLang      string
	indexer      VectorIndexer
}

func NewWorker(logger *slog.Logger, chunkSize, chunkOverlap int, ocrEnabled bool, ocrLang string, indexer VectorIndexer) *Worker {
	return &Worker{
		logger:       logger,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
		ocrEnabled:   ocrEnabled,
		ocrLang:      ocrLang,
		indexer:      indexer,
	}
}

func (w *Worker) Process(ctx context.Context, task JobTask) (domain.Document, error) {
	doc := task.Document

	result, err := LoadDocument(ctx, doc.StoragePath, doc.MimeType, w.ocrEnabled, w.ocrLang)
	if err != nil {
		return doc, err
	}

	chunks, err := ChunkText(result.Text, w.chunkSize, w.chunkOverlap)
	if err != nil {
		return doc, err
	}

	chunkDocs := make([]schema.Document, 0, len(chunks))
	for idx, chunk := range chunks {
		chunkDocs = append(chunkDocs, schema.Document{
			PageContent: chunk,
			Metadata: map[string]any{
				"kb_id":        doc.KBID,
				"document_id":  doc.ID,
				"source_label": doc.DisplayName,
				"chunk_index":  idx,
			},
		})
	}
	if err := w.indexer.AddDocumentChunks(ctx, task.KBNamespace, chunkDocs); err != nil {
		return doc, err
	}

	doc.ParserUsed = result.ParserUsed
	doc.ChunkCount = len(chunks)
	doc.Status = "ready"
	doc.ErrorMessage = ""

	w.logger.Info("document_indexed",
		"document_id", doc.ID,
		"display_name", doc.DisplayName,
		"chunk_count", len(chunks),
	)
	for idx, chunk := range chunks {
		if idx >= 3 {
			break
		}
		w.logger.Debug("indexed_chunk_preview",
			"document_id", doc.ID,
			"chunk_index", idx,
			"preview", previewChunk(chunk),
		)
	}

	return doc, nil
}

func previewChunk(chunk string) string {
	if len(chunk) <= 120 {
		return chunk
	}
	return fmt.Sprintf("%s...", chunk[:120])
}
