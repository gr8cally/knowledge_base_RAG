package rag

import (
	"context"

	"knowledge_base_RAG/internal/vector"

	"github.com/tmc/langchaingo/schema"
)

type Retriever struct {
	store          *vector.Store
	kbNamespace    string
	topK           int
	scoreThreshold float32
}

func NewRetriever(store *vector.Store, kbNamespace string, topK int, scoreThreshold float32) *Retriever {
	return &Retriever{
		store:          store,
		kbNamespace:    kbNamespace,
		topK:           topK,
		scoreThreshold: scoreThreshold,
	}
}

func (r *Retriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	return r.store.SimilaritySearch(ctx, r.kbNamespace, query, r.topK, r.scoreThreshold)
}
