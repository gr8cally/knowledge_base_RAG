package dto

type CreateKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
