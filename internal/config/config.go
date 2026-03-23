package config

import (
	"fmt"
)

// Config stores runtime configuration loaded from environment variables.
type Config struct {
	AppEnv              string
	HTTPAddr            string
	DataDir             string
	SQLitePath          string
	OpenRouterAPIKey    string
	ModelName           string
	OpenRouterBaseURL   string
	EmbeddingModelName  string
	EmbeddingEndpoint   string
	ChromaURL           string
	RAGTopK             int
	RAGScoreThreshold   float64
	ChunkSize           int
	ChunkOverlap        int
	ChatHistoryMaxTurns int
	MaxUploadMB         int
	IngestWorkers       int
	EnableURLIngest     bool
	OCREnabled          bool
	OCRLang             string
}

// Load reads configuration from env and validates required keys.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		DataDir:             getEnv("DATA_DIR", "./data"),
		SQLitePath:          getEnv("SQLITE_PATH", "./data/sqlite/app.db"),
		OpenRouterAPIKey:    getEnv("OPENROUTER_API_KEY", ""),
		ModelName:           getEnv("MODEL_NAME", ""),
		OpenRouterBaseURL:   getEnv("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		EmbeddingModelName:  getEnv("EMBEDDING_MODEL_NAME", "sentence-transformers/all-MiniLM-L6-v2"),
		EmbeddingEndpoint:   getEnv("EMBEDDING_ENDPOINT", "http://localhost:8081"),
		ChromaURL:           getEnv("CHROMA_URL", "http://localhost:8000"),
		RAGTopK:             getEnvInt("RAG_TOP_K", 6),
		RAGScoreThreshold:   getEnvFloat64("RAG_SCORE_THRESHOLD", 0.2),
		ChunkSize:           getEnvInt("CHUNK_SIZE", 800),
		ChunkOverlap:        getEnvInt("CHUNK_OVERLAP", 120),
		ChatHistoryMaxTurns: getEnvInt("CHAT_HISTORY_MAX_TURNS", 12),
		MaxUploadMB:         getEnvInt("MAX_UPLOAD_MB", 50),
		IngestWorkers:       getEnvInt("INGEST_WORKERS", 2),
		EnableURLIngest:     getEnvBool("ENABLE_URL_INGEST", true),
		OCREnabled:          getEnvBool("OCR_ENABLED", true),
		OCRLang:             getEnv("OCR_LANG", "eng"),
	}

	if cfg.OpenRouterAPIKey == "" {
		return Config{}, fmt.Errorf("OPENROUTER_API_KEY is required")
	}
	if cfg.ModelName == "" {
		return Config{}, fmt.Errorf("MODEL_NAME is required")
	}

	return cfg, nil
}
