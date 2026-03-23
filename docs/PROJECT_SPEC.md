# Personal Knowledge Base AI Agent

## Project Specification (Go + LangChainGo + Templ + HTMX)

## 0) Assessment of Proposed Feature Behavior

Your proposed behavior is strong and aligned with the assignment brief. I agree with almost all points, with a few implementation refinements:

| Proposed behavior | Decision | Notes |
|---|---|---|
| Multiple isolated knowledge bases | Agree | Core requirement. Isolation is enforced in SQLite (`kb_id`) and Chroma namespace (`vectorstores.WithNameSpace(kbNamespace)`). |
| View/delete/upload docs/URLs inside each KB | Agree | Required for lifecycle management and trust in the system state. |
| Conversation history per KB, create/resume conversations | Agree | Required for context continuity and "switch between previous conversations." |
| Same filename upload: hash compare, re-index only if changed | Agree | One logical document per filename. Same hash => skip; different hash => replace file/metadata and re-index. |
| Persist original uploaded files forever | Disagree | Hard-delete policy: deleting a doc removes vectors, DB row, and file from disk. |
| "Re-index all" from persisted files regardless of hash | Agree (with refinement) | Re-index active docs from each document's current stored file, ignoring hash checks. Use namespace generation switch to avoid serving partially rebuilt indices. |

### Additional recommendations

1. Keep the data model simple in v1: no document versioning.
2. Introduce ingestion jobs and per-item statuses for visibility and retry.
3. Use hard delete for documents (as requested), archive for conversations.
4. Add citation persistence (`message_citations`) so each answer is explainable later.
5. Implement OCR fallback for scanned PDFs/images; standard PDF text extraction alone is insufficient.
6. Guard re-index with per-KB locking to avoid race conditions with concurrent uploads.

### CLI status

No CLI in v1 (per your instruction). Add CLI as future TODO.

---

## 1) LangChainGo: Out-of-the-Box vs Custom Build

### LangChainGo out-of-the-box (use directly)

1. Document loading and splitting:
   - `documentloaders.NewPDF`, `NewText`, `NewHTML`, `NewCSV`, `NewRecursiveDirectory`
   - `LoadAndSplit(ctx, splitter)`
2. Text splitters:
   - Recursive/character-based splitters for chunking
3. Embedding abstraction:
   - `embeddings.Embedder` interface (plug-in model/provider)
4. Vector storage:
   - Chroma vector store integration
   - Namespace scoping (`vectorstores.WithNameSpace(...)`)
5. RAG retrieval chain:
   - `chains.NewConversationalRetrievalQA`
   - `ReturnSourceDocuments = true` for citations
6. LLM streaming:
   - Streaming interfaces and callback handlers
7. Memory primitives:
   - Conversation memory wrappers when paired with a custom persisted history backend

### Must be custom-built

1. Multi-tenant KB domain model (KBs, docs, conversations, jobs)
2. Hash-based dedupe/replace behavior tied to filename identity
3. Persistent file storage layout and file lifecycle rules
4. OCR path for scanned docs (and loader selection logic)
5. SQLite-backed chat history adapter and repository layer
6. Conversation/citation persistence and retrieval UI binding
7. Re-index orchestration with namespace generation switching
8. HTTP handlers, HTMX interactions, streaming endpoints, and Templ views
9. Ingestion worker queue, retries, progress reporting
10. Security controls (upload validation, URL fetch protections, path sanitization)

---

## 2) Full Project Directory Structure

```text
.
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── wiring.go
│   ├── config/
│   │   ├── config.go
│   │   └── env.go
│   ├── domain/
│   │   ├── kb.go
│   │   ├── document.go
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── citation.go
│   │   └── ingest_job.go
│   ├── storage/
│   │   ├── filestore/
│   │   │   ├── filestore.go
│   │   │   └── paths.go
│   │   └── sqlite/
│   │       ├── db.go
│   │       ├── tx.go
│   │       ├── kb_repo.go
│   │       ├── document_repo.go
│   │       ├── conversation_repo.go
│   │       ├── message_repo.go
│   │       ├── citation_repo.go
│   │       └── ingest_repo.go
│   ├── ingest/
│   │   ├── service.go
│   │   ├── worker.go
│   │   ├── loader_factory.go
│   │   ├── ocr.go
│   │   ├── hasher.go
│   │   └── chunker.go
│   ├── rag/
│   │   ├── chain.go
│   │   ├── retriever.go
│   │   ├── memory_sqlite.go
│   │   ├── citations.go
│   │   └── prompt.go
│   ├── llm/
│   │   ├── openrouter.go
│   │   └── streaming.go
│   ├── embeddings/
│   │   ├── provider.go
│   │   └── huggingface.go
│   ├── vector/
│   │   ├── chroma.go
│   │   └── namespace.go
│   ├── http/
│   │   ├── router.go
│   │   ├── middleware/
│   │   │   ├── recover.go
│   │   │   ├── logging.go
│   │   │   └── request_id.go
│   │   ├── handlers/
│   │   │   ├── kb_handler.go
│   │   │   ├── document_handler.go
│   │   │   ├── conversation_handler.go
│   │   │   ├── chat_handler.go
│   │   │   ├── ingest_handler.go
│   │   │   └── health_handler.go
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── web/
│   │   ├── templates/
│   │   │   ├── layout.templ
│   │   │   ├── kb_list.templ
│   │   │   ├── kb_detail.templ
│   │   │   ├── chat.templ
│   │   │   ├── components/
│   │   │   │   ├── upload_modal.templ
│   │   │   │   ├── citation_panel.templ
│   │   │   │   ├── conversation_list.templ
│   │   │   │   └── toasts.templ
│   │   │   └── generated/
│   │   └── static/
│   │       ├── css/app.css
│   │       ├── js/alpine.min.js
│   │       └── js/htmx.min.js
│   └── observability/
│       ├── logger.go
│       ├── metrics.go
│       └── tracing.go
├── migrations/
│   ├── 0001_init.sql
│   ├── 0002_citations.sql
│   └── 0003_reindex_generations.sql
├── data/
│   ├── sqlite/
│   └── files/
├── docs/
│   └── PROJECT_SPEC.md
├── .env.example
├── go.mod
└── README.md
```

---

## 3) SQLite Schema (All Tables)

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE knowledge_bases (
  id TEXT PRIMARY KEY,                     -- ULID/UUID
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  embedding_model TEXT NOT NULL,
  llm_model TEXT NOT NULL,
  index_generation INTEGER NOT NULL DEFAULT 1,
  active_namespace TEXT NOT NULL,          -- e.g. kb_<id>_g1
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  archived_at DATETIME
);

CREATE TABLE documents (
  id TEXT PRIMARY KEY,
  kb_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL CHECK (source_type IN ('file','url')),
  display_name TEXT NOT NULL,              -- logical identity in KB
  normalized_name TEXT NOT NULL,           -- lowercase/trimmed for uniqueness
  source_uri TEXT NOT NULL,                -- original URL or original filename
  sha256 TEXT NOT NULL,
  storage_path TEXT NOT NULL,              -- current file on disk
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  parser_used TEXT DEFAULT '',
  chunk_count INTEGER NOT NULL DEFAULT 0,
  token_count INTEGER NOT NULL DEFAULT 0,
  namespace_indexed TEXT DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('ready','processing','error')),
  error_message TEXT DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE(kb_id, normalized_name)           -- one active logical doc per name
);

CREATE TABLE ingestion_jobs (
  id TEXT PRIMARY KEY,
  kb_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  trigger_type TEXT NOT NULL CHECK (trigger_type IN ('upload','reindex_all','refresh_document')),
  status TEXT NOT NULL CHECK (status IN ('queued','running','completed','failed','cancelled')),
  force_reindex INTEGER NOT NULL DEFAULT 0, -- bool 0/1
  requested_by TEXT DEFAULT 'system',
  total_items INTEGER NOT NULL DEFAULT 0,
  processed_items INTEGER NOT NULL DEFAULT 0,
  skipped_items INTEGER NOT NULL DEFAULT 0,
  failed_items INTEGER NOT NULL DEFAULT 0,
  error_message TEXT DEFAULT '',
  created_at DATETIME NOT NULL,
  started_at DATETIME,
  finished_at DATETIME
);

CREATE TABLE ingestion_job_items (
  id TEXT PRIMARY KEY,
  job_id TEXT NOT NULL REFERENCES ingestion_jobs(id) ON DELETE CASCADE,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('queued','processing','completed','skipped','failed')),
  note TEXT DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE(job_id, document_id)
);

CREATE TABLE conversations (
  id TEXT PRIMARY KEY,
  kb_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  summary TEXT DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  last_message_at DATETIME,
  archived_at DATETIME
);

CREATE TABLE messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('system','user','assistant','tool')),
  content TEXT NOT NULL,
  model_name TEXT DEFAULT '',
  prompt_tokens INTEGER NOT NULL DEFAULT 0,
  completion_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  retrieval_query TEXT DEFAULT '',         -- condensed question used for retrieval
  created_at DATETIME NOT NULL
);

CREATE TABLE message_citations (
  id TEXT PRIMARY KEY,
  message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  citation_index INTEGER NOT NULL,
  kb_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  vector_chunk_id TEXT NOT NULL,           -- Chroma id
  source_label TEXT NOT NULL,              -- filename or URL
  excerpt TEXT NOT NULL,                   -- short snippet shown in UI
  page_number INTEGER,
  chunk_index INTEGER NOT NULL DEFAULT 0,
  score REAL NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  UNIQUE(message_id, citation_index)
);

CREATE TABLE reindex_runs (
  id TEXT PRIMARY KEY,
  kb_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  generation INTEGER NOT NULL,
  namespace TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued','running','completed','failed')),
  created_at DATETIME NOT NULL,
  started_at DATETIME,
  finished_at DATETIME,
  error_message TEXT DEFAULT '',
  UNIQUE(kb_id, generation)
);

CREATE INDEX idx_documents_kb_id ON documents(kb_id);
CREATE INDEX idx_conversations_kb_id ON conversations(kb_id);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id, created_at);
CREATE INDEX idx_citations_message_id ON message_citations(message_id);
CREATE INDEX idx_ingestion_jobs_kb_id ON ingestion_jobs(kb_id, created_at);
```

### Schema notes

1. `documents` is both logical identity and current file state for each filename in a KB.
2. Deleting a document is hard-delete: remove vectors, DB row, and file from disk.
3. Re-index all uses generation/namespace switching for consistency.

---

## 4) Backend Component Specifications

### 4.1 Config

`internal/config` loads env vars into a typed config struct with startup validation.

Required:
1. `OPENROUTER_API_KEY`
2. `MODEL_NAME`

Important defaults:
1. Go server `:8080`
2. SQLite path `./data/sqlite/app.db`
3. Chroma `http://localhost:8000`
4. Embedding model `sentence-transformers/all-MiniLM-L6-v2`

### 4.2 File Storage

Root: `./data/files`

Immutable storage layout:
`./data/files/<kb_id>/<document_id>/<sha256>_<original_name>`

Rules:
1. Persist file before enqueueing ingest job.
2. Never overwrite existing file path.
3. Never delete persisted originals in v1.
4. Validate extension + MIME sniff + max size.

### 4.3 Ingestion Service

Responsibilities:
1. Handle upload/URL ingestion requests.
2. Compute SHA-256.
3. Apply dedupe/replace logic.
4. Enqueue worker jobs.
5. Track progress and errors.

Filename/hash policy:
1. Match by `kb_id + normalized filename`.
2. If existing logical doc has same hash: skip and return user notice.
3. If hash differs: delete old vectors, replace file/metadata, then re-index.

### 4.4 Loader + OCR Strategy

Loader selection:
1. PDF text: `documentloaders.NewPDF`
2. `.md/.txt`: `NewText`
3. URL HTML: `NewHTML` after HTTP fetch and sanitization
4. CSV optional: `NewCSV`

OCR fallback:
1. If PDF extraction yields insufficient text (threshold), run OCR path.
2. OCR result saved as sidecar text artifact and indexed.
3. Store parser metadata (`parser_used='ocr'`) for observability.

### 4.5 Chunking + Embeddings

Chunking defaults:
1. Chunk size 800 tokens-equivalent (or char heuristic)
2. Overlap 120
3. Include metadata: `kb_id`, `document_id`, `source_label`, `page_number`, `chunk_index`, `sha256`

Embedding:
1. Adapter implementing `embeddings.Embedder`.
2. Default model `sentence-transformers/all-MiniLM-L6-v2`.
3. Batch embeddings for throughput; retry transient failures.

### 4.6 Vector Store (Chroma)

1. Use one Chroma collection for app domain.
2. Scope retrieval/index by namespace from KB (`active_namespace`).
3. On re-index all:
   - Create new generation namespace (`kb_<id>_g<n+1>`)
   - Rebuild fully into new namespace
   - Atomic DB switch `active_namespace` after success
   - Old namespace cleanup asynchronously

### 4.7 RAG Chain

Per request:
1. Build retriever (top-k, namespace-bound)
2. Create `ConversationalRetrievalQA` chain
3. Set `ReturnSourceDocuments = true`
4. Provide chat history from SQLite
5. Stream answer tokens to client
6. Persist assistant message + citations

Prompt policy:
1. Answer only from retrieved context
2. If insufficient context, say so explicitly
3. Include citation markers `[1] [2] ...` mapped to persisted citation rows

### 4.8 Memory Adapter (SQLite-backed)

Recommendation: use persistent history over ephemeral buffers.

Design:
1. Implement custom `schema.ChatMessageHistory` backed by SQLite repositories.
2. Fetch last N turns by token budget for chain input.
3. Keep full history in DB; short-term window passed to chain.
4. Optional future summarization table for very long conversations.

Why this is better than plain in-memory window:
1. Survives restarts
2. Multi-session consistency
3. Required for "resume existing conversations"

### 4.9 Service Boundaries

1. `KnowledgeBaseService`: CRUD KB, active namespace metadata
2. `DocumentService`: upload, dedupe/replace, delete, refresh one doc
3. `IngestionService`: orchestrate jobs and workers
4. `ChatService`: send message, run RAG chain, save message/citations
5. `ConversationService`: create/resume/list/rename/archive conversations

---

## 5) Frontend Pages and Component Specifications

Tech: Templ-rendered HTML + HTMX interactions + Alpine state; no JS build step.

### 5.1 Screens

1. KB Dashboard (`/`)
   - List all knowledge bases
   - Create KB modal
   - Show stats: docs count, conversation count, last updated
   - Action: open KB, archive KB

2. KB Detail (`/kbs/{kbID}`)
   - Tabs: Documents, Conversations, Settings
   - Documents tab:
     - Upload dropzone (files)
     - URL add form (optional feature toggle)
     - Document table with status (`ready/processing/error`)
     - Row actions: delete, refresh/re-index
     - Global action: Re-index All (confirm modal)
     - Progress panel fed by polling/SSE
   - Conversations tab:
     - List conversations by last message
     - New conversation button
     - Resume on click
   - Settings tab:
     - Display active model and embedding model
     - Show KB namespace/generation metadata

3. Chat View (`/kbs/{kbID}/conversations/{conversationID}`)
   - Message timeline
   - Composer with Enter-to-send and Shift+Enter newline
   - Streaming assistant output
   - Citation badges in answers; clickable side panel
   - Retrieved context panel shows source snippets/pages
   - Actions: stop generation, regenerate last response, new conversation

### 5.2 AI UX Best Practices (must-haves)

1. Explicit loading states for ingestion and generation.
2. Stream partial answer tokens to reduce perceived latency.
3. Show grounded citations and source snippets for every answer.
4. Clear "insufficient evidence" behavior when retrieval is weak.
5. Keep KB/conversation context visible so users understand scope.
6. Non-blocking uploads with job progress and retry actions.
7. Dedupe notices are explicit ("same file hash, upload skipped").

### 5.3 HTMX/Alpine interaction patterns

1. HTMX for form submit, tab/content swaps, partial refresh.
2. SSE endpoint for chat token stream and ingestion progress.
3. Alpine for local UI state: modal open/close, active tab, citation panel.

---

## 6) Full Ingestion Pipeline Flow (Step-by-Step)

### A. Upload / URL ingest request

1. User chooses KB and uploads file(s) or enters URL.
2. Backend validates KB exists and request constraints (size/type/URL policy).
3. File content is streamed to temp storage; SHA-256 computed during stream.
4. Determine logical document by `kb_id + normalized filename` (or canonical URL name for URL docs).

### B. Dedupe/replace decision

5. If no logical document exists:
   - Create `documents` row (`status='processing'`) with hash/storage metadata
6. If logical document exists:
   - Compare SHA with existing `documents.sha256`
   - Same SHA: create skipped job item and return notice; no re-index
   - Different SHA: mark document for replacement

### C. Persist + queue

7. Move upload from temp to immutable storage path.
8. Create ingestion job + job item(s).
9. Worker picks queued item.

### D. Parse/chunk/embed/index

10. Choose loader by mime/ext/source type.
11. Extract text; run OCR fallback when needed.
12. Split text into chunks with metadata.
13. Generate embeddings in batches.
14. Insert vectors to Chroma under KB active namespace.

### E. Finalize

15. Update `documents.status='ready'`, chunk/token counts, parser and namespace metadata.
16. For changed-file uploads, commit replacement file path/hash metadata.
17. Mark job item completed; aggregate job counters.
18. Publish progress events to UI.

### F. Re-index All

19. Acquire KB re-index lock.
20. Create new namespace generation.
21. Reprocess all active documents from their current stored files, ignoring hash checks.
22. On success, atomically switch KB `active_namespace`.
23. Mark re-index run completed; schedule old namespace cleanup.

---

## 7) Full Query/Chat Pipeline Flow (Step-by-Step)

1. User opens/creates a conversation within a KB.
2. User sends question.
3. Persist user message in SQLite immediately.
4. Load recent chat history (token-budgeted window) from SQLite.
5. Build namespace-scoped retriever for KB active namespace.
6. Initialize `ConversationalRetrievalQA` with `ReturnSourceDocuments=true`.
7. Chain condenses follow-up question using chat history.
8. Retriever fetches top-k chunks from Chroma.
9. LLM generates answer grounded in retrieved chunks; tokens streamed to UI.
10. Collect returned source documents and map to citation entries.
11. Persist assistant message + token stats + latency.
12. Persist `message_citations` rows with chunk/source metadata.
13. UI renders final answer with clickable citation markers and snippet panel.

Failure behavior:
1. If retrieval empty, assistant returns "insufficient evidence in selected KB."
2. If model call fails, show retry action and preserve user message.

---

## 8) HTTP API Routes

### 8.1 HTML routes (Templ pages)

1. `GET /` -> KB dashboard
2. `GET /kbs/{kbID}` -> KB detail (documents/conversations/settings)
3. `GET /kbs/{kbID}/conversations/{conversationID}` -> chat page

### 8.2 KB API

1. `GET /api/kbs`
2. `POST /api/kbs`
3. `PATCH /api/kbs/{kbID}`
4. `DELETE /api/kbs/{kbID}` (archive)
5. `POST /api/kbs/{kbID}/reindex-all`

### 8.3 Document API

1. `GET /api/kbs/{kbID}/documents`
2. `POST /api/kbs/{kbID}/documents/upload` (multipart)
3. `POST /api/kbs/{kbID}/documents/url` (optional)
4. `DELETE /api/kbs/{kbID}/documents/{documentID}` (hard delete + unindex + remove file)
5. `POST /api/kbs/{kbID}/documents/{documentID}/refresh`
6. `GET /api/kbs/{kbID}/documents/{documentID}/preview`

### 8.4 Ingestion jobs API

1. `GET /api/kbs/{kbID}/ingestion-jobs`
2. `GET /api/kbs/{kbID}/ingestion-jobs/{jobID}`
3. `GET /api/kbs/{kbID}/ingestion-jobs/{jobID}/events` (SSE)

### 8.5 Conversation/chat API

1. `GET /api/kbs/{kbID}/conversations`
2. `POST /api/kbs/{kbID}/conversations`
3. `PATCH /api/kbs/{kbID}/conversations/{conversationID}`
4. `DELETE /api/kbs/{kbID}/conversations/{conversationID}` (archive)
5. `GET /api/kbs/{kbID}/conversations/{conversationID}/messages`
6. `POST /api/kbs/{kbID}/conversations/{conversationID}/messages`
7. `GET /api/kbs/{kbID}/conversations/{conversationID}/stream` (SSE token stream)

### 8.6 Ops

1. `GET /healthz`
2. `GET /readyz`

---

## 9) `.env.example` Contents

```dotenv
# App
APP_ENV=development
HTTP_ADDR=:8080
DATA_DIR=./data
SQLITE_PATH=./data/sqlite/app.db

# LLM (required by assignment)
OPENROUTER_API_KEY=
MODEL_NAME=nvidia/nemotron-3-nano-30b-a3b:free
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1

# Embeddings
EMBEDDING_MODEL_NAME=sentence-transformers/all-MiniLM-L6-v2
EMBEDDING_PROVIDER=huggingface
EMBEDDING_ENDPOINT=http://localhost:8081

# Vector DB
CHROMA_URL=http://localhost:8000
CHROMA_COLLECTION=knowledge_base_rag

# RAG
RAG_TOP_K=6
RAG_SCORE_THRESHOLD=0.2
CHUNK_SIZE=800
CHUNK_OVERLAP=120
CHAT_HISTORY_MAX_TURNS=12

# Upload/Ingestion
MAX_UPLOAD_MB=50
INGEST_WORKERS=2
ENABLE_URL_INGEST=true
OCR_ENABLED=true
OCR_LANG=eng
```

---

## 10) Non-Functional Requirements

### 10.1 Concurrency and consistency

1. Per-KB lock for re-index operations; no overlapping `reindex-all` for same KB.
2. Uploads can run concurrently with per-document serialization on same logical filename.
3. SQLite writes wrapped in transactions to keep doc/job states consistent.

### 10.2 Streaming and latency

1. Use SSE for assistant token streaming and ingestion progress.
2. Target first-token latency < 2.5s under normal local conditions.
3. Heartbeats every 15s to keep SSE alive behind proxies.

### 10.3 Error handling

1. Structured JSON errors for API (`code`, `message`, `details`).
2. Retry transient failures (embedding/LLM/network) with capped exponential backoff.
3. Preserve failed job diagnostics (`error_message`) and allow retry.

### 10.4 Security

1. Validate file type by MIME sniff + extension allowlist.
2. Enforce max upload size and reject archives/executables.
3. Sanitize filenames; prevent path traversal.
4. URL ingestion uses safe fetch policy (timeout, size cap, local-network denylist).
5. Escape/sanitize rendered content to avoid XSS from document text.
6. Keep secrets only in env vars; never log API keys.

### 10.5 Observability

1. Structured logs with request/job IDs.
2. Metrics: ingestion duration, retrieval latency, token usage, failure counts.
3. Optional tracing hooks around ingestion and chat pipelines.

### 10.6 Testing strategy

1. Unit tests:
   - hash dedupe/replace rules
   - KB namespace switching logic
   - citation extraction/persistence
2. Integration tests:
   - SQLite repositories/migrations
   - Chroma insert/retrieval with namespace isolation
   - full upload->index->query flow
3. End-to-end tests:
   - HTMX chat flow with streaming
   - re-index all progression and zero-downtime switch
4. Regression fixtures:
   - scanned PDF OCR case
   - same filename same hash skip
   - same filename different hash replacement

---

## Future TODO (Post-v1)

1. Add CLI for ingestion, KB management, and non-UI chat.
2. Add auth/multi-user RBAC.
3. Add conversation summarization memory for long threads.
4. Add reranking stage and hybrid search (keyword + vector).
5. Add export/import for KB snapshots.
