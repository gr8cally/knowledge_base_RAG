package rag

import (
	"fmt"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/schema"
)

func BuildCitations(docs []schema.Document) []domain.Citation {
	now := time.Now().UTC()
	citations := make([]domain.Citation, 0, len(docs))
	seen := make(map[string]struct{}, len(docs))

	for _, doc := range docs {
		documentID, _ := doc.Metadata["document_id"].(string)
		sourceLabel, _ := doc.Metadata["source_label"].(string)
		chunkIndex := metadataInt(doc.Metadata, "chunk_index")
		key := fmt.Sprintf("%s:%d", documentID, chunkIndex)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		citations = append(citations, domain.Citation{
			ID:            uuid.NewString(),
			CitationIndex: len(citations) + 1,
			DocumentID:    documentID,
			SourceLabel:   sourceLabel,
			Excerpt:       excerpt(doc.PageContent),
			ChunkIndex:    chunkIndex,
			Score:         float64(doc.Score),
			CreatedAt:     now,
		})
	}

	return citations
}

func AppendCitationMarkers(answer string, citations []domain.Citation) string {
	answer = strings.TrimSpace(answer)
	if len(citations) == 0 {
		if answer == "" {
			return "I don't have enough evidence in the selected knowledge base to answer that."
		}
		return answer
	}

	markers := make([]string, 0, len(citations))
	for _, citation := range citations {
		markers = append(markers, fmt.Sprintf("[%d]", citation.CitationIndex))
	}

	if answer == "" {
		answer = "I found relevant evidence in the selected knowledge base."
	}

	return answer + "\n\nSources: " + strings.Join(markers, " ")
}

func metadataInt(metadata map[string]any, key string) int {
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
		return 0
	}
}

func excerpt(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= 240 {
		return content
	}
	return strings.TrimSpace(content[:240]) + "..."
}
