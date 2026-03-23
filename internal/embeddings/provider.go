package embeddings

import (
	"net/http"
	"os"

	"knowledge_base_RAG/internal/config"

	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	hfembeddings "github.com/tmc/langchaingo/embeddings/huggingface"
	hfllm "github.com/tmc/langchaingo/llms/huggingface"
)

// NewProvider returns the configured embedding provider.
func NewProvider(cfg config.Config, httpClient *http.Client) (langchainembeddings.Embedder, error) {
	token := os.Getenv("HF_TOKEN")
	if token == "" {
		token = os.Getenv("HUGGINGFACEHUB_API_TOKEN")
	}
	if token == "" {
		// Local inference servers typically ignore auth headers; langchaingo's client still requires a token value.
		token = "local"
	}

	client, err := hfllm.New(
		hfllm.WithToken(token),
		hfllm.WithModel(cfg.EmbeddingModelName),
		hfllm.WithURL(cfg.EmbeddingEndpoint),
		hfllm.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}

	return hfembeddings.NewHuggingface(
		hfembeddings.WithModel(cfg.EmbeddingModelName),
		hfembeddings.WithClient(*client),
	)
}
