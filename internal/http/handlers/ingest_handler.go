package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"knowledge_base_RAG/internal/domain"
	"knowledge_base_RAG/internal/ingest"
)

type ingestionJobService interface {
	ListJobs(ctx context.Context, kbID string) ([]domain.IngestionJob, error)
	GetJob(ctx context.Context, kbID, jobID string) (*domain.IngestionJob, error)
	SubscribeJob(ctx context.Context, kbID, jobID string) (*domain.IngestionJob, <-chan ingest.JobEvent, func(), error)
}

type IngestHandler struct {
	service ingestionJobService
}

func NewIngestHandler(service ingestionJobService) *IngestHandler {
	return &IngestHandler{service: service}
}

func (h *IngestHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.ListJobs(r.Context(), r.PathValue("kbID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "list_jobs_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (h *IngestHandler) GetAPI(w http.ResponseWriter, r *http.Request) {
	job, err := h.service.GetJob(r.Context(), r.PathValue("kbID"), r.PathValue("jobID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound), errors.Is(err, ingest.ErrIngestionJobNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "get_job_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *IngestHandler) Events(w http.ResponseWriter, r *http.Request) {
	job, events, cancel, err := h.service.SubscribeJob(r.Context(), r.PathValue("kbID"), r.PathValue("jobID"))
	if err != nil {
		switch {
		case errors.Is(err, ingest.ErrKnowledgeBaseNotFound), errors.Is(err, ingest.ErrIngestionJobNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "subscribe_job_failed", err.Error())
		}
		return
	}
	defer cancel()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "sse_not_supported", "response writer does not support streaming")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if err := writeSSE(w, ingest.JobEvent{
		Type: "snapshot",
		Job:  *job,
		At:   job.CreatedAt,
	}); err != nil {
		return
	}
	flusher.Flush()

	if job.Status == "completed" || job.Status == "failed" {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeSSE(w, event); err != nil {
				return
			}
			flusher.Flush()
			if event.Job.Status == "completed" || event.Job.Status == "failed" {
				return
			}
		}
	}
}

func writeSSE(w http.ResponseWriter, event ingest.JobEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: job\ndata: %s\n\n", payload)
	return err
}
