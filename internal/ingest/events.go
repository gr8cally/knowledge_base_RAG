package ingest

import (
	"sync"
	"time"

	"knowledge_base_RAG/internal/domain"
)

type JobEvent struct {
	Type           string              `json:"type"`
	Job            domain.IngestionJob `json:"job"`
	DocumentID     string              `json:"document_id,omitempty"`
	DocumentStatus string              `json:"document_status,omitempty"`
	Message        string              `json:"message,omitempty"`
	At             time.Time           `json:"at"`
}

type JobBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan JobEvent]struct{}
}

func NewJobBroker() *JobBroker {
	return &JobBroker{
		subscribers: make(map[string]map[chan JobEvent]struct{}),
	}
}

func (b *JobBroker) Publish(event JobEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[event.Job.ID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *JobBroker) Subscribe(jobID string) (<-chan JobEvent, func()) {
	ch := make(chan JobEvent, 8)

	b.mu.Lock()
	if b.subscribers[jobID] == nil {
		b.subscribers[jobID] = make(map[chan JobEvent]struct{})
	}
	b.subscribers[jobID][ch] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		subscribers := b.subscribers[jobID]
		if subscribers == nil {
			return
		}
		delete(subscribers, ch)
		close(ch)
		if len(subscribers) == 0 {
			delete(b.subscribers, jobID)
		}
	}

	return ch, cancel
}
