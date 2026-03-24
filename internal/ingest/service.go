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

var (
	ErrKnowledgeBaseNotFound = errors.New("knowledge base not found")
	ErrIngestionJobNotFound  = errors.New("ingestion job not found")
)

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
	ListByKB(ctx context.Context, kbID string) ([]domain.IngestionJob, error)
	GetByID(ctx context.Context, kbID, jobID string) (*domain.IngestionJob, error)
}

type UploadResult struct {
	Document domain.Document      `json:"document"`
	Job      *domain.IngestionJob `json:"job,omitempty"`
	Skipped  bool                 `json:"skipped"`
	Notice   string               `json:"notice,omitempty"`
}

type Service struct {
	logger        *slog.Logger
	kbGetter      KnowledgeBaseGetter
	docRepo       DocumentRepository
	jobRepo       IngestionJobRepository
	fileStore     *filestore.Store
	worker        *Worker
	broker        *JobBroker
	queue         chan JobTask
	workerCount   int
	startedWorker bool
}

func NewService(logger *slog.Logger, kbGetter KnowledgeBaseGetter, docRepo DocumentRepository, jobRepo IngestionJobRepository, fileStore *filestore.Store, worker *Worker, broker *JobBroker, workerCount int) *Service {
	if workerCount <= 0 {
		workerCount = 1
	}
	return &Service{
		logger:      logger,
		kbGetter:    kbGetter,
		docRepo:     docRepo,
		jobRepo:     jobRepo,
		fileStore:   fileStore,
		worker:      worker,
		broker:      broker,
		queue:       make(chan JobTask, max(workerCount*4, 8)),
		workerCount: workerCount,
	}
}

func (s *Service) Start(ctx context.Context) {
	if s.startedWorker {
		return
	}
	s.startedWorker = true

	for i := 0; i < s.workerCount; i++ {
		go s.runWorker(ctx, i+1)
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

func (s *Service) ListJobs(ctx context.Context, kbID string) ([]domain.IngestionJob, error) {
	kb, err := s.kbGetter.Get(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, ErrKnowledgeBaseNotFound
	}
	return s.jobRepo.ListByKB(ctx, kbID)
}

func (s *Service) GetJob(ctx context.Context, kbID, jobID string) (*domain.IngestionJob, error) {
	kb, err := s.kbGetter.Get(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, ErrKnowledgeBaseNotFound
	}

	job, err := s.jobRepo.GetByID(ctx, kbID, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrIngestionJobNotFound
	}
	return job, nil
}

func (s *Service) SubscribeJob(ctx context.Context, kbID, jobID string) (*domain.IngestionJob, <-chan JobEvent, func(), error) {
	job, err := s.GetJob(ctx, kbID, jobID)
	if err != nil {
		return nil, nil, nil, err
	}

	events, cancel := s.broker.Subscribe(jobID)
	return job, events, cancel, nil
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

	tempPath, sha, size, err := CopyMultipartToTempAndHash(src, filepath.Join(s.fileStore.RootDir(), ".tmp"))
	if err != nil {
		return UploadResult{}, err
	}
	tempMoved := false
	defer func() {
		if !tempMoved {
			_ = s.fileStore.Remove(tempPath)
		}
	}()

	normalizedName := normalizeName(header.Filename)
	existing, err := s.docRepo.FindByKBAndNormalizedName(ctx, kbID, normalizedName)
	if err != nil {
		return UploadResult{}, err
	}
	if existing != nil && existing.SHA256 == sha {
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
		s.publish(job, *existing, "completed", "same file hash, upload skipped")
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
		return UploadResult{}, err
	}

	if err := s.fileStore.Move(tempPath, doc.StoragePath); err != nil {
		return UploadResult{}, err
	}
	tempMoved = true

	if existing == nil {
		if err := s.docRepo.Create(ctx, doc); err != nil {
			return UploadResult{}, err
		}
	} else {
		if err := s.docRepo.Update(ctx, doc); err != nil {
			return UploadResult{}, err
		}
		if oldStoragePath != "" {
			_ = s.fileStore.Remove(oldStoragePath)
		}
	}

	task := JobTask{Job: job, Document: doc, KBNamespace: kb.Namespace}
	s.queue <- task
	s.publish(job, doc, "queued", "upload accepted")
	if existing != nil {
		s.logger.Warn("document_replace_vector_cleanup_deferred",
			"kb_id", kbID,
			"document_id", doc.ID,
			"display_name", doc.DisplayName,
		)
	}

	return UploadResult{Document: doc, Job: &job}, nil
}

func (s *Service) runWorker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-s.queue:
			s.processTask(ctx, workerID, task)
		}
	}
}

func (s *Service) processTask(ctx context.Context, workerID int, task JobTask) {
	doc := task.Document
	job := task.Job

	started := time.Now().UTC()
	job.Status = "running"
	job.StartedAt = &started
	job.ErrorMessage = ""
	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Error("ingest_job_update_failed", "job_id", job.ID, "worker_id", workerID, "error", err)
		return
	}
	s.publish(job, doc, "running", "ingestion started")

	// load doc, chunk, index, upsert
	processedDoc, err := s.worker.Process(ctx, task)
	if err != nil {
		doc.Status = "error"
		doc.ErrorMessage = err.Error()
		doc.UpdatedAt = time.Now().UTC()
		if updateErr := s.docRepo.Update(ctx, doc); updateErr != nil {
			s.logger.Error("document_update_failed", "job_id", job.ID, "document_id", doc.ID, "worker_id", workerID, "error", updateErr)
		}

		finished := time.Now().UTC()
		job.Status = "failed"
		job.FailedItems = 1
		job.ErrorMessage = err.Error()
		job.FinishedAt = &finished
		if updateErr := s.jobRepo.Update(ctx, job); updateErr != nil {
			s.logger.Error("ingest_job_update_failed", "job_id", job.ID, "worker_id", workerID, "error", updateErr)
		}
		s.publish(job, doc, "failed", err.Error())
		return
	}

	processedDoc.UpdatedAt = time.Now().UTC()
	if err := s.docRepo.Update(ctx, processedDoc); err != nil {
		s.logger.Error("document_update_failed", "job_id", job.ID, "document_id", processedDoc.ID, "worker_id", workerID, "error", err)
		finished := time.Now().UTC()
		job.Status = "failed"
		job.FailedItems = 1
		job.ErrorMessage = err.Error()
		job.FinishedAt = &finished
		_ = s.jobRepo.Update(ctx, job)
		s.publish(job, processedDoc, "failed", err.Error())
		return
	}

	finished := time.Now().UTC()
	job.Status = "completed"
	job.ProcessedItems = 1
	job.FinishedAt = &finished
	if err := s.jobRepo.Update(ctx, job); err != nil {
		s.logger.Error("ingest_job_update_failed", "job_id", job.ID, "worker_id", workerID, "error", err)
		return
	}

	s.logger.Info("document_ingested",
		"kb_id", processedDoc.KBID,
		"document_id", processedDoc.ID,
		"display_name", processedDoc.DisplayName,
		"chunk_count", processedDoc.ChunkCount,
		"job_id", job.ID,
		"worker_id", workerID,
	)
	s.publish(job, processedDoc, "completed", "ingestion completed")
}

func (s *Service) publish(job domain.IngestionJob, doc domain.Document, eventType, message string) {
	s.broker.Publish(JobEvent{
		Type:           eventType,
		Job:            job,
		DocumentID:     doc.ID,
		DocumentStatus: doc.Status,
		Message:        message,
		At:             time.Now().UTC(),
	})
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
