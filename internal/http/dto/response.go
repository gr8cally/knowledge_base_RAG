package dto

import "knowledge_base_RAG/internal/domain"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type KnowledgeBaseResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Namespace   string  `json:"namespace"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	ArchivedAt  *string `json:"archived_at,omitempty"`
}

func NewKnowledgeBaseResponse(kb domain.KnowledgeBase) KnowledgeBaseResponse {
	resp := KnowledgeBaseResponse{
		ID:          kb.ID,
		Name:        kb.Name,
		Description: kb.Description,
		Namespace:   kb.Namespace,
		CreatedAt:   kb.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   kb.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if kb.ArchivedAt != nil {
		v := kb.ArchivedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ArchivedAt = &v
	}
	return resp
}
