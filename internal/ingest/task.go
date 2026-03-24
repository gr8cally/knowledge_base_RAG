package ingest

import "knowledge_base_RAG/internal/domain"

type JobTask struct {
	Job         domain.IngestionJob
	Document    domain.Document
	KBNamespace string
}
