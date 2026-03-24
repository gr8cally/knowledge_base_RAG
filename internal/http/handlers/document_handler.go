package handlers

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"

	"knowledge_base_RAG/internal/domain"
	"knowledge_base_RAG/internal/ingest"
)

type documentService interface {
	ListDocuments(ctx context.Context, kbID string) ([]domain.Document, error)
	UploadFile(ctx context.Context, kbID string, header *multipart.FileHeader) (ingest.UploadResult, error)
	RefreshDocument(ctx context.Context, kbID, documentID string) (ingest.UploadResult, error)
	DeleteDocument(ctx context.Context, kbID, documentID string) error
	ReindexAll(ctx context.Context, kbID string) (*domain.IngestionJob, error)
}

type DocumentHandler struct {
	service     documentService
	maxUploadMB int
}

func NewDocumentHandler(service documentService, maxUploadMB int) *DocumentHandler {
	return &DocumentHandler{
		service:     service,
		maxUploadMB: maxUploadMB,
	}
}

func (h *DocumentHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	docs, err := h.service.ListDocuments(r.Context(), r.PathValue("kbID"))
	if err != nil {
		if errors.Is(err, ingest.ErrKnowledgeBaseNotFound) {
			http.NotFound(w, r)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "list_documents_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func (h *DocumentHandler) UploadAPI(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(h.maxUploadMB)<<20)
	if err := r.ParseMultipartForm(int64(h.maxUploadMB) << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_multipart", err.Error())
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeAPIError(w, http.StatusBadRequest, "missing_files", "no files uploaded")
		return
	}

	results := make([]ingest.UploadResult, 0, len(files))
	accepted := false
	for _, header := range files {
		result, err := h.service.UploadFile(r.Context(), r.PathValue("kbID"), header)
		if err != nil {
			if errors.Is(err, ingest.ErrKnowledgeBaseNotFound) {
				http.NotFound(w, r)
				return
			}
			if errors.Is(err, ingest.ErrIngestionQueueFull) {
				writeAPIError(w, http.StatusServiceUnavailable, "ingestion_queue_full", err.Error())
				return
			}
			writeAPIError(w, http.StatusBadRequest, "upload_failed", err.Error())
			return
		}
		if !result.Skipped {
			accepted = true
		}
		results = append(results, result)
	}

	status := http.StatusOK
	if accepted {
		status = http.StatusAccepted
	}
	writeJSON(w, status, results)
}

func (h *DocumentHandler) RefreshAPI(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.RefreshDocument(r.Context(), r.PathValue("kbID"), r.PathValue("documentID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound), errors.Is(err, ingest.ErrDocumentNotFound):
			http.NotFound(w, r)
		case errors.Is(err, ingest.ErrIngestionQueueFull):
			writeAPIError(w, http.StatusServiceUnavailable, "ingestion_queue_full", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "refresh_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *DocumentHandler) DeleteAPI(w http.ResponseWriter, r *http.Request) {
	err := h.service.DeleteDocument(r.Context(), r.PathValue("kbID"), r.PathValue("documentID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound), errors.Is(err, ingest.ErrDocumentNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusBadRequest, "delete_failed", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DocumentHandler) ReindexAllAPI(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.ReindexAll(r.Context(), r.PathValue("kbID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound):
			http.NotFound(w, r)
		case errors.Is(err, ingest.ErrReindexInProgress):
			writeAPIError(w, http.StatusConflict, "reindex_in_progress", err.Error())
		default:
			writeAPIError(w, http.StatusBadRequest, "reindex_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
