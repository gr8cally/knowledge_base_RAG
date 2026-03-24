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
    code { background: #f0e5d3; padding: 0.1rem 0.3rem; border-radius: 6px; }
  </style>
</head>
<body>
  <main class="shell">
    <section class="card">
      <div class="row">
        <div>
          <h1>Knowledge Bases</h1>
          <p class="muted">Phase 3 dashboard shell. Create KBs via <code>POST /api/kbs</code>.</p>
        </div>
        <div class="muted">{{ .Now }}</div>
      </div>
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
    .tabs { display: flex; gap: 0.75rem; margin-top: 1rem; }
    .tab { padding: 0.5rem 0.75rem; border: 1px solid #d9cbb5; border-radius: 999px; background: #f0e5d3; }
    .muted { color: #6f6256; }
    a { color: #8d3d1f; text-decoration: none; }
    button { background: #8d3d1f; color: #fff; border: 0; border-radius: 10px; padding: 0.7rem 1rem; cursor: pointer; }
    input[type="file"] { width: 100%; }
    table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
    th, td { text-align: left; padding: 0.65rem 0.4rem; border-bottom: 1px solid #eadfcd; vertical-align: top; }
    .status { display: inline-block; padding: 0.2rem 0.5rem; border-radius: 999px; background: #f0e5d3; }
    .status.ready, .status.completed { background: #d7f0d8; color: #20542a; }
    .status.processing, .status.running, .status.queued { background: #fde7b2; color: #694d00; }
    .status.error, .status.failed { background: #f7d8d8; color: #7a1f1f; }
    .stack { display: grid; gap: 1rem; }
    .message { margin-top: 0.75rem; min-height: 1.5rem; }
    code { background: #f0e5d3; padding: 0.1rem 0.3rem; border-radius: 6px; }
  </style>
</head>
<body data-kb-id="{{ .ID }}">
  <main class="shell">
    <section class="card">
      <p><a href="/">Back</a></p>
      <h1>{{ .Name }}</h1>
      <p class="muted">{{ .Description }}</p>
      <small class="muted">Namespace: {{ .Namespace }}</small>
      <div class="tabs">
        <div class="tab">Documents</div>
        <div class="tab">Conversations</div>
        <div class="tab">Settings</div>
      </div>
    </section>
    <section class="card">
      <h2>Documents</h2>
      <div class="stack">
        <form id="upload-form">
          <input id="upload-files" type="file" name="files" multiple />
          <div style="margin-top: 0.75rem;">
            <button type="submit">Upload and Index</button>
          </div>
        </form>
        <div id="upload-message" class="message muted"></div>
      </div>
      <h3>Documents</h3>
      <div id="documents-table" class="muted">Loading documents…</div>
      <h3 style="margin-top: 1.5rem;">Ingestion Jobs</h3>
      <div id="jobs-table" class="muted">Loading jobs…</div>
    </section>
    <section class="card">
      <h2>Conversations</h2>
      <p class="muted">Conversation management arrives in Phase 6.</p>
    </section>
    <section class="card">
      <h2>Settings</h2>
      <p class="muted">Current namespace: {{ .Namespace }}</p>
    </section>
  </main>
  <script>
    const kbID = document.body.dataset.kbId;
    const uploadForm = document.getElementById('upload-form');
    const uploadFiles = document.getElementById('upload-files');
    const uploadMessage = document.getElementById('upload-message');
    const documentsTable = document.getElementById('documents-table');
    const jobsTable = document.getElementById('jobs-table');

    function statusClass(value) {
      return 'status ' + String(value || '').toLowerCase();
    }

    function esc(value) {
      return String(value ?? '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
    }

    async function refreshDocuments() {
      const resp = await fetch('/api/kbs/' + kbID + '/documents');
      if (!resp.ok) {
        documentsTable.textContent = 'Failed to load documents.';
        return;
      }
      const docs = await resp.json();
      if (!docs.length) {
        documentsTable.innerHTML = '<p class="muted">No documents yet.</p>';
        return;
      }
      documentsTable.innerHTML = '<table><thead><tr><th>Name</th><th>Status</th><th>Parser</th><th>Chunks</th><th>Error</th></tr></thead><tbody>' +
        docs.map(doc => '<tr>' +
          '<td>' + esc(doc.display_name) + '</td>' +
          '<td><span class="' + statusClass(doc.status) + '">' + esc(doc.status) + '</span></td>' +
          '<td>' + esc(doc.parser_used || '-') + '</td>' +
          '<td>' + esc(doc.chunk_count) + '</td>' +
          '<td class="muted">' + esc(doc.error_message || '') + '</td>' +
        '</tr>').join('') +
        '</tbody></table>';
    }

    async function refreshJobs() {
      const resp = await fetch('/api/kbs/' + kbID + '/ingestion-jobs');
      if (!resp.ok) {
        jobsTable.textContent = 'Failed to load ingestion jobs.';
        return;
      }
      const jobs = await resp.json();
      if (!jobs.length) {
        jobsTable.innerHTML = '<p class="muted">No ingestion jobs yet.</p>';
        return;
      }
      jobsTable.innerHTML = '<table><thead><tr><th>Job</th><th>Status</th><th>Processed</th><th>Skipped</th><th>Failed</th><th>Error</th></tr></thead><tbody>' +
        jobs.map(job => '<tr>' +
          '<td><code>' + esc(job.id) + '</code></td>' +
          '<td><span class="' + statusClass(job.status) + '">' + esc(job.status) + '</span></td>' +
          '<td>' + esc(job.processed_items) + '/' + esc(job.total_items) + '</td>' +
          '<td>' + esc(job.skipped_items) + '</td>' +
          '<td>' + esc(job.failed_items) + '</td>' +
          '<td class="muted">' + esc(job.error_message || '') + '</td>' +
        '</tr>').join('') +
        '</tbody></table>';
    }

    function watchJob(jobID) {
      const stream = new EventSource('/api/kbs/' + kbID + '/ingestion-jobs/' + jobID + '/events');
      stream.addEventListener('job', async (event) => {
        const payload = JSON.parse(event.data);
        uploadMessage.textContent = (payload.message || payload.type) + ' (' + payload.job.status + ')';
        await Promise.all([refreshDocuments(), refreshJobs()]);
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
        await Promise.all([refreshDocuments(), refreshJobs()]);
        return;
      }

      const jobs = (payload || []).map(item => item.job).filter(Boolean);
      const skipped = (payload || []).filter(item => item.skipped);
      uploadMessage.textContent = skipped.length && !jobs.length ? skipped[0].notice : 'Upload accepted.';
      await Promise.all([refreshDocuments(), refreshJobs()]);
      for (const job of jobs) {
        if (job.status === 'queued' || job.status === 'running') {
          watchJob(job.id);
        }
      }
      uploadForm.reset();
    });

    Promise.all([refreshDocuments(), refreshJobs()]);
  </script>
</body>
</html>`
