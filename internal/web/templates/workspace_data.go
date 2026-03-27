package templates

import "knowledge_base_RAG/internal/domain"

type WorkspacePageData struct {
	KnowledgeBases       []domain.KnowledgeBase
	ActiveKBID           string
	ActiveKB             *domain.KnowledgeBase
	Conversations        []domain.Conversation
	ActiveConversationID string
}
