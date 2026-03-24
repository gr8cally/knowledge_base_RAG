package domain

import "time"

type Conversation struct {
	ID            string     `json:"id"`
	KBID          string     `json:"kb_id"`
	Title         string     `json:"title"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastMessageAt *time.Time `json:"last_message_at"`
	ArchivedAt    *time.Time `json:"archived_at,omitempty"`
}
