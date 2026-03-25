package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/http/dto"
)

type KBHandler struct {
	service *app.KnowledgeBaseService
}

func NewKBHandler(service *app.KnowledgeBaseService) *KBHandler {
	return &KBHandler{service: service}
}

func (h *KBHandler) Index(w http.ResponseWriter, r *http.Request) {
	kbs, err := h.service.List(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	tpl := template.Must(template.New("kb-list").Parse(kbListHTML))
	data := struct {
		KnowledgeBases any
		Now            string
	}{
		KnowledgeBases: kbs,
		Now:            time.Now().UTC().Format(time.RFC1123),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.Execute(w, data)
}

func (h *KBHandler) Detail(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("kbID")
	kb, err := h.service.Get(r.Context(), kbID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	if kb == nil {
		http.NotFound(w, r)
		return
	}

	tpl := template.Must(template.New("kb-detail").Parse(kbDetailHTML))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.Execute(w, kb)
}

func (h *KBHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	kbs, err := h.service.List(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	resp := make([]dto.KnowledgeBaseResponse, 0, len(kbs))
	for _, kb := range kbs {
		resp = append(resp, dto.NewKnowledgeBaseResponse(kb))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *KBHandler) CreateAPI(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateKnowledgeBaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	kb, err := h.service.Create(r.Context(), req.Name, req.Description)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, dto.NewKnowledgeBaseResponse(kb))
}

func (h *KBHandler) UpdateAPI(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateKnowledgeBaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	kb, err := h.service.Update(r.Context(), r.PathValue("kbID"), req.Name, req.Description)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	if kb == nil {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, dto.NewKnowledgeBaseResponse(*kb))
}

func (h *KBHandler) ArchiveAPI(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Archive(r.Context(), r.PathValue("kbID")); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "archive_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, dto.ErrorResponse{Code: code, Message: message})
}

const kbListHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Knowledge Bases</title>
  <style>
    body { font-family: sans-serif; margin: 2rem; background: #f6f1e8; color: #1f1a17; }
    .shell { max-width: 960px; margin: 0 auto; }
    .card { background: #fffdf8; border: 1px solid #d9cbb5; border-radius: 14px; padding: 1rem 1.25rem; margin-bottom: 1rem; }
    .row { display: flex; justify-content: space-between; gap: 1rem; align-items: center; }
    .muted { color: #6f6256; }
    a { color: #8d3d1f; text-decoration: none; }
    button { background: #8d3d1f; color: #fff; border: 0; border-radius: 10px; padding: 0.7rem 1rem; cursor: pointer; }
    input, textarea { width: 100%; box-sizing: border-box; padding: 0.7rem 0.8rem; border: 1px solid #d9cbb5; border-radius: 10px; background: #fff; }
    textarea { min-height: 88px; resize: vertical; }
    .stack { display: grid; gap: 0.75rem; }
    .message { min-height: 1.5rem; }
    code { background: #f0e5d3; padding: 0.1rem 0.3rem; border-radius: 6px; }
  </style>
</head>
<body>
  <main class="shell">
    <section class="card">
      <div class="row">
        <div>
          <h1>Knowledge Bases</h1>
          <p class="muted">Create and switch between isolated knowledge bases.</p>
        </div>
        <div class="muted">{{ .Now }}</div>
      </div>
    </section>
    <section class="card">
      <h2>Create Knowledge Base</h2>
      <form id="kb-create-form" class="stack">
        <input id="kb-name" type="text" placeholder="Knowledge base name" />
        <textarea id="kb-description" placeholder="Short description"></textarea>
        <div>
          <button type="submit">Create KB</button>
        </div>
      </form>
      <p id="kb-create-message" class="message muted"></p>
    </section>
    {{ if .KnowledgeBases }}
      {{ range .KnowledgeBases }}
      <section class="card">
        <div class="row">
          <div>
            <h2><a href="/kbs/{{ .ID }}">{{ .Name }}</a></h2>
            <p class="muted">{{ .Description }}</p>
            <small class="muted">Namespace: {{ .Namespace }}</small>
          </div>
          <a href="/kbs/{{ .ID }}">Open</a>
        </div>
      </section>
      {{ end }}
    {{ else }}
      <section class="card">
        <p>No knowledge bases yet.</p>
      </section>
    {{ end }}
  </main>
  <script>
    const createForm = document.getElementById('kb-create-form');
    const createName = document.getElementById('kb-name');
    const createDescription = document.getElementById('kb-description');
    const createMessage = document.getElementById('kb-create-message');

    createForm.addEventListener('submit', async (event) => {
      event.preventDefault();
      createMessage.textContent = 'Creating knowledge base…';
      const resp = await fetch('/api/kbs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: createName.value,
          description: createDescription.value
        })
      });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok) {
        createMessage.textContent = payload?.message || 'Failed to create knowledge base.';
        return;
      }
      window.location.href = '/kbs/' + payload.id;
    });
  </script>
</body>
</html>`

const kbDetailHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{ .Name }}</title>
  <style>
    body { font-family: sans-serif; margin: 2rem; background: #f6f1e8; color: #1f1a17; }
    .shell { max-width: 960px; margin: 0 auto; }
    .card { background: #fffdf8; border: 1px solid #d9cbb5; border-radius: 14px; padding: 1rem 1.25rem; margin-bottom: 1rem; }
    .muted { color: #6f6256; }
    a { color: #8d3d1f; text-decoration: none; }
    button { background: #8d3d1f; color: #fff; border: 0; border-radius: 10px; padding: 0.7rem 1rem; cursor: pointer; }
    button.secondary { background: #f0e5d3; color: #1f1a17; border: 1px solid #d9cbb5; }
    .actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
    .hero { background: linear-gradient(135deg, #fffaf0 0%, #f3e6d1 100%); }
    .hero-actions { margin-top: 1rem; display: flex; gap: 0.75rem; flex-wrap: wrap; }
    .section-head { display: flex; justify-content: space-between; gap: 1rem; align-items: flex-end; margin-bottom: 0.9rem; }
    .section-head p { margin: 0.2rem 0 0 0; }
    .conversation-list { display: grid; gap: 0.85rem; margin-top: 1rem; }
    .conversation-item { border: 1px solid #eadfcd; border-radius: 12px; padding: 0.9rem 1rem; background: #fff; display: flex; justify-content: space-between; gap: 1rem; align-items: flex-start; }
    .conversation-main { min-width: 0; }
    .conversation-title { font-weight: 700; margin-bottom: 0.25rem; }
    .conversation-meta { font-size: 0.9rem; color: #6f6256; }
    .conversation-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
    input[type="text"] { width: 100%; box-sizing: border-box; padding: 0.7rem 0.8rem; border: 1px solid #d9cbb5; border-radius: 10px; background: #fff; }
    input[type="file"] { width: 100%; }
    .status { display: inline-block; padding: 0.2rem 0.5rem; border-radius: 999px; background: #f0e5d3; }
    .status.ready, .status.completed { background: #d7f0d8; color: #20542a; }
    .status.processing, .status.running, .status.queued { background: #fde7b2; color: #694d00; }
    .status.error, .status.failed { background: #f7d8d8; color: #7a1f1f; }
    .stack { display: grid; gap: 1rem; }
    .message { margin-top: 0.75rem; min-height: 1.5rem; }
    .doc-list { display: grid; gap: 0.75rem; margin-top: 1rem; }
    .doc-item { border: 1px solid #eadfcd; border-radius: 12px; padding: 0.85rem 1rem; background: #fff; display: flex; justify-content: space-between; gap: 1rem; align-items: flex-start; }
    .doc-main { min-width: 0; }
    .doc-title { font-weight: 700; margin-bottom: 0.3rem; }
    .doc-meta { font-size: 0.9rem; color: #6f6256; display: grid; gap: 0.2rem; }
    .doc-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
    .empty { padding: 1rem 0; color: #6f6256; }
    code { background: #f0e5d3; padding: 0.1rem 0.3rem; border-radius: 6px; }
  </style>
</head>
<body data-kb-id="{{ .ID }}">
  <main class="shell">
    <section class="card hero">
      <p><a href="/">Back</a></p>
      <h1>{{ .Name }}</h1>
      <p class="muted">{{ .Description }}</p>
      <small class="muted">Namespace: {{ .Namespace }}</small>
      <div class="hero-actions">
        <button id="hero-new-conversation" type="button">New Conversation</button>
      </div>
    </section>
    <section class="card">
      <div class="section-head">
        <div>
          <h2>Conversations</h2>
          <p class="muted">Start with questions and resume previous threads here.</p>
        </div>
      </div>
      <div class="stack">
        <form id="conversation-form">
          <input id="conversation-title" type="text" placeholder="Conversation title" />
          <div style="margin-top: 0.75rem;">
            <button type="submit">New Conversation</button>
          </div>
        </form>
        <div id="conversation-message" class="message muted"></div>
      </div>
      <div id="conversations-table" class="muted">Loading conversations…</div>
    </section>
    <section class="card">
      <div class="section-head">
        <div>
          <h2>Documents</h2>
          <p class="muted">Manage the source files that ground conversation answers.</p>
        </div>
      </div>
      <div class="stack">
        <form id="upload-form">
          <input id="upload-files" type="file" name="files" multiple />
          <div style="margin-top: 0.75rem;">
            <button type="submit">Upload and Index</button>
            <button id="reindex-all" class="secondary" type="button">Re-index All</button>
          </div>
        </form>
        <div id="upload-message" class="message muted"></div>
      </div>
      <div id="documents-table" class="muted">Loading documents…</div>
    </section>
  </main>
  <script>
    const kbID = document.body.dataset.kbId;
    const uploadForm = document.getElementById('upload-form');
    const uploadFiles = document.getElementById('upload-files');
    const uploadMessage = document.getElementById('upload-message');
    const documentsTable = document.getElementById('documents-table');
    const reindexAllButton = document.getElementById('reindex-all');
    const heroNewConversation = document.getElementById('hero-new-conversation');
    const conversationForm = document.getElementById('conversation-form');
    const conversationTitle = document.getElementById('conversation-title');
    const conversationMessage = document.getElementById('conversation-message');
    const conversationsTable = document.getElementById('conversations-table');

    function statusClass(value) {
      return 'status ' + String(value || '').toLowerCase();
    }

    function esc(value) {
      return String(value ?? '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
    }

    function formatDateTime(value) {
      if (!value) {
        return '-';
      }
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return esc(value);
      }
      return new Intl.DateTimeFormat(undefined, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: 'numeric',
        minute: '2-digit'
      }).format(date);
    }

    async function refreshDocuments() {
      const resp = await fetch('/api/kbs/' + kbID + '/documents');
      if (!resp.ok) {
        documentsTable.textContent = 'Failed to load documents.';
        return;
      }
      const docs = await resp.json();
      if (!docs.length) {
        documentsTable.innerHTML = '<div class="empty">Upload documents to ground answers in this knowledge base.</div>';
        return;
      }
      documentsTable.innerHTML = '<div class="doc-list">' +
        docs.map(doc => '<div class="doc-item">' +
          '<div class="doc-main">' +
            '<div class="doc-title">' + esc(doc.display_name) + ' <span class="' + statusClass(doc.status) + '">' + esc(doc.status) + '</span></div>' +
            '<div class="doc-meta">' +
              '<div>Parser: ' + esc(doc.parser_used || 'pending') + ' · Chunks: ' + esc(doc.chunk_count) + '</div>' +
              (doc.error_message ? '<div>Error: ' + esc(doc.error_message) + '</div>' : '') +
            '</div>' +
          '</div>' +
          '<div class="doc-actions">' +
            '<button class="secondary" type="button" data-action="refresh" data-document-id="' + esc(doc.id) + '">Refresh</button>' +
            '<button class="secondary" type="button" data-action="delete" data-document-id="' + esc(doc.id) + '">Delete</button>' +
          '</div>' +
        '</div>').join('') +
      '</div>';
    }

    async function refreshConversations() {
      const resp = await fetch('/api/kbs/' + kbID + '/conversations');
      if (!resp.ok) {
        conversationsTable.textContent = 'Failed to load conversations.';
        return;
      }
      const items = await resp.json();
      if (!items.length) {
        conversationsTable.innerHTML = '<div class="empty">Start your first conversation to ask questions about this knowledge base.</div>';
        return;
      }
      conversationsTable.innerHTML = '<div class="conversation-list">' +
        items.map(item => '<div class="conversation-item">' +
          '<div class="conversation-main">' +
            '<div class="conversation-title"><a href="/kbs/' + kbID + '/conversations/' + esc(item.id) + '">' + esc(item.title) + '</a></div>' +
            '<div class="conversation-meta">Last active ' + esc(formatDateTime(item.last_message_at || item.updated_at || item.created_at)) + '</div>' +
          '</div>' +
          '<div class="conversation-actions">' +
            '<button class="secondary" type="button" data-conversation-action="open" data-conversation-id="' + esc(item.id) + '">Open</button>' +
            '<button class="secondary" type="button" data-conversation-action="archive" data-conversation-id="' + esc(item.id) + '">Archive</button>' +
          '</div>' +
        '</div>').join('') +
      '</div>';
    }

    function watchJob(jobID) {
      const stream = new EventSource('/api/kbs/' + kbID + '/ingestion-jobs/' + jobID + '/events');
      stream.addEventListener('job', async (event) => {
        const payload = JSON.parse(event.data);
        uploadMessage.textContent = (payload.message || payload.type) + ' (' + payload.job.status + ')';
        await refreshDocuments();
        if (payload.job.status === 'completed' || payload.job.status === 'failed') {
          stream.close();
        }
      });
      stream.onerror = () => stream.close();
    }

    uploadForm.addEventListener('submit', async (event) => {
      event.preventDefault();
      if (!uploadFiles.files.length) {
        uploadMessage.textContent = 'Choose at least one file.';
        return;
      }

      const body = new FormData();
      for (const file of uploadFiles.files) {
        body.append('files', file, file.name);
      }

      uploadMessage.textContent = 'Uploading…';
      const resp = await fetch('/api/kbs/' + kbID + '/documents/upload', { method: 'POST', body });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok) {
        uploadMessage.textContent = payload?.message || 'Upload failed.';
        await refreshDocuments();
        return;
      }

      const jobs = (payload || []).map(item => item.job).filter(Boolean);
      const skipped = (payload || []).filter(item => item.skipped);
      uploadMessage.textContent = skipped.length && !jobs.length ? skipped[0].notice : 'Upload accepted.';
      await refreshDocuments();
      for (const job of jobs) {
        if (job.status === 'queued' || job.status === 'running') {
          watchJob(job.id);
        }
      }
      uploadForm.reset();
    });

    heroNewConversation.addEventListener('click', () => {
      conversationTitle.focus();
      conversationTitle.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });

    documentsTable.addEventListener('click', async (event) => {
      const button = event.target.closest('button[data-action]');
      if (!button) {
        return;
      }
      const documentID = button.dataset.documentId;
      const action = button.dataset.action;
      if (action === 'delete' && !window.confirm('Delete this document and its indexed chunks?')) {
        return;
      }

      button.disabled = true;
      try {
        if (action === 'refresh') {
          uploadMessage.textContent = 'Refreshing document…';
          const resp = await fetch('/api/kbs/' + kbID + '/documents/' + documentID + '/refresh', { method: 'POST' });
          const payload = await resp.json().catch(() => null);
          if (!resp.ok) {
            uploadMessage.textContent = payload?.message || 'Refresh failed.';
          } else {
            uploadMessage.textContent = 'Refresh accepted.';
            if (payload?.job?.id) {
              watchJob(payload.job.id);
            }
          }
        } else if (action === 'delete') {
          uploadMessage.textContent = 'Deleting document…';
          const resp = await fetch('/api/kbs/' + kbID + '/documents/' + documentID, { method: 'DELETE' });
          if (!resp.ok) {
            const payload = await resp.json().catch(() => null);
            uploadMessage.textContent = payload?.message || 'Delete failed.';
          } else {
            uploadMessage.textContent = 'Document deleted.';
          }
        }
      } finally {
        await Promise.all([refreshDocuments(), refreshConversations()]);
        button.disabled = false;
      }
    });

    reindexAllButton.addEventListener('click', async () => {
      reindexAllButton.disabled = true;
      uploadMessage.textContent = 'Starting re-index all…';
      try {
        const resp = await fetch('/api/kbs/' + kbID + '/reindex-all', { method: 'POST' });
        const payload = await resp.json().catch(() => null);
        if (!resp.ok) {
          uploadMessage.textContent = payload?.message || 'Re-index all failed.';
          return;
        }
        uploadMessage.textContent = 'Re-index all accepted.';
        if (payload?.id) {
          watchJob(payload.id);
        }
        await Promise.all([refreshDocuments(), refreshConversations()]);
      } finally {
        reindexAllButton.disabled = false;
      }
    });

    conversationForm.addEventListener('submit', async (event) => {
      event.preventDefault();
      conversationMessage.textContent = 'Creating conversation…';
      const resp = await fetch('/api/kbs/' + kbID + '/conversations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: conversationTitle.value })
      });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok) {
        conversationMessage.textContent = payload?.message || 'Failed to create conversation.';
        return;
      }
      conversationMessage.textContent = 'Conversation created.';
      conversationForm.reset();
      await refreshConversations();
      window.location.href = '/kbs/' + kbID + '/conversations/' + payload.id;
    });

    conversationsTable.addEventListener('click', async (event) => {
      const button = event.target.closest('button[data-conversation-action]');
      if (!button) {
        return;
      }
      const action = button.dataset.conversationAction;
      const conversationID = button.dataset.conversationId;
      if (action === 'open') {
        window.location.href = '/kbs/' + kbID + '/conversations/' + conversationID;
        return;
      }
      if (action === 'archive' && !window.confirm('Archive this conversation?')) {
        return;
      }
      const resp = await fetch('/api/kbs/' + kbID + '/conversations/' + conversationID, { method: 'DELETE' });
      if (!resp.ok) {
        const payload = await resp.json().catch(() => null);
        conversationMessage.textContent = payload?.message || 'Failed to archive conversation.';
        return;
      }
      conversationMessage.textContent = 'Conversation archived.';
      await refreshConversations();
    });

    Promise.all([refreshDocuments(), refreshConversations()]);
  </script>
</body>
</html>`
