package domain

import "time"

type Document struct {
	ID             string    `json:"id"`
	KBID           string    `json:"kb_id"`
	SourceType     string    `json:"source_type"`
	DisplayName    string    `json:"display_name"`
	NormalizedName string    `json:"normalized_name"`
	SourceURI      string    `json:"source_uri"`
	SHA256         string    `json:"sha256"`
	StoragePath    string    `json:"storage_path"`
	MimeType       string    `json:"mime_type"`
	SizeBytes      int64     `json:"size_bytes"`
	ParserUsed     string    `json:"parser_used"`
	ChunkCount     int       `json:"chunk_count"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
