# Implementation Plan — Personal Knowledge Base AI Agent

## 1. Purpose
This plan converts [PROJECT_SPEC.md](./PROJECT_SPEC.md) into an execution roadmap that another engineer or LLM can run step-by-step. It front-loads foundation work needed for runtime stability, then delivers user-visible vertical slices. Each phase has strict scope, explicit commands, and objective exit gates.

## 2. Document Map
1. Purpose: why this plan exists and what it optimizes for.
2. Document map: what each section contains.
3. Phase status table: tracking grid for all implementation phases.
4. Conventions and Decisions: fixed technical choices to reduce ambiguity.
5. Open Questions: unresolved decisions not specified by the spec.
6. Phase definitions: detailed objectives, scope, tasks, and pass criteria.
7. Cross-phase rules: constraints that apply to all phases.
8. Deferred backlog: testing and post-MVP work.
9. Final handoff checklist: release-readiness gates for MVP.

## 3. Phase Status Table

| Phase | Title | Status | Notes |
|---|---|---|---|
| 0 | Runtime Bootstrap (Merged) | pending | Foundation; merged old 0 + 1 |
| 1 | SQLite Schema + Migration Runner | pending | Foundation |
| 2 | LangChainGo + External Connectivity Baseline | pending | Foundation |
| 3 | Knowledge Base Management Slice | pending | Feature |
| 4A | Parsing Pipeline Slice (No External Vector/Embedding Calls) | pending | Feature risk-burn |
| 4B | Upload-to-Index Slice (Embedding + Chroma Integration) | pending | Feature |
| 5 | Document Lifecycle Slice (Delete, Refresh, Re-index All) | pending | Feature |
| 6 | Conversation Management + Chat Shell Slice | pending | Feature |
| 7 | RAG Chat + Streaming + Citations Slice | pending | Feature |

## 4. Conventions and Decisions
1. Go version: `1.22+`.
2. Module path: `github.com/gr8cally/knowledge_base_RAG`.
3. Build invariant: run `templ generate` before every `go build` and `go run`.
4. Migration approach: custom in-process SQL runner in `internal/storage/sqlite/db.go` (no third-party migration dependency).
5. LLM provider: LangChainGo OpenAI-compatible client pointed to OpenRouter.
6. Embeddings: LangChainGo HuggingFace embedder via `EMBEDDING_ENDPOINT`.
7. Vector store isolation: use LangChainGo's Chroma store directly and scope every read/write with `vectorstores.WithNameSpace(kb.Namespace)`.
8. Persistence split:
   - App UX data in our tables (`knowledge_bases`, `documents`, `conversations`, `messages`, `message_citations`, `ingestion_jobs`).
   - LangChainGo memory data in `langchaingo_messages` (same SQLite file, separate concern).
9. Frontend stack: Templ + HTMX + Alpine.js only; no JS build step.
10. Path style in this plan: repository-relative paths only.
11. Framework-first rule: if LangChainGo already provides the loader, splitter, vector store, retriever, chain, or memory primitive needed for the brief, use it directly or wrap it thinly for configuration only.
12. MVP file scope rule: only the assignment-brief formats are in scope for ingestion: PDF, Markdown, text, scanned documents, and optional web pages.

## 5. Open Questions
1. `DELETE /api/kbs/{kbID}` archives KBs; should archived KBs be API-readable but non-writable, or fully hidden except admin/debug contexts?
2. SSE payload contract is not specified in the spec: exact event names and JSON envelope (`event`, `type`, `payload`) need a fixed decision.
3. URL identity normalization for dedupe (`normalized_name`) is unspecified for URL docs (full URL vs canonicalized host/path/query rules).
4. OCR binary resolution is unspecified: use system `tesseract` from `$PATH` only, or add explicit env config for binary path.
5. Ingestion worker lifecycle strategy is unspecified: always-on goroutine at server boot vs lazy-start per job.
6. Pinned LangChainGo Chroma wrapper may not expose document-level delete/upsert needed for Phase 5 replace/delete semantics; if confirmed, keep custom fallback limited to that narrow lifecycle gap only.

## 6. Phase 0 — Runtime Bootstrap (Merged)

### Objective
Stand up a compile/run-ready server baseline with config loading, middleware, and health/readiness endpoints in one phase.

### Spec references
1. `2) Project Directory Structure`
2. `4.1 Config`
3. `8.6 Ops`
4. `9) .env.example`
5. `10.5 Logging`

### Scope in
1. Initialize module and root server wiring.
2. Implement env loading with required/optional vars from spec section `9)`.
3. Implement middleware: request ID, structured logging, panic recovery.
4. Implement `/healthz` and `/readyz` with SQLite openability check.
5. Add minimal templates so `templ generate` works from day one.

### Scope out
1. No schema migrations.
2. No repositories.
3. No Chroma/embedding/LLM calls.
4. No feature API behavior beyond ops endpoints.

### Prerequisites
1. `templ` CLI installed.
2. Go toolchain `1.22+` installed.

### Files to create or modify
1. `go.mod`
2. `cmd/server/main.go`
3. `internal/config/config.go`
4. `internal/config/env.go`
5. `internal/http/router.go`
6. `internal/http/middleware/request_id.go`
7. `internal/http/middleware/logging.go`
8. `internal/http/middleware/recover.go`
9. `internal/http/handlers/health_handler.go`
10. `internal/observability/logger.go`
11. `internal/web/templates/layout.templ`
12. `internal/web/templates/generated/` (generated files)
13. `.env.example`
14. `README.md`

### Step-by-step tasks
1. Initialize module:
```bash
go mod init github.com/gr8cally/knowledge_base_RAG
go mod tidy
```
2. Create baseline router and ops handlers.
3. Implement config loader with required env vars:
   - `OPENROUTER_API_KEY`
   - `MODEL_NAME`
4. Add middleware chain in router.
5. Ensure templ generation works:
```bash
templ generate
```
6. Verify compile/run:
```bash
go build ./cmd/server
go run ./cmd/server
```
7. Verify ops endpoints:
```bash
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```

### Exit gate
1. `templ generate` exits `0`.
2. `go build ./cmd/server` exits `0`.
3. `/healthz` returns `HTTP 200`.
4. `/readyz` returns `HTTP 200` when SQLite path is writable.

### Handoff notes
Phase 1 assumes a stable app bootstrap and a reusable DB-open path in readiness checks.

## 7. Phase 1 — SQLite Schema + Migration Runner

### Objective
Implement schema migrations and DB bootstrap so persistence is deterministic across environments.

### Spec references
1. `3) SQLite Schema`
2. `2) Project Directory Structure` (`migrations/` and `internal/storage/sqlite/`)
3. `10.1 Concurrency and consistency`

### Scope in
1. Create migration SQL files from spec schema.
2. Build custom migration runner in SQLite bootstrap.
3. Enable foreign keys and migration version tracking.

### Scope out
1. No repository implementations yet.
2. No feature handlers.
3. No langchaingo memory wiring yet.

### Prerequisites
1. Phase 0 complete.

### Files to create or modify
1. `migrations/0001_init.sql`
2. `migrations/0002_citations.sql`
3. `internal/storage/sqlite/db.go`
4. `cmd/server/main.go`

### Step-by-step tasks
1. Create SQL migrations exactly matching spec section `3)`.
2. Implement a migration runner in `internal/storage/sqlite/db.go`:
   - creates `schema_migrations` table if missing.
   - applies pending `.sql` files in lexical order.
   - runs inside transactions.
3. Set `PRAGMA foreign_keys = ON` on every DB connection.
4. Wire migration execution at server startup before routes are served.
5. Verify:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
sqlite3 ./data/sqlite/app.db ".tables"
sqlite3 ./data/sqlite/app.db "SELECT name FROM schema_migrations ORDER BY name;"
```

### Exit gate
1. All spec tables exist in `./data/sqlite/app.db`.
2. `schema_migrations` contains `0001_init.sql` and `0002_citations.sql`.
3. Server still returns `200` from `/readyz` after migrations run.

### Handoff notes
Phase 2 will use the same DB handle and readiness infrastructure to validate external integrations without changing DB boot semantics.

## 8. Phase 2 — LangChainGo + External Connectivity Baseline

### Objective
Establish validated connectivity to Chroma and embedding service, and wire LangChainGo provider factories without feature logic.

### Spec references
1. `1) LangChainGo: Out-of-the-Box vs Custom Build`
2. `4.5 Chunking + Embeddings`
3. `4.6 Vector Store (Chroma)`
4. `9) .env.example`
5. `8.6 Ops`

### Scope in
1. Embedding provider factory.
2. Chroma client/factory and connectivity check.
3. App wiring for OpenRouter LLM client (factory-level only).
4. Readiness check extended to include Chroma connectivity.

### Scope out
1. No ingestion jobs.
2. No upload handlers.
3. No retrieval chain execution.

### Prerequisites
1. Phase 1 complete.
2. Chroma is running at `CHROMA_URL`.
3. HuggingFace embedding server is running at `EMBEDDING_ENDPOINT`.

### Files to create or modify
1. `internal/embeddings/provider.go`
2. `internal/vector/chroma.go`
3. `internal/app/wiring.go`
4. `internal/http/handlers/health_handler.go`
5. `internal/config/config.go`

### Step-by-step tasks
1. Implement embedding factory returning `embeddings.Embedder` using LangChainGo HuggingFace integration.
2. Implement Chroma client creation and heartbeat check.
3. Implement OpenRouter LLM factory in `internal/app/wiring.go` using LangChainGo OpenAI-compatible provider.
4. Update `/readyz` to fail when Chroma is unreachable.
5. Verify service dependencies outside app:
```bash
curl -fsS "${CHROMA_URL}/api/v1/heartbeat"
curl -fsS "${EMBEDDING_ENDPOINT}" || true
```
6. Verify app readiness behavior:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -i http://localhost:8080/readyz
```

### Exit gate
1. `/readyz` returns `HTTP 200` when SQLite + Chroma are reachable.
2. `/readyz` returns non-200 when Chroma is intentionally unavailable.
3. Server compiles and starts cleanly with external factories wired.

### Handoff notes
Phase 3 can start user-facing KB features on a stable runtime; Phase 4B will consume embedding/chroma factories already validated here.

## 9. Phase 3 — Knowledge Base Management Slice

### Objective
Deliver KB dashboard and KB CRUD APIs end-to-end.

### Spec references
1. `4.9 Service Boundaries` (`KnowledgeBaseService`)
2. `5.1 Screens` (KB dashboard/detail shell)
3. `8.1 HTML routes`
4. `8.2 KB API`

### Scope in
1. KB domain model and repository.
2. KB service + handlers.
3. Dashboard and KB detail shell pages.

### Scope out
1. Document ingestion.
2. Conversation/chat behavior.
3. Re-index implementation.

### Prerequisites
1. Phase 2 complete.

### Files to create or modify
1. `internal/domain/kb.go`
2. `internal/storage/sqlite/kb_repo.go`
3. `internal/http/handlers/kb_handler.go`
4. `internal/http/dto/request.go`
5. `internal/http/dto/response.go`
6. `internal/http/router.go`
7. `internal/web/templates/kb_list.templ`
8. `internal/web/templates/kb_detail.templ`
9. `internal/web/templates/components/toasts.templ`
10. `internal/web/templates/generated/`
11. `internal/app/app.go`
12. `internal/app/wiring.go`

### Step-by-step tasks
1. Implement `KnowledgeBaseService` methods: create/list/update/archive/get.
2. Create namespace on KB creation (`kb_<kb_id>` in DB row).
3. Implement API routes in spec section `8.2`.
4. Render dashboard page and KB list with HTMX-friendly partial updates.
5. Render KB detail shell with Documents/Conversations/Settings tabs (content placeholders allowed).
6. Verify APIs:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -s http://localhost:8080/api/kbs
curl -i -X POST http://localhost:8080/api/kbs -H "Content-Type: application/json" -d '{"name":"My KB","description":"notes"}'
curl -s http://localhost:8080/api/kbs
```

### Exit gate
1. `POST /api/kbs` returns `201` and persisted KB appears in `GET /api/kbs`.
2. `GET /` renders KB dashboard with at least one KB row after creation.

### Handoff notes
Phase 4A uses KB context and pages from this slice to attach document ingest workflows.

## 10. Phase 4A — Parsing Pipeline Slice (No External Vector/Embedding Calls)

### Objective
Isolate and validate file intake, hashing, loader selection, OCR fallback routing, and chunk generation before external indexing is introduced.

### Spec references
1. `4.2 File Storage`
2. `4.3 Ingestion Service` (dedupe/replace decision)
3. `4.4 Loader + OCR Strategy`
4. `4.5 Chunking + Embeddings` (chunking subset only)
5. `6) Ingestion Pipeline Flow` (stages A–D excluding embed/index)

### Scope in
1. Document + ingestion-job domain/repositories.
2. File storage service and path policy.
3. SHA-256 hasher and dedupe/replace decision logic.
4. Loader factory + LangChainGo text splitter pipeline.
5. Worker dry-run mode that parses/splits and logs chunk summary (no embedding/chroma calls yet).

### Scope out
1. Embedding generation.
2. Chroma insert/delete operations.
3. Full document upload UI integration.

### Prerequisites
1. Phase 3 complete.
2. Open Questions #3 and #4 resolved.

### Files to create or modify
1. `internal/domain/document.go`
2. `internal/domain/ingest_job.go`
3. `internal/storage/sqlite/document_repo.go`
4. `internal/storage/sqlite/ingest_repo.go`
5. `internal/storage/filestore/filestore.go`
6. `internal/storage/filestore/paths.go`
7. `internal/ingest/hasher.go`
8. `internal/ingest/loader_factory.go`
9. `internal/ingest/ocr.go`
10. `internal/ingest/service.go`
11. `internal/ingest/worker.go`
12. `internal/http/handlers/document_handler.go`
13. `internal/http/router.go`

### Step-by-step tasks
1. Implement document repository methods for:
   - create
   - find by `kb_id + normalized_name`
   - update metadata/status
   - list by KB
2. Implement ingestion job repository with counters (`total_items`, `processed_items`, `skipped_items`, `failed_items`).
3. Implement file persistence path generation:
   - `data/files/<kb_id>/<document_id>/<sha256>_<original_name>`
4. Implement upload intake in `document_handler.go` up to queue stage:
   - save temp file
   - compute SHA256
   - apply dedupe/replace decision
   - enqueue dry-run ingest job
5. Implement dry-run worker path:
   - parse with LangChainGo loaders
   - split with LangChainGo text splitter
   - update `chunk_count`
   - log chunk summary to stdout
   - set document status `ready` when parse/chunk succeeds
6. Keep Phase 4A file support limited to assignment-brief types only.
7. Verify dry-run behavior:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -i -F "files=@./README.md" http://localhost:8080/api/kbs/<KB_ID>/documents/upload
```
8. Check logs for chunk output and DB state:
```bash
sqlite3 ./data/sqlite/app.db "SELECT display_name, chunk_count, status FROM documents WHERE kb_id='<KB_ID>';"
```
9. Verify different-hash replace path (same filename, changed content):
```bash
printf "v1 content\n" > /tmp/replace-check.txt
curl -i -F "files=@/tmp/replace-check.txt;filename=replace-check.txt" http://localhost:8080/api/kbs/<KB_ID>/documents/upload
sqlite3 ./data/sqlite/app.db "SELECT id, sha256, storage_path FROM documents WHERE kb_id='<KB_ID>' AND normalized_name='replace-check.txt';"
# Save first sha256/storage_path values from query output.

printf "v2 changed content\n" > /tmp/replace-check.txt
curl -i -F "files=@/tmp/replace-check.txt;filename=replace-check.txt" http://localhost:8080/api/kbs/<KB_ID>/documents/upload
sqlite3 ./data/sqlite/app.db "SELECT id, sha256, storage_path FROM documents WHERE kb_id='<KB_ID>' AND normalized_name='replace-check.txt';"
# Confirm sha256 and storage_path changed compared to first upload.
ls -l <OLD_STORAGE_PATH_FROM_FIRST_UPLOAD>
# Old path should not exist after replacement.
```

### Exit gate
1. Uploading a markdown/text file produces a `documents` row with `status='ready'` and `chunk_count > 0`.
2. Worker logs chunk summary for the uploaded document.
3. Same-filename/same-hash upload is skipped with explicit notice.
4. Re-uploading same filename with different content updates `documents.sha256` and `documents.storage_path`, and removes the previous file from disk.

### Handoff notes
Phase 4B replaces dry-run worker path with full embedding + Chroma indexing while keeping validated parsing/dedupe code intact.

## 11. Phase 4B — Upload-to-Index Slice (Embedding + Chroma Integration)

### Objective
Complete the document ingestion vertical slice by adding embeddings and LangChainGo Chroma indexing to the already-validated parsing pipeline.

### Spec references
1. `4.3 Ingestion Service`
2. `4.5 Chunking + Embeddings`
3. `4.6 Vector Store (Chroma)`
4. `5.1 KB Detail` (documents tab interactions)
5. `6) Ingestion Pipeline Flow`
6. `8.3 Document API`
7. `8.4 Ingestion Jobs API`

### Scope in
1. Embed chunks via HuggingFace endpoint.
2. Insert vectors through LangChainGo's Chroma store, scoped by `vectorstores.WithNameSpace(kb.Namespace)`.
3. Add ingestion job status endpoints + SSE progress stream.
4. Connect document tab UI for upload/status/progress.

### Scope out
1. Hard delete/refresh/re-index-all lifecycle controls.
2. Chat/conversation functionality.

### Prerequisites
1. Phase 4A complete.
2. Chroma running.
3. Embedding endpoint running.

### Files to create or modify
1. `internal/embeddings/provider.go`
2. `internal/vector/chroma.go`
3. `internal/ingest/worker.go`
4. `internal/ingest/service.go`
5. `internal/http/handlers/document_handler.go`
6. `internal/http/handlers/ingest_handler.go`
7. `internal/http/router.go`
8. `internal/web/templates/kb_detail.templ`
9. `internal/web/templates/components/upload_modal.templ`
10. `internal/web/templates/generated/`
11. `internal/app/wiring.go`

### Step-by-step tasks
1. Replace dry-run worker step with full pipeline:
   - parse with LangChainGo loaders
   - split with LangChainGo text splitter
   - embed chunks
   - write vectors through LangChainGo Chroma store
2. Ensure metadata includes `kb_id`, `document_id`, `source_label`, `chunk_index`.
3. Keep supported file types limited to the assignment brief; do not add CSV support.
4. Use custom Chroma code only if a specific required operation is not exposed by the pinned LangChainGo wrapper; if that happens, document the exact API gap before coding around it.
5. Expose ingestion job APIs:
   - `GET /api/kbs/{kbID}/ingestion-jobs`
   - `GET /api/kbs/{kbID}/ingestion-jobs/{jobID}`
   - `GET /api/kbs/{kbID}/ingestion-jobs/{jobID}/events` (SSE)
6. Add documents-tab progress rendering via HTMX + SSE.
7. Verify end-to-end upload-to-index:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -i -F "files=@./README.md" http://localhost:8080/api/kbs/<KB_ID>/documents/upload
curl -s http://localhost:8080/api/kbs/<KB_ID>/ingestion-jobs
curl -s http://localhost:8080/api/kbs/<KB_ID>/documents
```

### Exit gate
1. Uploaded file reaches `documents.status='ready'` and has non-zero `chunk_count`.
2. Ingestion job reaches `status='completed'` with `processed_items=1`.
3. Vectors exist in the LangChainGo Chroma store under the KB namespace and are queryable by retriever setup in Phase 7.

### Handoff notes
Phase 5 will reuse vector delete and job orchestration primitives from this phase.

## 12. Phase 5 — Document Lifecycle Slice (Delete, Refresh, Re-index All)

### Objective
Deliver operational document controls: hard delete, single-doc refresh, and KB-wide re-index all.

### Spec references
1. `3) SQLite Schema`
2. `4.3 Ingestion Service`
3. `4.6 Vector Store (Chroma)`
4. `6) Ingestion Pipeline Flow` (Re-index All)
5. `8.2 KB API` (`POST /reindex-all`)
6. `8.3 Document API` (`DELETE`, `POST refresh`)
7. `10.1 Concurrency and consistency`

### Scope in
1. Hard delete endpoint:
   - delete vectors for doc
   - delete file from disk
   - delete document DB row
2. Refresh endpoint:
   - re-run ingestion from current file path
3. Re-index all endpoint:
   - per-KB mutex
   - clear and rebuild the KB namespace in LangChainGo Chroma
   - reprocess all ready docs
4. UI actions for delete/refresh/re-index-all.

### Scope out
1. Conversations and chat.

### Prerequisites
1. Phase 4B complete.
2. Open Question #5 resolved (worker lifecycle).

### Files to create or modify
1. `internal/ingest/service.go`
2. `internal/ingest/worker.go`
3. `internal/vector/chroma.go`
4. `internal/http/handlers/document_handler.go`
5. `internal/http/handlers/kb_handler.go`
6. `internal/http/router.go`
7. `internal/web/templates/kb_detail.templ`
8. `internal/web/templates/components/toasts.templ`
9. `internal/web/templates/generated/`

### Step-by-step tasks
1. Implement hard delete in a transaction-safe sequence:
   - unindex vectors by `document_id` using LangChainGo if available; otherwise use the narrowest possible fallback adapter for delete only
   - remove `storage_path` file
   - remove document row
2. Implement single-document refresh (`/refresh`) as new ingestion job.
3. Implement `POST /api/kbs/{kbID}/reindex-all`:
   - reject concurrent same-KB requests (`409`)
   - clear KB namespace in LangChainGo Chroma
   - iterate docs and ingest
   - update job counters
4. Expose progress through job endpoints/SSE.
5. Verify lifecycle operations:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -i -X POST http://localhost:8080/api/kbs/<KB_ID>/documents/<DOC_ID>/refresh
curl -i -X POST http://localhost:8080/api/kbs/<KB_ID>/reindex-all
curl -i -X DELETE http://localhost:8080/api/kbs/<KB_ID>/documents/<DOC_ID>
```
6. Verify file removal:
```bash
sqlite3 ./data/sqlite/app.db "SELECT storage_path FROM documents WHERE id='<DOC_ID>';"
# (Run before delete to capture path)
ls -l <CAPTURED_STORAGE_PATH>
# After delete, ls should fail for that path
```

### Exit gate
1. Hard-delete removes DB row and physical file.
2. Re-index-all creates a job with counters and terminal status.
3. Concurrent re-index-all for the same KB returns `HTTP 409`.

### Handoff notes
Phase 6 can build conversation/chat shell on top of stable document and KB lifecycle behavior.

## 13. Phase 6 — Conversation Management + Chat Shell Slice

### Objective
Deliver conversation CRUD and chat page shell with persisted message history, without LLM generation yet.

### Spec references
1. `4.9 Service Boundaries` (`ConversationService`)
2. `5.1 Screens` (Conversations tab, Chat view)
3. `8.1 HTML routes`
4. `8.5 Conversation / Chat API`

### Scope in
1. Conversation domain + repository.
2. Message domain + repository for user/assistant rows.
3. Conversation APIs and HTML navigation.
4. Chat page rendering and history load endpoint.

### Scope out
1. Retrieval chain.
2. Streaming generation.
3. Citation persistence.

### Prerequisites
1. Phase 5 complete.

### Files to create or modify
1. `internal/domain/conversation.go`
2. `internal/domain/message.go`
3. `internal/storage/sqlite/conversation_repo.go`
4. `internal/storage/sqlite/message_repo.go`
5. `internal/http/handlers/conversation_handler.go`
6. `internal/http/handlers/chat_handler.go`
7. `internal/http/router.go`
8. `internal/web/templates/chat.templ`
9. `internal/web/templates/components/conversation_list.templ`
10. `internal/web/templates/kb_detail.templ`
11. `internal/web/templates/generated/`
12. `internal/app/wiring.go`

### Step-by-step tasks
1. Implement conversation CRUD APIs under `/api/kbs/{kbID}/conversations`.
2. Implement chat page route:
   - `GET /kbs/{kbID}/conversations/{conversationID}`.
3. Implement message history endpoint:
   - `GET /api/kbs/{kbID}/conversations/{conversationID}/messages`.
4. Implement temporary POST message behavior:
   - persist user message row
   - return accepted placeholder while Phase 7 generation is pending.
5. Verify conversation flow:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -i -X POST http://localhost:8080/api/kbs/<KB_ID>/conversations -H "Content-Type: application/json" -d '{"title":"Session 1"}'
curl -s http://localhost:8080/api/kbs/<KB_ID>/conversations
curl -s http://localhost:8080/api/kbs/<KB_ID>/conversations/<CONV_ID>/messages
```

### Exit gate
1. User can create/resume/archive conversations from KB UI.
2. Message history endpoint returns persisted rows in chronological order.

### Handoff notes
Phase 7 replaces placeholder message POST with full RAG generation and citation persistence.

## 14. Phase 7 — RAG Chat + Streaming + Citations Slice

### Objective
Deliver complete conversational RAG with follow-up context, SSE token streaming, and source citations persisted and rendered.

### Spec references
1. `1) LangChainGo: Out-of-the-Box vs Custom Build`
2. `4.7 RAG Chain`
3. `4.8 Memory Adapter (SQLite-backed)`
4. `7) Chat Pipeline Flow`
5. `8.5 Conversation / Chat API`
6. `10.2 Streaming and latency`

### Scope in
1. Chat/citation domain and repository completion.
2. Retriever + `ConversationalRetrievalQA` with `ReturnSourceDocuments=true`.
3. LangChainGo sqlite memory adapter in same DB file.
4. SSE token stream endpoint.
5. Assistant message + citation persistence and UI rendering.

### Scope out
1. CLI.
2. Future TODO enhancements beyond MVP.

### Prerequisites
1. Phase 6 complete.
2. LangChainGo Chroma store populated for the KB namespace.
3. Valid OpenRouter credentials.
4. Open Question #2 resolved (SSE payload contract).

### Files to create or modify
1. `internal/domain/citation.go`
2. `internal/rag/retriever.go`
3. `internal/rag/chain.go`
4. `internal/rag/memory_sqlite.go`
5. `internal/rag/citations.go`
6. `internal/rag/prompt.go`
7. `internal/storage/sqlite/citation_repo.go`
8. `internal/http/handlers/chat_handler.go`
9. `internal/http/router.go`
10. `internal/web/templates/chat.templ`
11. `internal/web/templates/components/citation_panel.templ`
12. `internal/web/templates/generated/`
13. `internal/app/wiring.go`

### Step-by-step tasks
1. Wire OpenRouter LLM in `internal/app/wiring.go`:
```go
openai.New(
    openai.WithBaseURL(cfg.OpenRouterBaseURL),
    openai.WithToken(cfg.OpenRouterAPIKey),
    openai.WithModel(cfg.ModelName),
)
```
2. Build KB-scoped retriever and chain (`top-k=RAG_TOP_K`, `ReturnSourceDocuments=true`).
3. Implement `rag/memory_sqlite.go` thin adapter over LangChainGo sqlite3 history using conversation ID as session key.
4. Replace temporary POST message behavior:
   - persist user message
   - run chain
   - stream assistant tokens on `/stream`
   - persist assistant message + citations
5. Render inline citation markers and side panel excerpts.
6. Verify:
```bash
templ generate
go build ./cmd/server
go run ./cmd/server
curl -N http://localhost:8080/api/kbs/<KB_ID>/conversations/<CONV_ID>/stream
curl -i -X POST http://localhost:8080/api/kbs/<KB_ID>/conversations/<CONV_ID>/messages -H "Content-Type: application/json" -d '{"content":"Summarize the uploaded document"}'
sqlite3 ./data/sqlite/app.db "SELECT COUNT(*) FROM message_citations;"
```

### Exit gate
1. Assistant tokens stream over SSE for each chat request.
2. Assistant response and citations are persisted in SQLite.
3. Follow-up question in same conversation uses prior context and returns grounded output.

### Handoff notes
MVP is complete after this phase; remaining scope is deferred backlog.

## 15. Cross-Phase Rules
1. Before every phase exit check, run:
```bash
templ generate
go build ./cmd/server
```
2. Never log secrets or raw file contents.
3. Use structured API errors (`code`, `message`).
4. Keep LangChainGo memory table separate from app message/citation tables conceptually and in code boundaries.
5. All file paths must remain under `data/` for persistence operations.
6. Upload handling must enforce MIME + extension checks and size limits.
7. Do not introduce features outside [PROJECT_SPEC.md](./PROJECT_SPEC.md); record unresolved gaps under Open Questions.
8. Before implementing any framework-adjacent component, explicitly map it to a LangChainGo primitive first. Custom code is allowed only after writing down why the library primitive is insufficient for this project.

## 16. Deferred Backlog
1. CLI for ingestion, KB management, and non-UI chat.
2. Auth and multi-user RBAC.
3. Conversation summarization memory.
4. Reranking and hybrid retrieval.
5. KB export/import snapshots.
6. Testing strategy (deferred by instruction):
   - Unit: dedupe/replace, loader selection, citation mapping, error handling branches.
   - Integration: migrations, repos, Chroma operations, embedding calls.
   - E2E: upload-to-answer flow, SSE streams, re-index-all path.
   - Regression fixtures: OCR scans, same-hash skip, different-hash replace, hard-delete cleanup.

## 17. Final Handoff Checklist
1. `templ generate` and `go build ./cmd/server` pass on clean checkout.
2. `/healthz` and `/readyz` return expected status in healthy environment.
3. KB CRUD works from API and dashboard UI.
4. Upload ingestion supports dedupe/replace and reaches indexed-ready state.
5. Document delete/refresh/re-index-all work with correct job counters.
6. Conversation create/resume/archive and message history work.
7. Chat RAG returns streamed assistant output grounded in retrieved docs.
8. Answers include persisted citations visible in UI.
9. No absolute local-machine paths remain in plan file.
10. Open Questions are explicitly resolved or consciously deferred before implementation starts.
