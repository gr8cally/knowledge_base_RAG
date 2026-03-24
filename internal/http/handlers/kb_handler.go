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
  </style>
</head>
<body>
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
      <p class="muted">Upload and indexing arrive in Phase 4A/4B.</p>
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
</body>
</html>`
