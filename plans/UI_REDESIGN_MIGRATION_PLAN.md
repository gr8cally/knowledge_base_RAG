# UI Redesign Migration Plan

## Purpose

This plan translates the one-page workspace redesign into an execution-grade migration sequence that another LLM or engineer can follow with minimal interpretation.

This plan is intentionally:

1. phase-based
2. verification-driven
3. explicit about files and responsibilities
4. strict about what must keep working after each phase

The target outcome is a single workspace UI that satisfies the assignment brief requirement to provide a high-quality AI UX while preserving the current backend behavior.

## Planning Rules

Every phase in this plan must satisfy these rules:

1. The application must remain runnable at the end of the phase.
2. The phase must have observable success criteria.
3. Existing critical flows must not regress unless the phase explicitly replaces them.
4. Old routes may coexist temporarily, but the workspace route becomes the primary UX.
5. Templ + HTMX + Alpine.js must be used for the redesign.
6. SSE remains the mechanism for assistant streaming and ingestion progress.

## Current State Summary

Current UI state:

1. Home page lists KBs and supports KB creation.
2. KB detail page contains conversations and documents.
3. Chat is still primarily a separate page.
4. UI is currently inline-template-heavy and more imperative than originally planned.
5. Current visible UX is still closer to CRUD/admin than AI workspace.

## Target State Summary

Target UI state:

1. One workspace page at `/`
2. Left sidebar:
   - KB list
   - active KB summary
   - conversation list
   - sources panel
3. Main pane:
   - active conversation header
   - message timeline
   - citations/evidence
   - sticky composer
4. Old standalone KB/chat pages reduced to compatibility or redirect paths

## State Contract

Workspace state is carried by query parameters:

1. `?kb=<KB_ID>`
2. `?kb=<KB_ID>&conversation=<CONVERSATION_ID>`

Rules:

1. If `kb` is missing:
   - select the first KB by most recent update if any exist
   - otherwise render the empty workspace onboarding state
2. If `kb` is present but invalid:
   - render a recoverable empty state with KB list still visible
3. If `conversation` is present but does not belong to the active KB:
   - ignore it
   - fall back to the most recent conversation in the active KB
   - if none exists, render the empty conversation state
4. URL must remain the source of truth for selected KB and conversation

## Route Strategy

Primary route:

1. `GET /`

Temporary compatibility routes:

1. `GET /kbs/{kbID}`
2. `GET /kbs/{kbID}/conversations/{conversationID}`

These remain during migration and are retired in the final phase.

Suggested HTML partial endpoints:

1. `GET /ui/sidebar/kbs`
2. `GET /ui/sidebar/summary`
3. `GET /ui/sidebar/conversations`
4. `GET /ui/sidebar/documents`
5. `GET /ui/main/chat`
6. `GET /ui/main/empty`

Parameters are passed via query string:

1. `kb`
2. `conversation`

## Component Ownership

Recommended Templ ownership:

1. `workspace.templ`
   - page shell and global two-pane layout
2. `components/kb_list.templ`
   - KB switcher list
3. `components/kb_summary.templ`
   - active KB summary card
4. `components/conversation_list.templ`
   - active KB conversation list
5. `components/document_list.templ`
   - sources panel
6. `components/chat_header.templ`
   - active conversation header
7. `components/message_timeline.templ`
   - message list container
8. `components/message_item.templ`
   - user/assistant message rendering
9. `components/citation_block.templ`
   - expandable evidence block
10. `components/chat_composer.templ`
   - sticky composer

## Regression Checklist

These behaviors must be checked repeatedly through the migration:

1. create KB
2. switch KB
3. create conversation
4. open conversation
5. archive conversation
6. upload document
7. refresh document
8. delete document
9. re-index all
10. send user message
11. stream assistant response
12. persist citations
13. reload page and recover selected state

## Phase 1: Introduce Workspace Shell

### Objective

Create the new one-page workspace shell at `/` without yet moving real data interactions into HTMX partials.

### Scope In

1. New workspace route at `/`
2. Two-pane layout
3. Static or server-rendered initial sections
4. Empty placeholders for sidebar and main pane

### Scope Out

1. KB switching behavior
2. conversation switching behavior
3. document actions
4. SSE integration

### Files To Create Or Modify

1. `internal/http/router.go`
2. `internal/http/handlers/workspace_handler.go`
3. `internal/web/templates/workspace.templ`
4. `internal/web/templates/components/kb_list.templ`
5. `internal/web/templates/components/kb_summary.templ`
6. `internal/web/templates/components/conversation_list.templ`
7. `internal/web/templates/components/document_list.templ`
8. `internal/web/templates/components/chat_header.templ`
9. `internal/web/templates/components/message_timeline.templ`
10. `internal/web/templates/generated/`

### Backend Changes

1. Add workspace HTML route handler.
2. Resolve initial selected KB/conversation from query params.
3. Pass initial workspace state into the main template.

### Frontend Changes

1. Build desktop two-pane shell.
2. Build mobile stacked shell.
3. Render placeholder sections for:
   - KB list
   - KB summary
   - conversation list
   - document list
   - main pane

### Success Criteria

1. Opening `/` renders the new two-pane layout.
2. If KBs exist, one is selected on initial load.
3. If no KBs exist, the page shows onboarding instead of an error.
4. Existing `/kbs/{kbID}` and `/kbs/{kbID}/conversations/{conversationID}` routes still function.

### Verification Steps

1. Run the server.
2. Open `/`.
3. Confirm the new shell is visible.
4. Open `/kbs/{kbID}` and confirm the old page still renders.
5. Open `/kbs/{kbID}/conversations/{conversationID}` and confirm the old page still renders.

### Regression Checks

1. KB creation from old UI still works.
2. Chat page still works.

### Exit Gate

Pass only if:

1. workspace shell exists and renders
2. no existing page is broken

## Phase 2: KB Switching In Sidebar

### Objective

Make the left sidebar KB list control workspace state.

### Scope In

1. KB list partial
2. active KB summary partial
3. query-param driven active KB switching

### Scope Out

1. conversation switching
2. chat pane behavior
3. document actions

### Files To Create Or Modify

1. `internal/http/router.go`
2. `internal/http/handlers/workspace_handler.go`
3. `internal/web/templates/workspace.templ`
4. `internal/web/templates/components/kb_list.templ`
5. `internal/web/templates/components/kb_summary.templ`

### Backend Changes

1. Add sidebar partial endpoints for KB list and KB summary.
2. Resolve active KB using the state contract.

### Frontend Changes

1. KB clicks update URL query param `kb`.
2. KB switch refreshes:
   - KB summary
   - conversation list
   - documents panel
   - main pane placeholder

### Success Criteria

1. Clicking a KB updates the workspace without a full page reload.
2. The active KB summary always matches the selected KB.
3. Reloading the page preserves the selected KB.

### Verification Steps

1. Create at least 2 KBs.
2. Switch between them from the sidebar.
3. Confirm the URL changes.
4. Reload and confirm the same KB remains selected.

### Regression Checks

1. Old KB routes still work.
2. KB creation still works.

### Exit Gate

Pass only if:

1. KB switching is stable
2. active KB summary is correct after reload and switching

## Phase 3: Conversation List And Selection

### Objective

Move conversation creation and selection into the workspace sidebar.

### Scope In

1. conversation list partial
2. new conversation action in sidebar
3. active conversation selection via query param

### Scope Out

1. assistant streaming in workspace
2. document panel migration

### Files To Create Or Modify

1. `internal/http/router.go`
2. `internal/http/handlers/workspace_handler.go`
3. `internal/http/handlers/conversation_handler.go`
4. `internal/web/templates/components/conversation_list.templ`
5. `internal/web/templates/components/chat_header.templ`
6. `internal/web/templates/components/message_timeline.templ`

### Backend Changes

1. Add conversation list partial endpoint for active KB.
2. Add workspace-friendly conversation create response path.
3. Apply state contract for invalid/missing conversation IDs.

### Frontend Changes

1. New conversation action updates the sidebar list.
2. Selecting a conversation updates `conversation` query param.
3. Main pane loads either:
   - selected conversation
   - empty conversation state

### Success Criteria

1. User can create a conversation from the workspace.
2. Conversation appears immediately in sidebar.
3. Clicking a conversation updates the main pane.
4. Refresh preserves both KB and conversation selection.

### Verification Steps

1. Select a KB.
2. Create a conversation.
3. Confirm the sidebar updates.
4. Click the conversation.
5. Reload and confirm the same conversation remains active.

### Regression Checks

1. Old conversation routes still work.
2. Old standalone chat page still works.

### Exit Gate

Pass only if:

1. conversation list is fully usable from the workspace
2. active conversation state is stable

## Phase 4: Main Chat Pane Migration

### Objective

Make the workspace main pane the primary chat surface.

### Scope In

1. active conversation header
2. message timeline in workspace
3. chat composer in workspace
4. user message submission from workspace

### Scope Out

1. final deprecation of old chat route
2. document panel migration

### Files To Create Or Modify

1. `internal/http/router.go`
2. `internal/http/handlers/workspace_handler.go`
3. `internal/http/handlers/chat_handler.go`
4. `internal/web/templates/components/chat_header.templ`
5. `internal/web/templates/components/message_timeline.templ`
6. `internal/web/templates/components/message_item.templ`
7. `internal/web/templates/components/chat_composer.templ`

### Backend Changes

1. Add main-pane partial endpoint for active chat content.
2. Reuse existing message APIs where possible.

### Frontend Changes

1. Main pane renders message timeline for selected conversation.
2. Composer submits user messages from the workspace.
3. User message appears immediately after submit.

### Success Criteria

1. User can fully conduct chat from the workspace page.
2. Message history loads in the main pane.
3. No navigation to old standalone chat page is required for normal use.

### Verification Steps

1. Select KB and conversation.
2. Send a user message.
3. Confirm it appears in the main pane.

### Regression Checks

1. standalone chat route still works
2. assistant generation backend still behaves as before

### Exit Gate

Pass only if:

1. workspace chat is usable for user message flow
2. message history renders correctly

## Phase 5: SSE Assistant Streaming In Workspace

### Objective

Move assistant streaming into the workspace chat pane.

### Scope In

1. SSE streaming in workspace
2. assistant placeholder
3. final assistant persistence reflected in workspace

### Scope Out

1. citation collapse pattern
2. old route removal

### Files To Create Or Modify

1. `internal/http/handlers/chat_handler.go`
2. `internal/http/middleware/logging.go`
3. `internal/web/templates/components/message_timeline.templ`
4. `internal/web/templates/components/message_item.templ`
5. `internal/web/templates/components/chat_composer.templ`

### Backend Changes

1. Reuse current `/stream` SSE route or expose a workspace-oriented equivalent.
2. Ensure middleware preserves `http.Flusher`.

### Frontend Changes

1. Workspace composer triggers SSE stream.
2. Assistant placeholder is inserted immediately.
3. Final assistant answer replaces live placeholder.
4. Reload shows persisted assistant answer.

### Success Criteria

1. Assistant tokens visibly stream in the main pane.
2. Final answer persists after reload.
3. No false stream-failure state appears on normal completion.

### Verification Steps

1. Send a message from workspace.
2. Observe token streaming.
3. Reload page.
4. Confirm final assistant answer remains.

### Regression Checks

1. direct stream route still works
2. citations still persist

### Exit Gate

Pass only if:

1. streaming is stable
2. persisted assistant messages are consistent with streamed output

## Phase 6: Sidebar Sources Panel Migration

### Objective

Move uploads and document management into the workspace sidebar.

### Scope In

1. compact sources panel in sidebar
2. upload control
3. refresh/delete/reindex actions
4. document status updates in place

### Scope Out

1. legacy route removal
2. advanced evidence UI

### Files To Create Or Modify

1. `internal/http/router.go`
2. `internal/http/handlers/document_handler.go`
3. `internal/http/handlers/ingest_handler.go`
4. `internal/http/handlers/workspace_handler.go`
5. `internal/web/templates/components/document_list.templ`
6. `internal/web/templates/workspace.templ`

### Backend Changes

1. Add sidebar document-list partial endpoint.
2. Keep current document APIs.
3. Keep SSE ingestion endpoints but stop presenting jobs as a separate page section.

### Frontend Changes

1. Upload files from the sources panel.
2. Refresh and delete documents in place.
3. Re-index all from the sources panel.
4. Reflect progress through document row status and inline messaging.

### Success Criteria

1. Upload works from sidebar.
2. Document rows update status correctly.
3. Refresh/delete/reindex remain functional.
4. No visible ingestion jobs panel exists in the workspace.

### Verification Steps

1. Upload a file.
2. Wait for it to become ready.
3. Refresh one document.
4. Delete one document.
5. Re-index all.

### Regression Checks

1. retrieval still works after uploads and re-index
2. no document action breaks conversation/chat behavior

### Exit Gate

Pass only if:

1. sources panel fully replaces the old documents management flow for normal use
2. document lifecycle actions remain correct

## Phase 7: Evidence UX Refinement

### Objective

Make citations clear, expandable, and less noisy.

### Scope In

1. inline citation markers
2. expandable sources block
3. explicit insufficient-evidence message treatment

### Scope Out

1. legacy route removal

### Files To Create Or Modify

1. `internal/http/handlers/chat_handler.go`
2. `internal/web/templates/components/message_item.templ`
3. `internal/web/templates/components/citation_block.templ`

### Backend Changes

1. No major API change expected.
2. Keep citation payload contract stable.

### Frontend Changes

1. Show inline markers in assistant answers.
2. Render expandable sources section.
3. Keep excerpts hidden until expanded.

### Success Criteria

1. Grounded answers show citations without overwhelming the answer body.
2. Evidence excerpts are accessible in one click/tap.
3. Weak retrieval cases show an explicit insufficient-evidence response.

### Verification Steps

1. Ask a well-grounded question.
2. Confirm citations appear.
3. Expand evidence.
4. Ask an unsupported question.
5. Confirm insufficient-evidence response appears clearly.

### Regression Checks

1. citations still persist to SQLite
2. assistant answer remains readable with or without citations

### Exit Gate

Pass only if:

1. evidence is both visible and unobtrusive

## Phase 8: Legacy Page Demotion And Cleanup

### Objective

Retire the old multi-page UI as the primary experience.

### Scope In

1. redirect or demote old KB detail/chat pages
2. remove dead inline UI code no longer used
3. update docs to reflect the workspace as the primary UI

### Scope Out

1. future enhancements

### Files To Create Or Modify

1. `internal/http/router.go`
2. `internal/http/handlers/kb_handler.go`
3. `internal/http/handlers/chat_handler.go`
4. `README.md`
5. `docs/PROJECT_SPEC.md`
6. `docs/UI_REDESIGN_SPEC.md`

### Backend Changes

1. redirect `/kbs/{kbID}` to `/?kb=<KB_ID>`
2. redirect `/kbs/{kbID}/conversations/{conversationID}` to `/?kb=<KB_ID>&conversation=<CONVERSATION_ID>`
3. keep APIs intact

### Frontend Changes

1. remove dead standalone-page templates or mark them compatibility-only

### Success Criteria

1. The workspace is the clear primary UI.
2. Old routes do not create split-brain navigation.
3. Documentation reflects the new primary workflow.

### Verification Steps

1. Open old KB route.
2. Confirm redirect or compatibility behavior.
3. Open old conversation route.
4. Confirm redirect or compatibility behavior.
5. Read README and confirm setup/run instructions reference the workspace.

### Regression Checks

1. all API routes still function
2. direct links into KB/conversation state still work

### Exit Gate

Pass only if:

1. workspace route is the official primary UX
2. docs and behavior are aligned

## Final Acceptance Criteria

The redesign migration is complete only when all of these are true:

1. the primary user experience is a single workspace page
2. KB switching happens in the sidebar
3. conversation switching happens in the sidebar
4. documents are managed from the sidebar
5. chat happens in the main pane
6. assistant responses stream in the main pane
7. citations are visible and expandable
8. active KB context is always obvious
9. the UI uses Templ + HTMX + Alpine.js with SSE where required
10. the product feels like an AI knowledge assistant rather than a CRUD dashboard
