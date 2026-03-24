package vector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	chromaemb "github.com/amikos-tech/chroma-go/pkg/embeddings"
	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
)

const namespaceMetadataKey = "kb_namespace"

type Store struct {
	client         chroma.Client
	embedder       langchainembeddings.Embedder
	collectionName string

	mu         sync.Mutex
	collection chroma.Collection
}

func NewStore(chromaURL, collectionName string, embedder langchainembeddings.Embedder, httpClient *http.Client) (*Store, error) {
	client, err := chroma.NewHTTPClient(
		chroma.WithBaseURL(chromaURL),
		chroma.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("new chroma v2 client: %w", err)
	}

	return &Store{
		client:         client,
		embedder:       embedder,
		collectionName: collectionName,
	}, nil
}

func (s *Store) AddDocumentChunks(ctx context.Context, kbNamespace string, docs []schema.Document) error {
	if len(docs) == 0 {
		return nil
	}

	collection, err := s.getCollection(ctx)
	if err != nil {
		return err
	}

	texts := make([]string, 0, len(docs))
	ids := make([]chroma.DocumentID, 0, len(docs))
	metadatas := make([]chroma.DocumentMetadata, 0, len(docs))
	for idx, doc := range docs {
		metadata, err := buildMetadata(doc.Metadata, kbNamespace)
		if err != nil {
			return fmt.Errorf("build chroma metadata: %w", err)
		}
		texts = append(texts, doc.PageContent)
		ids = append(ids, chroma.DocumentID(chunkID(kbNamespace, doc, idx)))
		metadatas = append(metadatas, metadata)
	}

	rawEmbeddings, err := s.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed document chunks: %w", err)
	}

	embeddings, err := chromaemb.NewEmbeddingsFromFloat32(rawEmbeddings)
	if err != nil {
		return fmt.Errorf("convert document embeddings: %w", err)
	}

	if err := collection.Add(
		ctx,
		chroma.WithIDs(ids...),
		chroma.WithTexts(texts...),
		chroma.WithMetadatas(metadatas...),
		chroma.WithEmbeddings(embeddings...),
	); err != nil {
		return fmt.Errorf("add documents to chroma: %w", err)
	}

	return nil
}

func (s *Store) SimilaritySearch(ctx context.Context, kbNamespace, query string, numDocuments int, scoreThreshold float32) ([]schema.Document, error) {
	collection, err := s.getCollection(ctx)
	if err != nil {
		return nil, err
	}

	queryEmbedding, err := s.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	result, err := collection.Query(
		ctx,
		chroma.WithQueryEmbeddings(chromaemb.NewEmbeddingFromFloat32(queryEmbedding)),
		chroma.WithNResults(numDocuments),
		chroma.WithWhereQuery(chroma.EqString(namespaceMetadataKey, kbNamespace)),
		chroma.WithIncludeQuery(chroma.IncludeDocuments, chroma.IncludeMetadatas, chroma.Include("distances")),
	)
	if err != nil {
		return nil, fmt.Errorf("chroma similarity search: %w", err)
	}

	return queryResultToDocuments(result, scoreThreshold)
}

func (s *Store) DeleteDocument(ctx context.Context, kbNamespace, documentID string) error {
	collection, err := s.getCollection(ctx)
	if err != nil {
		return err
	}

	if err := collection.Delete(
		ctx,
		chroma.WithWhereDelete(chroma.And(
			chroma.EqString(namespaceMetadataKey, kbNamespace),
			chroma.EqString("document_id", documentID),
		)),
	); err != nil {
		return fmt.Errorf("delete document from chroma: %w", err)
	}

	return nil
}

func (s *Store) DeleteNamespace(ctx context.Context, kbNamespace string) error {
	collection, err := s.getCollection(ctx)
	if err != nil {
		return err
	}

	if err := collection.Delete(
		ctx,
		chroma.WithWhereDelete(chroma.EqString(namespaceMetadataKey, kbNamespace)),
	); err != nil {
		return fmt.Errorf("delete namespace from chroma: %w", err)
	}

	return nil
}

func (s *Store) getCollection(ctx context.Context) (chroma.Collection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collection != nil {
		return s.collection, nil
	}

	collection, err := s.client.GetOrCreateCollection(
		ctx,
		s.collectionName,
		chroma.WithEmbeddingFunctionCreate(chromaEmbedderAdapter{embedder: s.embedder}),
		chroma.WithHNSWSpaceCreate(chromaemb.COSINE),
		chroma.WithIfNotExistsCreate(),
	)
	if err != nil {
		return nil, fmt.Errorf("get or create chroma collection: %w", err)
	}

	s.collection = collection
	return collection, nil
}

func buildMetadata(source map[string]any, kbNamespace string) (chroma.DocumentMetadata, error) {
	metadataMap := make(map[string]any, len(source)+1)
	for key, value := range source {
		metadataMap[key] = value
	}
	metadataMap[namespaceMetadataKey] = kbNamespace

	metadata, err := chroma.NewDocumentMetadataFromMap(metadataMap)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

func chunkID(kbNamespace string, doc schema.Document, fallbackIndex int) string {
	documentID, _ := doc.Metadata["document_id"].(string)
	chunkIndex := metadataInt(doc.Metadata, "chunk_index", fallbackIndex)

	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%s", kbNamespace, documentID, chunkIndex, doc.PageContent)))
	return hex.EncodeToString(sum[:])
}

func metadataInt(metadata map[string]any, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}

	switch value := metadata[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	default:
		return fallback
	}
}

func queryResultToDocuments(result chroma.QueryResult, scoreThreshold float32) ([]schema.Document, error) {
	documentGroups := result.GetDocumentsGroups()
	metadataGroups := result.GetMetadatasGroups()
	distanceGroups := result.GetDistancesGroups()

	if len(documentGroups) == 0 {
		return nil, nil
	}

	documents := make([]schema.Document, 0, len(documentGroups[0]))
	for idx, document := range documentGroups[0] {
		pageContent := document.ContentString()
		metadata, err := metadataToMap(metadataGroups, idx)
		if err != nil {
			return nil, err
		}

		score := float32(0)
		if len(distanceGroups) > 0 && len(distanceGroups[0]) > idx {
			score = 1 - float32(distanceGroups[0][idx])
			if score < 0 {
				score = 0
			}
		}
		if scoreThreshold > 0 && score < scoreThreshold {
			continue
		}

		documents = append(documents, schema.Document{
			PageContent: pageContent,
			Metadata:    metadata,
			Score:       score,
		})
	}

	return documents, nil
}

func metadataToMap(groups []chroma.DocumentMetadatas, idx int) (map[string]any, error) {
	if len(groups) == 0 || len(groups[0]) <= idx || groups[0][idx] == nil {
		return map[string]any{}, nil
	}

	raw, err := json.Marshal(groups[0][idx])
	if err != nil {
		return nil, fmt.Errorf("marshal chroma metadata: %w", err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal chroma metadata: %w", err)
	}
	return metadata, nil
}

type chromaEmbedderAdapter struct {
	embedder langchainembeddings.Embedder
}

func (a chromaEmbedderAdapter) EmbedDocuments(ctx context.Context, texts []string) ([]chromaemb.Embedding, error) {
	vectors, err := a.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, err
	}
	return chromaemb.NewEmbeddingsFromFloat32(vectors)
}

func (a chromaEmbedderAdapter) EmbedQuery(ctx context.Context, text string) (chromaemb.Embedding, error) {
	vector, err := a.embedder.EmbedQuery(ctx, text)
	if err != nil {
		return nil, err
	}
	return chromaemb.NewEmbeddingFromFloat32(vector), nil
}

// CheckHealth verifies that the configured Chroma endpoint responds to a heartbeat request.
func CheckHealth(ctx context.Context, chromaURL string, httpClient *http.Client) error {
	baseURL := strings.TrimRight(chromaURL, "/")
	paths := []string{"/api/v2/heartbeat", "/api/v1/heartbeat"}

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
