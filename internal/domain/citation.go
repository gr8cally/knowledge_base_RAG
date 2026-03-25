package domain

import "time"

type Citation struct {
	ID            string    `json:"id"`
	MessageID     string    `json:"message_id"`
	CitationIndex int       `json:"citation_index"`
	DocumentID    string    `json:"document_id"`
	SourceLabel   string    `json:"source_label"`
	Excerpt       string    `json:"excerpt"`
	ChunkIndex    int       `json:"chunk_index"`
	Score         float64   `json:"score"`
	CreatedAt     time.Time `json:"created_at"`
}
