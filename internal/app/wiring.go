package app

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"knowledge_base_RAG/internal/config"
	"knowledge_base_RAG/internal/embeddings"
	"knowledge_base_RAG/internal/ingest"
	"knowledge_base_RAG/internal/storage/filestore"
	"knowledge_base_RAG/internal/storage/sqlite"
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

func (d *Dependencies) NewVectorStore(embedder langchainembeddings.Embedder) (*vector.Store, error) {
	return vector.NewStore(d.Config.ChromaURL, d.Config.ChromaCollection, embedder, d.InferenceHTTPClient)
}

func (d *Dependencies) NewKnowledgeBaseService() *KnowledgeBaseService {
	return NewKnowledgeBaseService(sqlite.NewKnowledgeBaseRepo(d.Config.SQLitePath))
}

func (d *Dependencies) NewDocumentService(kbService *KnowledgeBaseService, embedder langchainembeddings.Embedder) (*ingest.Service, error) {
	fileStore := filestore.New(filepath.Join(d.Config.DataDir, "files"))
	vectorStore, err := d.NewVectorStore(embedder)
	if err != nil {
		return nil, err
	}
	worker := ingest.NewWorker(d.Logger, d.Config.ChunkSize, d.Config.ChunkOverlap, d.Config.OCREnabled, d.Config.OCRLang, vectorStore)
	return ingest.NewService(
		d.Logger,
		kbService,
		sqlite.NewDocumentRepo(d.Config.SQLitePath),
		sqlite.NewIngestRepo(d.Config.SQLitePath),
		fileStore,
		worker,
		ingest.NewJobBroker(),
		d.Config.IngestWorkers,
	), nil
}

func (d *Dependencies) NewConversationService(kbService *KnowledgeBaseService) *ConversationService {
	return NewConversationService(
		kbService,
		sqlite.NewConversationRepo(d.Config.SQLitePath),
		sqlite.NewMessageRepo(d.Config.SQLitePath),
	)
}
