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
	Config     config.Config
	Logger     *slog.Logger
	HTTPClient *http.Client
}

func NewDependencies(cfg config.Config, logger *slog.Logger) *Dependencies {
	return &Dependencies{
		Config: cfg,
		Logger: logger,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *Dependencies) NewLLM() (*openai.LLM, error) {
	return openai.New(
		openai.WithBaseURL(d.Config.OpenRouterBaseURL),
		openai.WithToken(d.Config.OpenRouterAPIKey),
		openai.WithModel(d.Config.ModelName),
		openai.WithHTTPClient(d.HTTPClient),
	)
}

func (d *Dependencies) NewEmbedder() (langchainembeddings.Embedder, error) {
	return embeddings.NewProvider(d.Config, d.HTTPClient)
}

func (d *Dependencies) CheckChroma(ctx context.Context) error {
	return vector.CheckHealth(ctx, d.Config.ChromaURL, d.HTTPClient)
}
