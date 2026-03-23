# Implementation Plan — Personal Knowledge Base AI Agent

## 1. Purpose
This plan translates [PROJECT_SPEC.md](/Users/calugo/Projects/kno_base_RAG/docs/PROJECT_SPEC.md) into an execution sequence that another engineer or LLM can implement without hidden assumptions. It prioritizes infrastructure-first foundation phases, then end-to-end vertical feature slices. Each phase defines concrete outputs, commands, and verifiable exit gates.

## 2. Document Map
1. Purpose: why this plan exists and how it should be used.
2. Document map: quick description of each section.
3. Phase status table: delivery tracker for all phases.
4. Conventions and decisions: fixed implementation conventions.
5. Open questions: unresolved items from the spec that need explicit decisions.
6. Phase 0–7: ordered implementation phases with scope, tasks, and exit gates.
7. Cross-phase rules: constraints that apply to every phase.
8. Deferred backlog: post-MVP work including full testing strategy.
9. Final handoff checklist: objective release-readiness criteria.

## 3. Phase Status Table

| Phase | Title | Status | Notes |
|---|---|---|---|
| 0 | Bootstrap and Build Baseline | pending | Foundation |
| 1 | Config, Router, and Runtime Baseline | pending | Foundation |
| 2 | SQLite Migrations and Repository Foundation | pending | Foundation |
| 3 | Knowledge Base Management Slice | pending | Feature |
| 4 | Document Upload + Ingestion + Indexing Slice | pending | Feature |
| 5 | Document Lifecycle Slice (Delete, Refresh, Re-index All) | pending | Feature |
| 6 | Conversation Management + Chat Shell Slice | pending | Feature |
| 7 | RAG Chat + Streaming + Citations Slice | pending | Feature |

## 4. Conventions and Decisions
1. Go version: `1.22+` (from assignment brief).
2. Module path: not defined in spec; use `<MODULE_PATH>` placeholder until resolved (see Open Questions).
3. Build rule: run `templ generate` before every `go build` and `go run`.
4. Agent framework: `github.com/tmc/langchaingo` only; do not re-implement LangChain primitives from spec section `1)`.
5. LLM wiring: langchaingo OpenAI provider pointed at OpenRouter (`OPENROUTER_API_KEY`, `MODEL_NAME`, `OPENROUTER_BASE_URL`).
6. Vector isolation: one Chroma collection per KB (`kb_<id>`), selected via `vectorstores.WithNameSpace(kb.Namespace)`.
7. Embeddings: langchaingo huggingface embedder via `EMBEDDING_ENDPOINT`; default model `sentence-transformers/all-MiniLM-L6-v2`.
8. Persistence: SQLite for application data and langchaingo sqlite memory table in the same DB file.
9. Message persistence: app `messages` table remains the UI/system-of-record for display + citations; `langchaingo_messages` is chain-memory internals only.
10. Naming conventions:
   - IDs: ULID/UUID as text.
   - KB namespace: `kb_<kb_id>`.
   - Paths: all file paths under `./data/files` and `./data/sqlite`.

## 5. Open Questions
1. What is the final Go module path for `go.mod` (`<MODULE_PATH>`)?
2. Which migration execution approach should be used at runtime: custom SQL runner in `internal/storage/sqlite/db.go` or a third-party migration tool? The spec defines SQL files but not migration runner mechanism.
3. What is the exact ingestion worker startup model: always-on worker goroutine in `cmd/server/main.go` or start/stop via app wiring lifecycle hooks? The spec requires workers but not lifecycle detail.
4. Should KB archive (`DELETE /api/kbs/{kbID}`) physically block further writes immediately or only hide from UI? Spec says archive but not enforcement behavior.
5. What is the exact SSE payload schema for chat tokens and ingestion events (`event`, `data` JSON fields)? Endpoints are defined, payload contract is not.
6. For URL ingestion, what canonical `normalized_name` should be used for uniqueness (host+path, full URL, or user-provided label)? Spec requires normalized identity but not URL normalization algorithm.
7. Tesseract binary requirement is specified conceptually; installation and path resolution strategy (PATH vs configurable env var) is not specified.

## 6. Phase 0 — Bootstrap and Build Baseline

### Objective
Create a compilable server skeleton with the spec-defined directory layout and a repeatable `templ generate` + build/run workflow.

### Spec references
1. `2) Project Directory Structure`
2. `9) .env.example`
3. `10) Non-Functional Requirements` (logging baseline)

### Scope in
1. Initialize module and base folders/files from spec structure.
2. Add minimal `cmd/server/main.go` that starts an HTTP server.
3. Add placeholder router and one placeholder template so `templ generate` succeeds.
4. Ensure `.env.example` and top-level `README.md` are present and minimally aligned with run commands.

### Scope out
1. No DB schema implementation.
2. No Chroma/embedding/LLM wiring.
3. No business logic or feature handlers.

### Prerequisites
1. Open Question #1 resolved (module path) or temporary placeholder accepted.
2. `templ` CLI installed locally.

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/go.mod`
2. `/Users/calugo/Projects/kno_base_RAG/cmd/server/main.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/layout.templ`
5. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/generated/` (generated output)
6. `/Users/calugo/Projects/kno_base_RAG/.env.example`
7. `/Users/calugo/Projects/kno_base_RAG/README.md`

### Step-by-step tasks
1. Initialize module.
   ```bash
   go mod init <MODULE_PATH>
   go mod tidy
   ```
2. Create directories exactly as defined in spec section `2)`.
3. Implement minimal server entrypoint:
   - `main.go` loads HTTP address from env with default `:8080`.
   - Starts server with router from `internal/http/router.go`.
4. Add placeholder route in router (`GET /healthz` returns static 200 text or JSON).
5. Add minimal `layout.templ` component and ensure templ generation path exists.
6. Build/run verification sequence:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   ```
7. Verify health endpoint:
   ```bash
   curl -i http://localhost:8080/healthz
   ```

### Exit gate
1. `templ generate` exits `0`.
2. `go build ./cmd/server` exits `0`.
3. `curl -i http://localhost:8080/healthz` returns `HTTP/1.1 200`.

### Handoff notes
Phase 1 depends on a stable app entrypoint and router that already compiles and runs.

## 7. Phase 1 — Config, Router, and Runtime Baseline

### Objective
Establish typed configuration loading, middleware scaffolding, and production-safe runtime wiring with health/readiness endpoints.

### Spec references
1. `4.1 Config`
2. `8) HTTP API Routes` (Ops routes)
3. `10.2 Streaming and latency` (SSE readiness consideration)
4. `10.5 Logging`

### Scope in
1. Implement env parsing in `internal/config`.
2. Add request ID, logging, and panic recovery middleware.
3. Implement `GET /healthz` and `GET /readyz` handlers.
4. Ensure readiness checks SQLite connectivity; Chroma check can be stubbed with explicit TODO if Phase 2 not complete.

### Scope out
1. No feature APIs.
2. No migrations or repositories yet.
3. No Chroma/embedding/LLM clients yet.

### Prerequisites
1. Phase 0 complete.

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/internal/config/config.go`
2. `/Users/calugo/Projects/kno_base_RAG/internal/config/env.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/http/middleware/recover.go`
5. `/Users/calugo/Projects/kno_base_RAG/internal/http/middleware/logging.go`
6. `/Users/calugo/Projects/kno_base_RAG/internal/http/middleware/request_id.go`
7. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/health_handler.go`
8. `/Users/calugo/Projects/kno_base_RAG/internal/observability/logger.go`
9. `/Users/calugo/Projects/kno_base_RAG/cmd/server/main.go`

### Step-by-step tasks
1. Add `Config` struct containing at least:
   - `HTTP_ADDR`, `SQLITE_PATH`, `CHROMA_URL`, `OPENROUTER_API_KEY`, `MODEL_NAME`, `OPENROUTER_BASE_URL`, `EMBEDDING_ENDPOINT`, `EMBEDDING_MODEL_NAME`, `RAG_TOP_K`, `RAG_SCORE_THRESHOLD`, `CHUNK_SIZE`, `CHUNK_OVERLAP`, `CHAT_HISTORY_MAX_TURNS`, `MAX_UPLOAD_MB`, `INGEST_WORKERS`, `ENABLE_URL_INGEST`, `OCR_ENABLED`, `OCR_LANG`.
2. Implement env loader with defaults from spec section `9)`.
3. Wire middleware chain in `router.go`.
4. Implement `health_handler.go` responses:
   - `/healthz` liveness always `200`.
   - `/readyz` checks config loaded and SQLite openable.
5. Build/run verification:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   curl -i http://localhost:8080/healthz
   curl -i http://localhost:8080/readyz
   ```

### Exit gate
1. Both `/healthz` and `/readyz` return `HTTP 200`.
2. Startup fails fast with clear error if required env vars (`OPENROUTER_API_KEY`, `MODEL_NAME`) are missing.

### Handoff notes
Phase 2 will plug the real SQLite schema/migrations into the readiness check path created here.

## 8. Phase 2 — SQLite Migrations and Repository Foundation

### Objective
Implement the full SQLite schema and repository layer so all later feature slices can persist state reliably.

### Spec references
1. `3) SQLite Schema`
2. `4.8 Memory Adapter (SQLite-backed)`
3. `8) HTTP API Routes` (resource model support)
4. `10.1 Concurrency and consistency`

### Scope in
1. Add migration SQL files for all spec tables.
2. Implement SQLite connection, migration bootstrap, and repository interfaces/implementations.
3. Ensure langchaingo sqlite memory table can coexist in same DB file.

### Scope out
1. No full feature handlers yet.
2. No ingestion workers or RAG chain.

### Prerequisites
1. Phase 1 complete.
2. Open Question #2 resolved (migration execution approach).

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/migrations/0001_init.sql`
2. `/Users/calugo/Projects/kno_base_RAG/migrations/0002_citations.sql`
3. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/db.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/kb_repo.go`
5. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/document_repo.go`
6. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/conversation_repo.go`
7. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/message_repo.go`
8. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/citation_repo.go`
9. `/Users/calugo/Projects/kno_base_RAG/internal/storage/sqlite/ingest_repo.go`
10. `/Users/calugo/Projects/kno_base_RAG/internal/domain/kb.go`
11. `/Users/calugo/Projects/kno_base_RAG/internal/domain/document.go`
12. `/Users/calugo/Projects/kno_base_RAG/internal/domain/conversation.go`
13. `/Users/calugo/Projects/kno_base_RAG/internal/domain/message.go`
14. `/Users/calugo/Projects/kno_base_RAG/internal/domain/citation.go`
15. `/Users/calugo/Projects/kno_base_RAG/internal/domain/ingest_job.go`
16. `/Users/calugo/Projects/kno_base_RAG/internal/rag/memory_sqlite.go`
17. `/Users/calugo/Projects/kno_base_RAG/cmd/server/main.go`

### Step-by-step tasks
1. Create `0001_init.sql` with tables and indexes from spec section `3)`.
2. Create `0002_citations.sql` for `message_citations` (if split migration strategy is chosen); otherwise keep no-op file with comment and track in Open Questions.
3. Implement DB bootstrap in `db.go`:
   - open SQLite at `SQLITE_PATH`
   - enforce `PRAGMA foreign_keys = ON`
   - run migrations on startup.
4. Implement repository methods used by later phases:
   - KB: create/list/update/archive/get
   - Document: create/find-by-name/update/delete/list-by-kb
   - Conversation: create/list-by-kb/get/update/archive
   - Message: create/list-by-conversation
   - Citation: bulk insert by assistant message
   - Ingestion job: create/update/progress/get/list
5. Add thin wrapper in `rag/memory_sqlite.go`:
   - Accept shared `*sql.DB`, conversation/session ID, limit.
   - Return langchaingo sqlite3 history object.
6. Run server and verify schema creation:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   sqlite3 ./data/sqlite/app.db ".tables"
   ```

### Exit gate
1. `sqlite3 ./data/sqlite/app.db ".tables"` contains all spec tables (`knowledge_bases`, `documents`, `ingestion_jobs`, `conversations`, `messages`, `message_citations`) plus langchaingo table after memory adapter use.
2. `curl -i http://localhost:8080/readyz` returns `HTTP 200`.

### Handoff notes
Phase 3 can now implement KB UI/API directly on top of repository layer.

## 9. Phase 3 — Knowledge Base Management Slice

### Objective
Deliver end-to-end KB CRUD UX (list/create/update/archive) from HTTP API through storage and Templ UI.

### Spec references
1. `4.9 Service Boundaries` (`KnowledgeBaseService`)
2. `5.1 Screens` (KB Dashboard and KB Detail entrypoint)
3. `8.1 HTML routes`
4. `8.2 KB API`

### Scope in
1. KB dashboard page with list and create modal.
2. API handlers for KB list/create/update/archive.
3. KB detail shell page with tabs (content placeholders acceptable in this phase).

### Scope out
1. Document ingestion and chat features.
2. Re-index all behavior implementation (route can exist but return not-implemented if needed until Phase 5).

### Prerequisites
1. Phase 2 complete.

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/internal/app/app.go`
2. `/Users/calugo/Projects/kno_base_RAG/internal/app/wiring.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/kb_handler.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
5. `/Users/calugo/Projects/kno_base_RAG/internal/http/dto/request.go`
6. `/Users/calugo/Projects/kno_base_RAG/internal/http/dto/response.go`
7. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/kb_list.templ`
8. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/kb_detail.templ`
9. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/components/toasts.templ`
10. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/generated/`

### Step-by-step tasks
1. Implement `KnowledgeBaseService` with methods matching handlers.
2. Define routes:
   - HTML: `GET /`, `GET /kbs/{kbID}`
   - API: `GET/POST/PATCH/DELETE /api/kbs...`
3. Implement create flow:
   - Insert KB row with namespace `kb_<id>`.
4. Implement archive flow:
   - Set `archived_at` and hide archived KBs from default lists.
5. Implement HTMX partial updates for list refresh after create/archive.
6. Verify:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   curl -s http://localhost:8080/api/kbs
   curl -s -X POST http://localhost:8080/api/kbs -H "Content-Type: application/json" -d '{"name":"My KB","description":"notes"}'
   ```

### Exit gate
1. Creating a KB via `POST /api/kbs` returns `201` and persisted record appears in `GET /api/kbs`.
2. Opening `/` in browser shows KB list and create action.

### Handoff notes
Phase 4 will attach document tab actions and ingestion status to the existing KB detail page.

## 10. Phase 4 — Document Upload + Ingestion + Indexing Slice

### Objective
Deliver end-to-end document upload/URL ingest with dedupe/replace behavior, chunking, embeddings, and indexing into KB-scoped Chroma.

### Spec references
1. `4.2 File Storage`
2. `4.3 Ingestion Service`
3. `4.4 Loader + OCR Strategy`
4. `4.5 Chunking + Embeddings`
5. `4.6 Vector Store (Chroma)`
6. `5.1 KB Detail (Documents tab)`
7. `6) Ingestion Pipeline Flow`
8. `8.3 Document API`
9. `8.4 Ingestion Jobs API`

### Scope in
1. Multipart upload and optional URL ingest endpoints.
2. SHA-256 dedupe/replace logic.
3. File persistence under `./data/files/<kb_id>/<document_id>/<sha256>_<name>`.
4. Ingestion worker: parse/chunk/embed/index + job counters.
5. Documents tab UI showing status and dedupe notices.

### Scope out
1. Delete/refresh/re-index-all actions (Phase 5).
2. Chat/conversation features.

### Prerequisites
1. Phase 3 complete.
2. Chroma running at `CHROMA_URL`.
3. HuggingFace embedding server running at `EMBEDDING_ENDPOINT`.
4. Open Question #3 and #6 resolved.

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/internal/storage/filestore/filestore.go`
2. `/Users/calugo/Projects/kno_base_RAG/internal/storage/filestore/paths.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/service.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/worker.go`
5. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/loader_factory.go`
6. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/ocr.go`
7. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/hasher.go`
8. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/chunker.go`
9. `/Users/calugo/Projects/kno_base_RAG/internal/embeddings/provider.go`
10. `/Users/calugo/Projects/kno_base_RAG/internal/vector/chroma.go`
11. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/document_handler.go`
12. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/ingest_handler.go`
13. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
14. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/components/upload_modal.templ`
15. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/kb_detail.templ`
16. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/generated/`
17. `/Users/calugo/Projects/kno_base_RAG/internal/app/wiring.go`

### Step-by-step tasks
1. Implement upload handler for `POST /api/kbs/{kbID}/documents/upload`:
   - validate size/type.
   - stream to temp file and compute SHA256.
   - resolve `normalized_name`.
2. Implement dedupe/replace logic:
   - same hash: return skip notice.
   - different hash: delete old vectors by `document_id`, replace file metadata, enqueue ingest job.
3. Implement URL ingest route `POST /api/kbs/{kbID}/documents/url` (guarded by `ENABLE_URL_INGEST`).
4. Implement ingestion worker pipeline for single document:
   - loader select.
   - OCR fallback for low-text PDF and image/scanned docs.
   - chunk (`CHUNK_SIZE`, `CHUNK_OVERLAP`).
   - embed batches.
   - insert vectors into KB collection.
5. Update job counters and document status fields.
6. Implement job list/get and SSE event endpoint for progress.
7. Verify service dependencies:
   ```bash
   curl -fsS "${CHROMA_URL}/api/v1/heartbeat"
   ```
8. Build/run and verify upload flow:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   curl -i -F "files=@/absolute/path/sample.md" http://localhost:8080/api/kbs/<KB_ID>/documents/upload
   curl -s http://localhost:8080/api/kbs/<KB_ID>/documents
   ```

### Exit gate
1. Uploading a supported file creates a `documents` row with `status='ready'` after job completion.
2. Re-uploading same filename/same hash returns skip notice without new vectors.
3. `GET /api/kbs/{kbID}/ingestion-jobs/{jobID}` shows `processed_items=1` and `status='completed'`.

### Handoff notes
Phase 5 depends on vector delete and ingestion job primitives built here.

## 11. Phase 5 — Document Lifecycle Slice (Delete, Refresh, Re-index All)

### Objective
Deliver document delete/refresh and KB-wide re-index all behavior end-to-end with job progress and concurrency control.

### Spec references
1. `3) SQLite Schema` (document + job state)
2. `4.3 Ingestion Service` (replace and job tracking)
3. `4.6 Vector Store (Chroma)` (single-doc delete, collection rebuild)
4. `5.1 KB Detail` (row actions + re-index modal)
5. `6) Ingestion Pipeline Flow` (Re-index All)
6. `8.2 KB API` (`POST /reindex-all`)
7. `8.3 Document API` (`DELETE`, `POST refresh`)
8. `10.1 Concurrency and consistency`

### Scope in
1. Hard delete document endpoint (unindex + remove DB row + remove file).
2. Refresh single document endpoint.
3. Re-index all endpoint with per-KB mutex and full collection drop/recreate.
4. UI controls and progress feedback for these actions.

### Scope out
1. Conversation/chat logic.

### Prerequisites
1. Phase 4 complete.
2. Chroma and embedding endpoint running.
3. Open Question #3 resolved.

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/document_handler.go`
2. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/kb_handler.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/service.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/ingest/worker.go`
5. `/Users/calugo/Projects/kno_base_RAG/internal/vector/chroma.go`
6. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
7. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/kb_detail.templ`
8. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/components/toasts.templ`
9. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/generated/`

### Step-by-step tasks
1. Implement hard delete handler:
   - load document by id.
   - delete vectors for `document_id` from KB namespace.
   - delete file at `storage_path`.
   - delete document row in transaction.
2. Implement refresh handler:
   - create ingestion job for existing document.
   - re-run parse/chunk/embed/index from `storage_path`.
3. Implement KB re-index all handler:
   - acquire per-KB mutex.
   - create running ingestion job.
   - drop and recreate KB Chroma collection.
   - iterate ready docs and re-index from disk.
   - update counters and final status.
4. Return `409` on concurrent `reindex-all` for same KB.
5. Wire UI actions on documents tab and progress panel updates.
6. Build/run verification:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   curl -i -X POST http://localhost:8080/api/kbs/<KB_ID>/reindex-all
   curl -i -X POST http://localhost:8080/api/kbs/<KB_ID>/documents/<DOC_ID>/refresh
   curl -i -X DELETE http://localhost:8080/api/kbs/<KB_ID>/documents/<DOC_ID>
   ```

### Exit gate
1. `DELETE /api/kbs/{kbID}/documents/{documentID}` returns success and file no longer exists on disk.
2. `POST /api/kbs/{kbID}/reindex-all` creates a job whose final status is `completed` or `failed` with counters populated.
3. A second simultaneous `reindex-all` request for same KB returns `HTTP 409`.

### Handoff notes
Phase 6 will use the now-stable KB/document surfaces and statuses in conversation and chat pages.

## 12. Phase 6 — Conversation Management + Chat Shell Slice

### Objective
Deliver conversation creation/resume/listing and message history UI shell (without RAG generation yet).

### Spec references
1. `4.9 Service Boundaries` (`ConversationService`)
2. `5.1 Screens` (Conversations tab + Chat view shell)
3. `8.1 HTML routes`
4. `8.5 Conversation / Chat API`

### Scope in
1. Conversations tab list/create/rename/archive.
2. Chat page route and history loading from `messages`.
3. Message POST endpoint stub that persists user message and returns accepted/not-implemented generation response placeholder.

### Scope out
1. RAG retrieval and LLM answer generation.
2. Citation creation.

### Prerequisites
1. Phase 5 complete.

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/conversation_handler.go`
2. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/chat_handler.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/kb_detail.templ`
5. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/chat.templ`
6. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/components/conversation_list.templ`
7. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/generated/`
8. `/Users/calugo/Projects/kno_base_RAG/internal/app/wiring.go`

### Step-by-step tasks
1. Implement conversation handlers:
   - list/create/rename/archive under `/api/kbs/{kbID}/conversations`.
2. Implement HTML routes:
   - `GET /kbs/{kbID}/conversations/{conversationID}` chat page.
3. Implement message history endpoint:
   - `GET /api/kbs/{kbID}/conversations/{conversationID}/messages`.
4. Implement `POST /messages` stub:
   - persist user message row.
   - return placeholder response indicating generation arrives in Phase 7.
5. Update templates for conversation switch/new conversation flow.
6. Build/run verification:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   curl -s -X POST http://localhost:8080/api/kbs/<KB_ID>/conversations -H "Content-Type: application/json" -d '{"title":"Session 1"}'
   curl -s http://localhost:8080/api/kbs/<KB_ID>/conversations
   ```

### Exit gate
1. User can create a conversation and reopen it from KB detail conversations tab.
2. `GET /api/kbs/{kbID}/conversations/{conversationID}/messages` returns persisted messages in chronological order.

### Handoff notes
Phase 7 will replace message POST stub with full RAG chain execution and SSE token streaming.

## 13. Phase 7 — RAG Chat + Streaming + Citations Slice

### Objective
Deliver full conversational RAG with follow-up context, token streaming, and persisted source citations.

### Spec references
1. `1) LangChainGo: Out-of-the-Box vs Custom Build`
2. `4.6 Vector Store (Chroma)`
3. `4.7 RAG Chain`
4. `4.8 Memory Adapter (SQLite-backed)`
5. `5.1 Chat View`
6. `7) Chat Pipeline Flow`
7. `8.5 Conversation / Chat API`
8. `10.2 Streaming and latency`

### Scope in
1. LLM wiring to OpenRouter using langchaingo OpenAI provider.
2. Retriever + `ConversationalRetrievalQA` with `ReturnSourceDocuments=true`.
3. Conversation memory via langchaingo sqlite3 history per conversation.
4. SSE chat stream endpoint and incremental UI rendering.
5. Persist assistant messages and `message_citations`.

### Scope out
1. CLI.
2. Post-MVP features in spec Future TODO.

### Prerequisites
1. Phase 6 complete.
2. Chroma running and populated with at least one indexed document.
3. HuggingFace embedding endpoint running.
4. Valid `OPENROUTER_API_KEY` and `MODEL_NAME`.
5. Open Question #5 resolved (SSE payload contract).

### Files to create or modify
1. `/Users/calugo/Projects/kno_base_RAG/internal/rag/retriever.go`
2. `/Users/calugo/Projects/kno_base_RAG/internal/rag/chain.go`
3. `/Users/calugo/Projects/kno_base_RAG/internal/rag/citations.go`
4. `/Users/calugo/Projects/kno_base_RAG/internal/rag/prompt.go`
5. `/Users/calugo/Projects/kno_base_RAG/internal/http/handlers/chat_handler.go`
6. `/Users/calugo/Projects/kno_base_RAG/internal/http/router.go`
7. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/chat.templ`
8. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/components/citation_panel.templ`
9. `/Users/calugo/Projects/kno_base_RAG/internal/web/templates/generated/`
10. `/Users/calugo/Projects/kno_base_RAG/internal/app/wiring.go`

### Step-by-step tasks
1. Wire OpenRouter LLM client with langchaingo OpenAI provider in `internal/app/wiring.go`:
   ```go
   openai.New(
       openai.WithBaseURL(cfg.OpenRouterBaseURL),
       openai.WithToken(cfg.OpenRouterAPIKey),
       openai.WithModel(cfg.ModelName),
   )
   ```
2. Build retriever using KB namespace and `RAG_TOP_K`.
3. Build conversational retrieval chain with source docs enabled.
4. Connect `rag/memory_sqlite.go` session to conversation ID and history limit.
5. Replace message POST stub in `chat_handler.go`:
   - persist user message.
   - execute chain.
   - stream assistant tokens over SSE endpoint.
   - persist final assistant message + citations.
6. Render citation markers and side panel excerpts in chat template.
7. Build/run and verify full path:
   ```bash
   templ generate
   go build ./cmd/server
   go run ./cmd/server
   curl -N http://localhost:8080/api/kbs/<KB_ID>/conversations/<CONV_ID>/stream
   curl -s -X POST http://localhost:8080/api/kbs/<KB_ID>/conversations/<CONV_ID>/messages -H "Content-Type: application/json" -d '{"content":"What does document X say about Y?"}'
   ```

### Exit gate
1. Sending a chat message produces streamed assistant tokens on `/stream`.
2. Assistant response is persisted in `messages`, with at least one `message_citations` row when retrieval finds context.
3. Follow-up question in same conversation uses prior turns and returns context-aware answer.

### Handoff notes
MVP is functionally complete after this phase; remaining work goes to deferred backlog.

## 14. Cross-Phase Rules
1. Always run:
   ```bash
   templ generate
   go build ./cmd/server
   ```
   before declaring a phase complete.
2. Do not log secrets (`OPENROUTER_API_KEY`) or document contents.
3. Use structured JSON errors for API (`code`, `message`).
4. Keep app `messages` table and langchaingo `langchaingo_messages` logically separate.
5. Enforce path sanitization and file type limits on all upload paths.
6. Preserve route contracts from spec section `8)`; do not rename endpoints.
7. Keep UI implementation to Templ + HTMX + Alpine.js only (no JS build step).

## 15. Deferred Backlog
1. CLI for ingestion/KB/chat workflows (spec Future TODO #1).
2. Auth and multi-user RBAC (spec Future TODO #2).
3. Conversation summarization memory for long sessions (spec Future TODO #3).
4. Reranking and hybrid search (spec Future TODO #4).
5. KB export/import snapshots (spec Future TODO #5).
6. Testing strategy (deferred from all phases):
   - Unit: hash dedupe/replace, citation extraction, loader selection, service error paths.
   - Integration: SQLite migrations/repos, Chroma write/read isolation, ingestion pipeline.
   - End-to-end: HTMX conversation flow, SSE streaming, re-index-all progression.
   - Regression fixtures: OCR scanned docs, same-hash skip, different-hash replacement, hard-delete cleanup.

## 16. Final Handoff Checklist
1. `templ generate` and `go build ./cmd/server` succeed on clean checkout.
2. Server starts with `.env` values and responds `200` on `/healthz` and `/readyz`.
3. KB lifecycle works: create, list, update, archive.
4. Document lifecycle works: upload, same-hash skip, different-hash replace, hard delete.
5. Ingestion jobs expose progress and completion state via API/SSE.
6. Re-index all works per KB and rejects concurrent runs with `409`.
7. Conversation lifecycle works: create, list, resume, archive.
8. Chat works end-to-end: user message, retrieval, streamed assistant response, persisted citations.
9. Answers include citation markers and source excerpts from retrieved documents.
10. No unsupported features implemented beyond spec scope; unresolved items are recorded in Open Questions.
