package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"knowledge_base_RAG/internal/domain"
	"knowledge_base_RAG/internal/storage/filestore"

	"github.com/google/uuid"
)

var ErrKnowledgeBaseNotFound = errors.New("knowledge base not found")

type KnowledgeBaseGetter interface {
	Get(ctx context.Context, id string) (*domain.KnowledgeBase, error)
}

type DocumentRepository interface {
	Create(ctx context.Context, doc domain.Document) error
	Update(ctx context.Context, doc domain.Document) error
	ListByKB(ctx context.Context, kbID string) ([]domain.Document, error)
	FindByKBAndNormalizedName(ctx context.Context, kbID, normalizedName string) (*domain.Document, error)
}

type IngestionJobRepository interface {
	Create(ctx context.Context, job domain.IngestionJob) error
	Update(ctx context.Context, job domain.IngestionJob) error
}

type UploadResult struct {
	Document domain.Document      `json:"document"`
	Job      *domain.IngestionJob `json:"job,omitempty"`
	Skipped  bool                 `json:"skipped"`
	Notice   string               `json:"notice,omitempty"`
}

type Service struct {
	logger    *slog.Logger
	kbGetter  KnowledgeBaseGetter
	docRepo   DocumentRepository
	jobRepo   IngestionJobRepository
	fileStore *filestore.Store
	worker    *Worker
}

func NewService(logger *slog.Logger, kbGetter KnowledgeBaseGetter, docRepo DocumentRepository, jobRepo IngestionJobRepository, fileStore *filestore.Store, worker *Worker) *Service {
	return &Service{
		logger:    logger,
		kbGetter:  kbGetter,
		docRepo:   docRepo,
		jobRepo:   jobRepo,
		fileStore: fileStore,
		worker:    worker,
	}
}

func (s *Service) ListDocuments(ctx context.Context, kbID string) ([]domain.Document, error) {
	kb, err := s.kbGetter.Get(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, ErrKnowledgeBaseNotFound
	}
	return s.docRepo.ListByKB(ctx, kbID)
}

func (s *Service) UploadFile(ctx context.Context, kbID string, header *multipart.FileHeader) (UploadResult, error) {
	kb, err := s.kbGetter.Get(ctx, kbID)
	if err != nil {
		return UploadResult{}, err
	}
	if kb == nil {
		return UploadResult{}, ErrKnowledgeBaseNotFound
	}

	src, err := header.Open()
	if err != nil {
		return UploadResult{}, fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()

	tempPath, sha, size, err := CopyMultipartToTempAndHash(src)
	if err != nil {
		return UploadResult{}, err
	}

	normalizedName := normalizeName(header.Filename)
	existing, err := s.docRepo.FindByKBAndNormalizedName(ctx, kbID, normalizedName)
	if err != nil {
		return UploadResult{}, err
	}
	if existing != nil && existing.SHA256 == sha {
		_ = s.fileStore.Remove(tempPath)
		now := time.Now().UTC()
		job := domain.IngestionJob{
			ID:             uuid.NewString(),
			KBID:           kbID,
			TriggerType:    "upload",
			Status:         "completed",
			TotalItems:     1,
			ProcessedItems: 0,
			SkippedItems:   1,
			FailedItems:    0,
			ErrorMessage:   "",
			CreatedAt:      now,
			StartedAt:      &now,
			FinishedAt:     &now,
		}
		if err := s.jobRepo.Create(ctx, job); err != nil {
			return UploadResult{}, err
		}
		return UploadResult{
			Document: *existing,
			Job:      &job,
			Skipped:  true,
			Notice:   "same file hash, upload skipped",
		}, nil
	}

	now := time.Now().UTC()
	doc := domain.Document{
		ID:             uuid.NewString(),
		KBID:           kbID,
		SourceType:     "file",
		DisplayName:    header.Filename,
		NormalizedName: normalizedName,
		SourceURI:      header.Filename,
		SHA256:         sha,
		StoragePath:    "",
		MimeType:       detectMimeType(header),
		SizeBytes:      size,
		ParserUsed:     "",
		ChunkCount:     0,
		Status:         "processing",
		ErrorMessage:   "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	oldStoragePath := ""
	if existing != nil {
		doc.ID = existing.ID
		doc.CreatedAt = existing.CreatedAt
		oldStoragePath = existing.StoragePath
	}

	doc.StoragePath = s.fileStore.FinalPath(kbID, doc.ID, sha, header.Filename)

	job := domain.IngestionJob{
		ID:             uuid.NewString(),
		KBID:           kbID,
		TriggerType:    "upload",
		Status:         "queued",
		TotalItems:     1,
		ProcessedItems: 0,
		SkippedItems:   0,
		FailedItems:    0,
		ErrorMessage:   "",
		CreatedAt:      now,
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		_ = s.fileStore.Remove(tempPath)
		return UploadResult{}, err
	}

	if err := s.fileStore.Move(tempPath, doc.StoragePath); err != nil {
		return UploadResult{}, err
	}

	if existing == nil {
		if err := s.docRepo.Create(ctx, doc); err != nil {
			return UploadResult{}, err
		}
	} else {
		if err := s.fileStore.Remove(oldStoragePath); err != nil {
			return UploadResult{}, err
		}
		if err := s.docRepo.Update(ctx, doc); err != nil {
			return UploadResult{}, err
		}
	}

	started := time.Now().UTC()
	job.Status = "running"
	job.StartedAt = &started
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return UploadResult{}, err
	}

	if err := s.worker.Process(ctx, &doc); err != nil {
		doc.Status = "error"
		doc.ErrorMessage = err.Error()
		doc.UpdatedAt = time.Now().UTC()
		_ = s.docRepo.Update(ctx, doc)

		finished := time.Now().UTC()
		job.Status = "failed"
		job.FailedItems = 1
		job.ErrorMessage = err.Error()
		job.FinishedAt = &finished
		_ = s.jobRepo.Update(ctx, job)
		return UploadResult{Document: doc, Job: &job}, err
	}

	doc.UpdatedAt = time.Now().UTC()
	if err := s.docRepo.Update(ctx, doc); err != nil {
		return UploadResult{}, err
	}

	finished := time.Now().UTC()
	job.Status = "completed"
	job.ProcessedItems = 1
	job.FinishedAt = &finished
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return UploadResult{}, err
	}

	s.logger.Info("document_ingested_dry_run",
		"kb_id", kb.ID,
		"document_id", doc.ID,
		"display_name", doc.DisplayName,
		"chunk_count", doc.ChunkCount,
	)

	return UploadResult{Document: doc, Job: &job}, nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(filepath.Base(name)))
}

func detectMimeType(header *multipart.FileHeader) string {
	if header.Header != nil {
		if v := header.Header.Get("Content-Type"); v != "" {
			return v
		}
	}
	return "application/octet-stream"
}
