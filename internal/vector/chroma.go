package vector

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	lcchroma "github.com/tmc/langchaingo/vectorstores/chroma"
)

type Store struct {
	store vectorstores.VectorStore
}

func NewStore(chromaURL, collectionName string, embedder langchainembeddings.Embedder) (*Store, error) {
	store, err := lcchroma.New(
		lcchroma.WithChromaURL(chromaURL),
		lcchroma.WithEmbedder(embedder),
		lcchroma.WithNameSpace(collectionName),
	)
	if err != nil {
		return nil, fmt.Errorf("new langchaingo chroma store: %w", err)
	}

	return &Store{store: store}, nil
}

func (s *Store) AddDocumentChunks(ctx context.Context, kbNamespace string, docs []schema.Document) error {
	if len(docs) == 0 {
		return nil
	}

	_, err := s.store.AddDocuments(ctx, docs, vectorstores.WithNameSpace(kbNamespace))
	if err != nil {
		return fmt.Errorf("add documents to chroma: %w", err)
	}
	return nil
}

func (s *Store) SimilaritySearch(ctx context.Context, kbNamespace, query string, numDocuments int, scoreThreshold float32) ([]schema.Document, error) {
	docs, err := s.store.SimilaritySearch(
		ctx,
		query,
		numDocuments,
		vectorstores.WithNameSpace(kbNamespace),
		vectorstores.WithScoreThreshold(scoreThreshold),
	)
	if err != nil {
		return nil, fmt.Errorf("chroma similarity search: %w", err)
	}
	return docs, nil
}

// CheckHealth verifies that the configured Chroma endpoint responds to a heartbeat request.
func CheckHealth(ctx context.Context, chromaURL string, httpClient *http.Client) error {
	baseURL := strings.TrimRight(chromaURL, "/")
	paths := []string{"/api/v1/heartbeat", "/api/v2/heartbeat"}

	var lastErr error
	for _, path := range paths {
		pathCtx, cancel := context.WithTimeout(ctx, time.Second)
		req, err := http.NewRequestWithContext(pathCtx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			cancel()
			return fmt.Errorf("build chroma request: %w", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		cancel()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return fmt.Errorf("chroma healthcheck failed: %w", lastErr)
}
