CREATE TABLE message_citations (
  id TEXT PRIMARY KEY,
  message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  citation_index INTEGER NOT NULL,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  source_label TEXT NOT NULL,
  excerpt TEXT NOT NULL,
  chunk_index INTEGER NOT NULL DEFAULT 0,
  score REAL NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  UNIQUE(message_id, citation_index)
);

CREATE INDEX idx_citations_message_id ON message_citations(message_id);
