package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"knowledge_base_RAG/internal/config"
	"knowledge_base_RAG/internal/embeddings"
	"knowledge_base_RAG/internal/vector"

	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

type Dependencies struct {
	Config              config.Config
	Logger              *slog.Logger
	InferenceHTTPClient *http.Client
	HealthHTTPClient    *http.Client
}

func NewDependencies(cfg config.Config, logger *slog.Logger) *Dependencies {
	return &Dependencies{
		Config: cfg,
		Logger: logger,
		InferenceHTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		HealthHTTPClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (d *Dependencies) NewLLM() (*openai.LLM, error) {
	return openai.New(
		openai.WithBaseURL(d.Config.OpenRouterBaseURL),
		openai.WithToken(d.Config.OpenRouterAPIKey),
		openai.WithModel(d.Config.ModelName),
		openai.WithHTTPClient(d.InferenceHTTPClient),
	)
}

func (d *Dependencies) NewEmbedder() (langchainembeddings.Embedder, error) {
	return embeddings.NewProvider(d.Config, d.InferenceHTTPClient)
}

func (d *Dependencies) CheckChroma(ctx context.Context) error {
	return vector.CheckHealth(ctx, d.Config.ChromaURL, d.HealthHTTPClient)
}
