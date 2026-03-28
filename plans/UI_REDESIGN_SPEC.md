# UI Redesign Specification

## Purpose

This document replaces the current multi-page, admin-leaning UI direction with a single-workspace interface that better satisfies the assignment brief requirement to include a UI with best-practices in AI UX.

This redesign keeps the current backend domain model and APIs, but changes the frontend interaction model to make conversation the primary workflow.

## Design Goals

1. Make querying the knowledge base the primary interaction.
2. Keep KB switching, conversation switching, and document management on one screen.
3. Reduce navigation cost by removing separate KB detail and chat pages as the main UX path.
4. Use Templ + HTMX + Alpine.js as originally planned.
5. Preserve SSE-based chat streaming and ingestion progress.
6. Keep source grounding visible without making the interface noisy.

## Core Decision

The application should move to a single workspace page with two panes:

1. Left pane: KB selection, active KB summary, conversations, and document controls.
2. Right pane: active conversation header, message timeline, citations/evidence, and composer.

This replaces the current primary flow of:

1. KB list page
2. KB detail page
3. separate chat page

## V1 Non-Goals

1. No conversation search bar in v1.
2. No separate ingestion jobs panel in the visible UI.
3. No settings page/tab in the main UX.
4. No JavaScript build step.

## Primary Route

Use one main workspace route:

- `/`

Workspace state is controlled by query parameters:

- `?kb=<KB_ID>`
- `?kb=<KB_ID>&conversation=<CONVERSATION_ID>`

This gives:

1. direct-linkable state
2. browser back/forward compatibility
3. no need for separate KB detail and chat pages as the primary interaction path

Existing routes may remain for compatibility, but the main UI should be the workspace route.

## Layout Specification

### Desktop Layout

Use a two-pane layout:

1. Left pane width: 28% to 32%
2. Right pane width: 68% to 72%

The left pane should be visually lighter and more compact than the main pane.

### Mobile Layout

Stack vertically:

1. Top collapsible workspace navigation drawer
2. Full-width conversation pane below

On mobile:

1. KB list and conversations should be accessible through a drawer or collapsible sections
2. composer should remain pinned near the bottom

## Left Pane Specification

The left pane should contain four sections in this order.

### 1. App Header

Contents:

1. app name
2. short support text such as `Personal knowledge workspace`

### 2. Knowledge Bases

Contents:

1. KB list
2. active KB highlight
3. `New KB` action

Each KB row shows:

1. KB name
2. optional short description preview
3. active state

Interaction:

1. clicking a KB switches workspace state to that KB
2. if no conversation is selected, auto-select the most recent conversation if one exists

### 3. Active KB Summary

This is a compact summary card for the selected KB.

Contents:

1. KB name
2. short description
3. document count
4. conversation count
5. readiness summary such as:
   - `5 ready`
   - `1 processing`
   - `1 error`

Purpose:

1. make active retrieval scope obvious
2. prevent “wrong KB” mistakes
3. reinforce that answers are grounded in the currently selected corpus

### 4. Conversations

This is the most important left-pane section after KB switching.

Contents:

1. section heading
2. `New Conversation` action
3. list of conversations for the active KB sorted by `last_message_at DESC`

Each conversation row shows:

1. title
2. last activity timestamp
3. optional one-line preview of the latest message in a later iteration
4. active state
5. subtle archive action

V1 decision:

1. no conversation search bar

### 5. Sources

This is a compact document-management section, not a primary panel.

Contents:

1. multi-file upload control
2. `Re-index All` action
3. compact document list

Each document row shows:

1. file name
2. status chip
3. secondary metadata:
   - parser
   - chunk count
   - short error only if present
4. actions:
   - refresh
   - delete

This section must remain visually subordinate to conversations.

## Right Pane Specification

### 1. Conversation Header

Contents:

1. conversation title
2. active KB name
3. lightweight KB grounding note such as:
   - `Answering from 5 indexed documents`
4. optional rename conversation action

### 2. Message Timeline

Display:

1. user and assistant turns in chronological order
2. readable timestamps
3. clear visual distinction between user and assistant
4. narrow readable line lengths

Assistant messages must include:

1. answer content
2. inline citation markers
3. evidence block or evidence toggle

### 3. Evidence / Citations

Best-practice AI UX requirement:

1. every grounded answer should visibly connect to retrieved evidence

V1 pattern:

1. inline citation markers such as `[1] [2]`
2. expandable `Sources` block under the assistant answer
3. each citation entry shows:
   - source label
   - excerpt
   - optional score in a subtle format if retained

The evidence UI must be present but not always visually dominant.

### 4. Composer

The composer should be sticky at the bottom of the right pane.

Contents:

1. multiline input
2. send button
3. generation status line
4. disabled/loading state during active generation where appropriate

Behavior:

1. `Enter` sends
2. `Shift+Enter` adds newline
3. when streaming, the assistant placeholder appears immediately

## AI UX Requirements

The redesigned UI must satisfy these AI UX requirements.

### 1. Active Context Clarity

The user must always know:

1. which KB is active
2. which conversation is active
3. that answers are grounded in uploaded documents

### 2. Fast Perceived Response

Use streaming to:

1. show assistant output progressively
2. reduce perceived latency
3. keep the user oriented during long generations

### 3. Evidence Visibility

Every answer should include references to retrieved source documents.

The interface should make evidence:

1. visible
2. clickable/expandable
3. secondary to the answer, not louder than the answer

### 4. Guided Empty States

Empty states must guide the user.

Examples:

1. no KB selected:
   - `Create or select a knowledge base to begin`
2. no conversations:
   - `Start your first conversation for this knowledge base`
3. no documents:
   - `Upload documents to ground answers`
4. no retrieval evidence:
   - `I don't have enough evidence in this knowledge base to answer that`

### 5. Safe Scope Switching

Switching KBs should clearly change the retrieval context.

Recommended:

1. highlight active KB strongly
2. reset or change the conversation view when the active KB changes
3. never let the user think they are chatting against one KB while another is active

## HTMX / Alpine / SSE Architecture

The redesign should move back toward the original frontend stack decisions.

### Templ

Use Templ for:

1. main workspace shell
2. sidebar sections
3. conversation list partial
4. document list partial
5. message list partial
6. citation/evidence partial

### HTMX

Use HTMX for:

1. switching KB content without full page reload
2. switching conversations without full page reload
3. creating KBs
4. creating conversations
5. uploading documents
6. refreshing document/document list partials after actions
7. re-index-all action responses

### Alpine.js

Use Alpine for local state only:

1. mobile drawer open/close
2. evidence panel toggle
3. active section expansion
4. optimistic local composer state

Do not use Alpine to replace server-rendered data flows.

### SSE

Keep SSE for:

1. assistant token stream
2. ingestion progress

In the redesigned UI:

1. assistant SSE updates the message timeline in place
2. ingestion SSE updates only the relevant document/status areas
3. do not surface ingestion jobs as a separate visible table

## Screen States

### State A: No KBs

Left pane:

1. empty KB list
2. prominent `New KB`

Right pane:

1. onboarding message
2. short explanation of what the app does

### State B: KB Selected, No Documents

Left pane:

1. active KB summary
2. conversations section
3. sources section with upload emphasized

Right pane:

1. conversation area either disabled or shows:
   - `Upload documents to ground answers in this knowledge base`

### State C: KB Selected, Documents Ready, No Conversation

Right pane:

1. empty conversation onboarding
2. `Start a conversation`

### State D: Active Conversation

Right pane:

1. normal timeline
2. citations
3. sticky composer

## Interaction Flows

### Flow 1: Switch KB

1. user clicks KB in sidebar
2. HTMX updates active KB summary, conversation list, document panel, and main pane
3. if recent conversation exists, select it automatically or prompt user to choose

### Flow 2: Start Conversation

1. user clicks `New Conversation`
2. server creates conversation
3. sidebar conversation list refreshes
4. main pane loads the new empty conversation
5. composer gains focus

### Flow 3: Upload Document

1. user uploads file from sources section
2. upload control shows progress/acceptance state
3. documents list refreshes
4. status chip updates as ingestion progresses
5. no separate jobs panel is shown

### Flow 4: Ask Question

1. user submits message
2. user message appears immediately
3. assistant placeholder appears
4. SSE streams assistant tokens
5. completion swaps in final answer with citations

## Route and Partial Plan

### Workspace route

1. `GET /`
   - render workspace shell

### Suggested HTMX partial routes

1. `GET /ui/sidebar/kbs`
2. `GET /ui/sidebar/summary?kb=<KB_ID>`
3. `GET /ui/sidebar/conversations?kb=<KB_ID>`
4. `GET /ui/sidebar/documents?kb=<KB_ID>`
5. `GET /ui/main/chat?kb=<KB_ID>&conversation=<CONVERSATION_ID>`
6. `GET /ui/main/empty?kb=<KB_ID>`

These can be implemented as HTML partial endpoints or handled by existing handlers returning partial Templ fragments.

## Component Breakdown

Recommended Templ components:

1. `workspace.templ`
2. `components/kb_list.templ`
3. `components/kb_summary.templ`
4. `components/conversation_list.templ`
5. `components/document_list.templ`
6. `components/chat_header.templ`
7. `components/message_timeline.templ`
8. `components/message_item.templ`
9. `components/citation_block.templ`
10. `components/chat_composer.templ`

## Migration Plan

### Step 1

Introduce the new workspace shell on `/` while keeping current routes working.

### Step 2

Extract current inline HTML from handlers into Templ components.

### Step 3

Move KB switching, conversation switching, and document panel refreshes to HTMX partial swaps.

### Step 4

Embed the existing SSE chat stream inside the workspace chat pane.

### Step 5

Demote legacy standalone pages:

1. `/kbs/{kbID}`
2. `/kbs/{kbID}/conversations/{conversationID}`

These may redirect into the workspace route with query params.

## Acceptance Criteria

The redesign is complete when:

1. the primary user experience is a single workspace page
2. KB switching does not require page-hopping between admin-style screens
3. conversations are the dominant workflow
4. document management is accessible but secondary
5. assistant answers stream in place
6. citations are visible and expandable
7. the UI is implemented with Templ + HTMX + Alpine.js, with SSE retained for streaming
8. the experience feels like a knowledge assistant, not a CRUD dashboard
