# Personal Knowledge Base AI Agent

## Project Specification (Go + LangChainGo + Templ + HTMX)

---

## 1) LangChainGo: Out-of-the-Box vs Custom Build

### LangChainGo out-of-the-box (use directly)

1. Document loading and splitting:
   - `documentloaders.NewPDF`, `NewText`, `NewHTML`
   - `Load(ctx)` when extraction and chunking are separate steps
   - `LoadAndSplit(ctx, splitter)` only when the loader itself should own splitting
2. Text splitters:
   - Recursive/character-based splitters for chunking
3. Embedding abstraction:
   - `embeddings.Embedder` interface (plug-in model/provider)
4. Vector storage:
   - Chroma vector store integration from LangChainGo
   - `vectorstores.WithNameSpace(kb.Namespace)` scopes read/write operations per KB
5. RAG retrieval chain:
   - `chains.NewConversationalRetrievalQA`
   - `ReturnSourceDocuments = true` for citations
6. LLM streaming:
   - Streaming interfaces and callback handlers
7. Memory primitives:
   - Conversation memory wrappers paired with a custom persisted history backend

### Must be custom-built

1. Multi-KB domain model (KBs, docs, conversations, ingestion jobs)
2. Hash-based dedupe/replace behavior tied to filename identity
3. Persistent file storage layout and file lifecycle rules
4. OCR path for scanned docs and loader selection logic
5. Thin memory adapter wiring langchaingo's sqlite3 memory to our DB and conversation ID
6. Conversation/citation persistence and UI rendering (our `messages` table, separate from langchaingo's internal table)
7. HTTP handlers, HTMX interactions, streaming endpoints, and Templ views
8. Ingestion worker queue and progress reporting
9. Security controls (upload validation, URL fetch protections, path sanitization)

Rule:
1. If LangChainGo already provides a loader, splitter, vector store, retriever, chain, or memory primitive needed by the brief, use it directly or wrap it thinly for configuration only.
2. Custom code is allowed only for application-specific orchestration, persistence, dedupe/replace policy, OCR, and UI concerns.

---

## 2) Project Directory Structure

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
│   │   └── hasher.go
│   ├── rag/
│   │   ├── chain.go
│   │   ├── retriever.go
│   │   ├── memory_sqlite.go     -- thin adapter (~15 lines) wrapping langchaingo sqlite3 memory
│   │   ├── citations.go
│   │   └── prompt.go
│   ├── embeddings/
│   │   └── provider.go          -- factory only; wraps langchaingo huggingface embedder
│   ├── vector/
│   │   └── chroma.go          -- thin LangChainGo Chroma factory/wrapper only
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
│   └── web/
│       ├── templates/
│       │   ├── layout.templ
│       │   ├── kb_list.templ
│       │   ├── kb_detail.templ
│       │   ├── chat.templ
│       │   ├── components/
│       │   │   ├── upload_modal.templ
│       │   │   ├── citation_panel.templ
│       │   │   ├── conversation_list.templ
│       │   │   └── toasts.templ
│       │   └── generated/
│       └── static/
│           ├── css/app.css
│           ├── js/alpine.min.js
│           └── js/htmx.min.js
├── migrations/
│   ├── 0001_init.sql
│   └── 0002_citations.sql
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

## 3) SQLite Schema

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE knowledge_bases (
  id          TEXT PRIMARY KEY,           -- ULID/UUID
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  namespace   TEXT NOT NULL,              -- LangChainGo vectorstore namespace for this KB: 'kb_' || id
  created_at  DATETIME NOT NULL,
  updated_at  DATETIME NOT NULL,
  archived_at DATETIME
);

CREATE TABLE documents (
  id             TEXT PRIMARY KEY,
  kb_id          TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  source_type    TEXT NOT NULL CHECK (source_type IN ('file','url')),
  display_name   TEXT NOT NULL,           -- logical identity in KB
  normalized_name TEXT NOT NULL,          -- lowercase/trimmed for uniqueness
  source_uri     TEXT NOT NULL,           -- original URL or original filename
  sha256         TEXT NOT NULL,
  storage_path   TEXT NOT NULL,           -- current file path on disk
  mime_type      TEXT NOT NULL,
  size_bytes     INTEGER NOT NULL,
  parser_used    TEXT NOT NULL DEFAULT '',
  chunk_count    INTEGER NOT NULL DEFAULT 0,
  status         TEXT NOT NULL CHECK (status IN ('ready','processing','error')),
  error_message  TEXT NOT NULL DEFAULT '',
  created_at     DATETIME NOT NULL,
  updated_at     DATETIME NOT NULL,
  UNIQUE(kb_id, normalized_name)          -- one active logical doc per name per KB
);

CREATE TABLE ingestion_jobs (
  id               TEXT PRIMARY KEY,
  kb_id            TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  trigger_type     TEXT NOT NULL CHECK (trigger_type IN ('upload','reindex_all','refresh_document')),
  status           TEXT NOT NULL CHECK (status IN ('queued','running','completed','failed')),
  total_items      INTEGER NOT NULL DEFAULT 0,
  processed_items  INTEGER NOT NULL DEFAULT 0,
  skipped_items    INTEGER NOT NULL DEFAULT 0,
  failed_items     INTEGER NOT NULL DEFAULT 0,
  error_message    TEXT NOT NULL DEFAULT '',
  created_at       DATETIME NOT NULL,
  started_at       DATETIME,
  finished_at      DATETIME
);

CREATE TABLE conversations (
  id              TEXT PRIMARY KEY,
  kb_id           TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
  title           TEXT NOT NULL,
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL,
  last_message_at DATETIME,
  archived_at     DATETIME
);

CREATE TABLE messages (
  id              TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  role            TEXT NOT NULL CHECK (role IN ('user','assistant')),
  content         TEXT NOT NULL,
  created_at      DATETIME NOT NULL
);

CREATE TABLE message_citations (
  id             TEXT PRIMARY KEY,
  message_id     TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  citation_index INTEGER NOT NULL,
  document_id    TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  source_label   TEXT NOT NULL,           -- filename or URL shown in UI
  excerpt        TEXT NOT NULL,           -- short snippet shown in UI
  chunk_index    INTEGER NOT NULL DEFAULT 0,
  score          REAL NOT NULL DEFAULT 0,
  created_at     DATETIME NOT NULL,
  UNIQUE(message_id, citation_index)
);

CREATE INDEX idx_documents_kb_id ON documents(kb_id);
CREATE INDEX idx_conversations_kb_id ON conversations(kb_id);
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id, created_at);
CREATE INDEX idx_citations_message_id ON message_citations(message_id);
CREATE INDEX idx_ingestion_jobs_kb_id ON ingestion_jobs(kb_id, created_at);
```

### Schema notes

1. `documents` holds both logical identity and current file state per filename in a KB.
2. Deleting a document is hard-delete: remove its vectors from the KB's Chroma collection, delete the DB row, delete the file from disk.
3. Re-index all drops the KB's Chroma collection and recreates it from scratch — no namespace rotation.
4. Individual document status (`ready`/`processing`/`error`) is tracked on the `documents` row; job-level progress counters live on `ingestion_jobs`.

---

## 4) Backend Component Specifications

### 4.1 Config

`internal/config` loads env vars into a typed config struct with startup validation.

Required env vars:
1. `OPENROUTER_API_KEY`
2. `MODEL_NAME`

Defaults:
1. Server bind: `:8080`
2. SQLite path: `./data/sqlite/app.db`
3. Chroma URL: `http://localhost:8000`
4. Embedding model: `BAAI/bge-small-en-v1.5`

### 4.2 File Storage

Root: `./data/files`

Storage layout:
```
./data/files/<kb_id>/<document_id>/<sha256>_<original_name>
```

Rules:
1. Persist file to disk before enqueueing the ingest job.
2. Never overwrite an existing file path (content-addressed by sha256 prefix).
3. Deleting a document removes its file from disk.
4. Validate extension + MIME sniff + max size on upload.

### 4.3 Ingestion Service

Responsibilities:
1. Handle upload/URL ingestion requests.
2. Compute SHA-256.
3. Apply dedupe/replace logic.
4. Enqueue worker jobs.
5. Track progress via counters on `ingestion_jobs`.

Filename/hash policy:
1. Match by `kb_id + normalized filename`.
2. Same hash as existing doc: skip, return user notice ("same file hash, upload skipped").
3. Different hash: delete old vectors from Chroma, replace file and metadata, re-index.

### 4.4 Loader + OCR Strategy

Loader selection (by MIME/extension):
1. PDF: `documentloaders.NewPDF`; if extracted text is below threshold (< 100 chars/page average), fall through to OCR.
2. `.md` / `.txt`: `documentloaders.NewText`
3. URL: HTTP fetch → sanitize HTML → `documentloaders.NewHTML`
4. Images / scanned PDFs: Tesseract OCR wrapper; record `parser_used='ocr'` on the document row.
5. CSV is not part of MVP scope because it is not required by the assignment brief, even though LangChainGo has a CSV loader.

OCR is a single call to the system Tesseract binary. No sidecar artifacts — extracted text is passed directly to the LangChainGo text splitter.

### 4.5 Chunking + Embeddings

Chunking defaults:
1. Chunk size: 800 chars
2. Overlap: 120 chars
3. Chunk metadata: `kb_id`, `document_id`, `source_label`, `chunk_index`

Embedding:
1. `embeddings/provider.go` exposes a single factory function returning a `embeddings.Embedder`.
2. Internally uses `embeddings/huggingface.NewHuggingface(WithModel(...), WithClient(...))` from langchaingo — no custom embedding code.
3. Default model: `BAAI/bge-small-en-v1.5` via `EMBEDDING_ENDPOINT`.
4. Batch embeddings; retry transient failures with simple linear backoff (3 attempts).
5. Chunking uses a LangChainGo text splitter; do not maintain a separate custom chunker implementation.

### 4.6 Vector Store (Chroma)

1. Use LangChainGo's Chroma vector store integration directly.
2. Use one application collection and scope documents by `vectorstores.WithNameSpace(kb.Namespace)`.
3. `knowledge_bases.namespace` stores the KB namespace string (`kb_<id>`), not a standalone Chroma collection name.
4. On single document delete or replace: delete that document's vectors from the store by document metadata or deterministic IDs, then reinsert if replacing.
4. On re-index all:
   - Clear that KB namespace from the shared Chroma store.
   - Reprocess all `ready` documents from their stored files, ignoring hash checks.
   - Brief downtime during rebuild is acceptable (single-user app, eval context).

### 4.7 RAG Chain

LLM client: configured in `app/wiring.go` using langchaingo's OpenAI provider pointed at OpenRouter:
```go
openai.New(
    openai.WithBaseURL(cfg.OpenRouterBaseURL),
    openai.WithToken(cfg.OpenRouterAPIKey),
    openai.WithModel(cfg.ModelName),
)
```
Streaming is handled via langchaingo's built-in callback system — no custom streaming code.

Per chat request:
1. Build retriever from the LangChainGo Chroma store scoped with `vectorstores.WithNameSpace(kb.Namespace)` (top-k = `RAG_TOP_K`).
2. Create `ConversationalRetrievalQA` chain with `ReturnSourceDocuments = true`.
3. Provide recent chat history (last N turns) from SQLite.
4. Stream answer tokens to client via SSE.
5. Persist assistant message to `messages`.
6. Persist `message_citations` rows from returned source documents.

Prompt policy:
1. Answer only from retrieved context.
2. If retrieval is empty or below score threshold, respond with "insufficient evidence in selected knowledge base."
3. Include citation markers `[1] [2] ...` in the answer body, mapped to persisted `message_citations` rows.

### 4.8 Memory Adapter (SQLite-backed)

1. `rag/memory_sqlite.go` is a thin adapter (~15 lines) wrapping langchaingo's `memory/sqlite3` package.
2. Uses `sqlite3.NewSqliteChatMessageHistory(WithDB(ourDB), WithSession(conversationID), WithLimit(N))` — namespaced per conversation.
3. Passed directly into `NewConversationalRetrievalQAFromLLM` as the `schema.Memory`.
4. LangChainGo manages its own `langchaingo_messages` table in the same SQLite file. This is separate from our `messages` table, which is used for UI rendering and citation linking. Both coexist in the same DB file.

### 4.9 Service Boundaries

1. `KnowledgeBaseService`: CRUD KB, namespace management.
2. `DocumentService`: upload, dedupe/replace, delete, refresh single doc.
3. `IngestionService`: orchestrate jobs and workers.
4. `ChatService`: send message, run RAG chain, save message + citations.
5. `ConversationService`: create/list/resume/archive conversations.

---

## 5) Frontend Pages and Component Specifications

Tech: Templ-rendered HTML + HTMX interactions + Alpine.js local state. No JS build step.

### 5.1 Screens

**KB Dashboard (`/`)**
- List all knowledge bases (name, doc count, conversation count, last updated).
- Create KB modal (name + optional description).
- Action: open KB, archive KB.

**KB Detail (`/kbs/{kbID}`)**
- Tabs: Documents | Conversations | Settings
- Documents tab:
  - Upload dropzone (multi-file).
  - URL add form (behind `ENABLE_URL_INGEST` flag).
  - Document table with status badge (`ready` / `processing` / `error`).
  - Row actions: delete, refresh (re-index single doc).
  - Global action: Re-index All (confirm modal).
  - Progress panel updated via SSE from active ingestion job.
- Conversations tab:
  - List conversations sorted by `last_message_at`.
  - New conversation button.
  - Click to resume.
- Settings tab:
  - Display active LLM model and embedding model (read-only from config).

**Chat View (`/kbs/{kbID}/conversations/{conversationID}`)**
- Message timeline (user + assistant turns).
- Composer: Enter to send, Shift+Enter for newline.
- Streaming assistant output via SSE.
- Citation badges inline in answer; click to open side panel with excerpt.
- Actions: new conversation.

### 5.2 AI UX Best Practices (must-haves)

1. Loading states during ingestion and generation (spinner, disabled inputs).
2. Stream partial answer tokens to reduce perceived latency.
3. Citation badges and source excerpts for every answer.
4. Explicit "insufficient evidence" message when retrieval yields no results.
5. Active KB name visible in chat view header.
6. Non-blocking uploads: job progress shown via SSE; errors surfaced with doc-row status.
7. Dedupe notice displayed inline when same-hash upload is skipped.

### 5.3 HTMX/Alpine Interaction Patterns

1. HTMX: form submit, tab/content swaps, partial page refresh after actions.
2. SSE: chat token stream (`/stream` endpoint) and ingestion progress (`/events` endpoint).
3. Alpine: local UI state for modal open/close, active tab, citation panel toggle.

---

## 6) Ingestion Pipeline Flow

### A. Upload / URL ingest request

1. User selects KB and uploads file(s) or enters URL.
2. Backend validates KB exists and enforces constraints (size limit, allowed types, URL policy).
3. File is streamed to temp path; SHA-256 computed during stream.
4. Determine logical document identity: `kb_id` + `normalized_name`.

### B. Dedupe/replace decision

5. No existing logical doc → create `documents` row (`status='processing'`).
6. Existing logical doc found:
   - Same SHA: return "same file hash, upload skipped" notice; no further action.
   - Different SHA: delete old vectors from Chroma, update `documents` row metadata, set `status='processing'`.

### C. Persist + queue

7. Move file from temp to immutable storage path.
8. Create `ingestion_jobs` row (`status='queued'`, `total_items=1`).
9. Worker goroutine picks queued job.

### D. Parse / chunk / embed / index

10. Select loader by MIME type / extension / source type.
11. Extract text; run OCR if PDF text is below threshold.
12. Split into chunks with metadata.
13. Generate embeddings in batches.
14. Insert vectors through LangChainGo's Chroma vector store using `vectorstores.WithNameSpace(kb.Namespace)`.

### E. Finalize

15. Update `documents` row: `status='ready'`, `chunk_count`, `parser_used`.
16. Update `ingestion_jobs`: `processed_items++`, `status='completed'`, `finished_at`.
17. Publish job completion event via SSE to UI.

### F. Re-index All

18. Acquire in-memory per-KB mutex (reject concurrent reindex for same KB).
19. Create `ingestion_jobs` row (`trigger_type='reindex_all'`, `status='running'`, `total_items=N`).
20. Clear the KB namespace in the shared Chroma store.
21. For each `ready` document in KB: parse → chunk → embed → insert (ignoring hash checks).
    - On success: `processed_items++`.
    - On failure: set `documents.status='error'`, `failed_items++`.
22. Mark job `completed` (or `failed` if all docs failed); set `finished_at`.
23. Publish final progress event to UI.

---

## 7) Chat Pipeline Flow

1. User opens or creates a conversation within a KB.
2. User sends a question.
3. Persist user message to `messages` immediately.
4. Load last `CHAT_HISTORY_MAX_TURNS` messages from SQLite for the conversation.
5. Build the KB's Chroma retriever using `vectorstores.WithNameSpace(kb.Namespace)` to scope to the correct namespace.
6. Initialize `ConversationalRetrievalQA` chain with `ReturnSourceDocuments=true`.
7. Chain condenses follow-up question using chat history into a standalone query.
8. Retriever fetches top-k chunks from Chroma.
9. LLM generates answer grounded in retrieved chunks; tokens streamed to client via SSE.
10. Map returned source documents to `message_citations` rows.
11. Persist assistant message to `messages`.
12. Persist `message_citations` rows.
13. UI renders final answer with citation badges; side panel shows excerpts.

Failure behavior:
1. Empty retrieval or all scores below threshold → assistant replies with "insufficient evidence in selected knowledge base."
2. LLM call failure → surface error toast; preserve user message; allow retry.

---

## 8) HTTP API Routes

### 8.1 HTML routes (Templ pages)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | KB dashboard |
| `GET` | `/kbs/{kbID}` | KB detail (documents / conversations / settings tabs) |
| `GET` | `/kbs/{kbID}/conversations/{conversationID}` | Chat view |

### 8.2 KB API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/kbs` | List all KBs |
| `POST` | `/api/kbs` | Create KB |
| `PATCH` | `/api/kbs/{kbID}` | Update KB name/description |
| `DELETE` | `/api/kbs/{kbID}` | Archive KB |
| `POST` | `/api/kbs/{kbID}/reindex-all` | Trigger re-index all |

### 8.3 Document API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/kbs/{kbID}/documents` | List documents |
| `POST` | `/api/kbs/{kbID}/documents/upload` | Upload file(s) (multipart) |
| `POST` | `/api/kbs/{kbID}/documents/url` | Add URL document (optional) |
| `DELETE` | `/api/kbs/{kbID}/documents/{documentID}` | Hard delete: unindex + remove file + delete row |
| `POST` | `/api/kbs/{kbID}/documents/{documentID}/refresh` | Re-index single document |

### 8.4 Ingestion Jobs API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/kbs/{kbID}/ingestion-jobs` | List jobs for KB |
| `GET` | `/api/kbs/{kbID}/ingestion-jobs/{jobID}` | Get job status and counters |
| `GET` | `/api/kbs/{kbID}/ingestion-jobs/{jobID}/events` | SSE: job progress stream |

### 8.5 Conversation / Chat API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/kbs/{kbID}/conversations` | List conversations |
| `POST` | `/api/kbs/{kbID}/conversations` | Create conversation |
| `PATCH` | `/api/kbs/{kbID}/conversations/{conversationID}` | Rename conversation |
| `DELETE` | `/api/kbs/{kbID}/conversations/{conversationID}` | Archive conversation |
| `GET` | `/api/kbs/{kbID}/conversations/{conversationID}/messages` | Load message history |
| `POST` | `/api/kbs/{kbID}/conversations/{conversationID}/messages` | Send message (triggers RAG) |
| `GET` | `/api/kbs/{kbID}/conversations/{conversationID}/stream` | SSE: assistant token stream |

### 8.6 Ops

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness check |
| `GET` | `/readyz` | Readiness check (DB + Chroma reachable) |

---

## 9) `.env.example`

```dotenv
# App
APP_ENV=development
HTTP_ADDR=:8080
DATA_DIR=./data
SQLITE_PATH=./data/sqlite/app.db

# LLM (required)
OPENROUTER_API_KEY=
MODEL_NAME=nvidia/nemotron-3-nano-30b-a3b:free
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1

# Embeddings
EMBEDDING_MODEL_NAME=BAAI/bge-small-en-v1.5
EMBEDDING_ENDPOINT=https://router.huggingface.co/hf-inference

# Vector DB
CHROMA_URL=http://localhost:8000

# RAG
RAG_TOP_K=6
RAG_SCORE_THRESHOLD=0.2
CHUNK_SIZE=800
CHUNK_OVERLAP=120
CHAT_HISTORY_MAX_TURNS=12

# Upload / Ingestion
MAX_UPLOAD_MB=50
INGEST_WORKERS=2
ENABLE_URL_INGEST=true
OCR_ENABLED=true
OCR_LANG=eng
```

---

## 10) Non-Functional Requirements

### 10.1 Concurrency and consistency

1. In-memory per-KB mutex for re-index operations; concurrent `reindex-all` for the same KB is rejected with HTTP 409.
2. Uploads for the same logical filename are serialized per KB via the same mutex.
3. SQLite writes use transactions to keep document and job states consistent.

### 10.2 Streaming and latency

1. SSE for assistant token streaming and ingestion job progress.
2. SSE heartbeat every 15s to keep connections alive behind proxies.

### 10.3 Error handling

1. Structured JSON error responses for API: `{"code": "...", "message": "..."}`.
2. Transient failures (embedding/LLM/network): retry up to 3 times with linear backoff.
3. Failed job/document errors stored in `error_message` column; visible in document table row.

### 10.4 Security

1. File type validated by MIME sniff + extension allowlist (PDF, MD, TXT, PNG, JPG, TIFF).
2. Enforce `MAX_UPLOAD_MB`; reject archives and executables.
3. Sanitize filenames; prevent path traversal.
4. URL ingestion: enforce timeout, response size cap, block private/loopback addresses.
5. Sanitize document text rendered in HTML to prevent XSS.
6. API keys only in env vars; never logged.

### 10.5 Logging

1. Structured JSON logs (using `log/slog`) with `request_id` and `job_id` fields.
2. Log levels: DEBUG (dev), INFO (prod).
3. Never log API keys or file contents.

---

## Future TODO (Post-v1)

1. Add CLI for ingestion, KB management, and non-UI chat (bonus per brief).
2. Add auth / multi-user RBAC.
3. Add conversation summarization for very long threads.
4. Add reranking stage and hybrid search (keyword + vector).
5. Add export/import for KB snapshots.
6. Testing strategy:
   - Unit: hash dedupe/replace rules, citation extraction, loader selection.
   - Integration: SQLite repos/migrations, Chroma insert/retrieval with namespace isolation, full upload→index→query flow.
   - E2E: HTMX chat flow with streaming, re-index all progression.
   - Regression fixtures: scanned PDF OCR, same-hash skip, different-hash replacement.
