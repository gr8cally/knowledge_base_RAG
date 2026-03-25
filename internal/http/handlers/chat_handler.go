package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/domain"
)

type chatService interface {
	GetConversation(ctx context.Context, kbID, conversationID string) (*domain.Conversation, error)
	ListMessages(ctx context.Context, kbID, conversationID string) ([]app.ChatMessageView, error)
	AddUserMessage(ctx context.Context, kbID, conversationID, content string) (domain.Message, error)
	StreamAssistant(ctx context.Context, kbID, conversationID, userMessageID string, stream func(app.ChatStreamEvent) error) error
}

type kbLookup interface {
	Get(ctx context.Context, id string) (*domain.KnowledgeBase, error)
}

type ChatHandler struct {
	service   chatService
	kbService kbLookup
}

func NewChatHandler(service chatService, kbService kbLookup) *ChatHandler {
	return &ChatHandler{service: service, kbService: kbService}
}

func (h *ChatHandler) Page(w http.ResponseWriter, r *http.Request) {
	kbID := r.PathValue("kbID")
	conversationID := r.PathValue("conversationID")

	kb, err := h.kbService.Get(r.Context(), kbID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "get_kb_failed", err.Error())
		return
	}
	if kb == nil {
		http.NotFound(w, r)
		return
	}
	conv, err := h.service.GetConversation(r.Context(), kbID, conversationID)
	if err != nil {
		if errors.Is(err, app.ErrConversationNotFound) || errors.Is(err, app.ErrKnowledgeBaseNotFound) {
			http.NotFound(w, r)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "get_conversation_failed", err.Error())
		return
	}

	tpl := template.Must(template.New("chat-page").Parse(chatPageHTML))
	data := struct {
		KB           *domain.KnowledgeBase
		Conversation *domain.Conversation
		Now          string
	}{
		KB:           kb,
		Conversation: conv,
		Now:          time.Now().UTC().Format(time.RFC1123),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.Execute(w, data)
}

func (h *ChatHandler) MessagesAPI(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListMessages(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"))
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound), errors.Is(err, app.ErrConversationNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusInternalServerError, "list_messages_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *ChatHandler) PostMessageAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	msg, err := h.service.AddUserMessage(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"), req.Content)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrKnowledgeBaseNotFound), errors.Is(err, app.ErrConversationNotFound):
			http.NotFound(w, r)
		default:
			writeAPIError(w, http.StatusBadRequest, "post_message_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"message":    msg,
		"status":     "accepted",
		"stream_url": fmt.Sprintf("/api/kbs/%s/conversations/%s/stream?message_id=%s", r.PathValue("kbID"), r.PathValue("conversationID"), msg.ID),
	})
}

func (h *ChatHandler) StreamAPI(w http.ResponseWriter, r *http.Request) {
	userMessageID := r.URL.Query().Get("message_id")
	if userMessageID == "" {
		writeAPIError(w, http.StatusBadRequest, "missing_message_id", "message_id is required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "sse_not_supported", "response writer does not support streaming")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	_ = h.service.StreamAssistant(r.Context(), r.PathValue("kbID"), r.PathValue("conversationID"), userMessageID, func(event app.ChatStreamEvent) error {
		if err := writeChatSSE(w, event); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})
}

func writeChatSSE(w http.ResponseWriter, event app.ChatStreamEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: assistant\ndata: %s\n\n", payload)
	return err
}

const chatPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{ .Conversation.Title }}</title>
  <style>
    body { font-family: sans-serif; margin: 2rem; background: #f6f1e8; color: #1f1a17; }
    .shell { max-width: 960px; margin: 0 auto; }
    .card { background: #fffdf8; border: 1px solid #d9cbb5; border-radius: 14px; padding: 1rem 1.25rem; margin-bottom: 1rem; }
    .muted { color: #6f6256; }
    a { color: #8d3d1f; text-decoration: none; }
    .timeline { display: grid; gap: 0.75rem; }
    .msg { border: 1px solid #eadfcd; border-radius: 12px; padding: 0.85rem 1rem; background: #fff; }
    .msg.user { background: #f0e5d3; }
    .meta { font-size: 0.85rem; color: #6f6256; margin-bottom: 0.35rem; text-transform: uppercase; }
    textarea { width: 100%; min-height: 100px; }
    button { background: #8d3d1f; color: #fff; border: 0; border-radius: 10px; padding: 0.7rem 1rem; cursor: pointer; }
    code { background: #f0e5d3; padding: 0.1rem 0.3rem; border-radius: 6px; }
  </style>
</head>
<body data-kb-id="{{ .KB.ID }}" data-conversation-id="{{ .Conversation.ID }}">
  <main class="shell">
    <section class="card">
      <p><a href="/kbs/{{ .KB.ID }}">Back to KB</a></p>
      <h1>{{ .Conversation.Title }}</h1>
      <p class="muted">{{ .KB.Name }} · {{ .Now }}</p>
    </section>
    <section class="card">
      <h2>Messages</h2>
      <div id="timeline" class="timeline muted">Loading messages…</div>
    </section>
    <section class="card">
      <h2>Send Message</h2>
      <form id="message-form">
        <textarea id="message-content" placeholder="Ask a question about this knowledge base"></textarea>
        <div style="margin-top: 0.75rem;">
          <button type="submit">Send</button>
        </div>
      </form>
      <p id="message-status" class="muted" style="min-height:1.5rem;"></p>
      <p class="muted">Answers stream from the server and are persisted with citations.</p>
    </section>
  </main>
  <script>
    const kbID = document.body.dataset.kbId;
    const conversationID = document.body.dataset.conversationId;
    const timeline = document.getElementById('timeline');
    const form = document.getElementById('message-form');
    const content = document.getElementById('message-content');
    const status = document.getElementById('message-status');

    function esc(value) {
      return String(value ?? '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
    }

    async function loadMessages() {
      const resp = await fetch('/api/kbs/' + kbID + '/conversations/' + conversationID + '/messages');
      if (!resp.ok) {
        timeline.textContent = 'Failed to load messages.';
        return;
      }
      const messages = await resp.json();
      if (!messages.length) {
        timeline.innerHTML = '<p class="muted">No messages yet.</p>';
        return;
      }
      timeline.innerHTML = messages.map(item => '<div class="msg ' + esc(item.message.role) + '">' +
        '<div class="meta">' + esc(item.message.role) + '</div>' +
        '<div>' + esc(item.message.content).replaceAll('\n', '<br />') + '</div>' +
        renderCitations(item.citations || []) +
      '</div>').join('');
    }

    function renderCitations(citations) {
      if (!citations.length) {
        return '';
      }
      return '<div class="muted" style="margin-top:0.75rem;"><strong>Sources</strong><br />' +
        citations.map(c => '[' + esc(c.citation_index) + '] ' + esc(c.source_label) + ' · ' + esc(c.excerpt)).join('<br />') +
      '</div>';
    }

    function appendUserMessage(message) {
      const empty = timeline.querySelector('p.muted');
      if (empty) {
        timeline.innerHTML = '';
      }
      timeline.insertAdjacentHTML('beforeend', '<div class="msg user">' +
        '<div class="meta">user</div>' +
        '<div>' + esc(message.content).replaceAll('\n', '<br />') + '</div>' +
      '</div>');
    }

    function ensureAssistantMessage() {
      let node = document.getElementById('assistant-live');
      if (node) {
        return node;
      }
      timeline.insertAdjacentHTML('beforeend', '<div id="assistant-live" class="msg assistant">' +
        '<div class="meta">assistant</div>' +
        '<div class="assistant-content"></div>' +
        '<div class="assistant-citations muted" style="margin-top:0.75rem;"></div>' +
      '</div>');
      return document.getElementById('assistant-live');
    }

    function streamAssistant(streamURL) {
      const live = ensureAssistantMessage();
      const contentNode = live.querySelector('.assistant-content');
      const citationsNode = live.querySelector('.assistant-citations');

      return new Promise((resolve, reject) => {
        const source = new EventSource(streamURL);
        let terminal = false;
        let sawPayload = false;

        source.addEventListener('assistant', (event) => {
          sawPayload = true;
          const payload = JSON.parse(event.data);
          if (payload.type === 'snapshot') {
            contentNode.textContent = '';
            citationsNode.innerHTML = '';
            status.textContent = 'Generating answer…';
            return;
          }
          if (payload.type === 'token') {
            contentNode.textContent += payload.content || '';
            return;
          }
          if (payload.type === 'completed') {
            terminal = true;
            contentNode.innerHTML = esc(payload.content || '').replaceAll('\n', '<br />');
            citationsNode.innerHTML = renderCitations(payload.citations || []);
            live.removeAttribute('id');
            status.textContent = 'Answer complete.';
            source.close();
            resolve(payload);
            return;
          }
          if (payload.type === 'error') {
            terminal = true;
            status.textContent = payload.error || 'Assistant response failed.';
            live.remove();
            source.close();
            reject(new Error(payload.error || 'Assistant response failed.'));
          }
        });

        source.onerror = () => {
          source.close();
          if (terminal) {
            return;
          }
          if (sawPayload && contentNode.textContent.trim()) {
            terminal = true;
            live.removeAttribute('id');
            status.textContent = 'Answer complete.';
            resolve({ content: contentNode.textContent });
            return;
          }
          live.remove();
          status.textContent = 'Assistant stream failed.';
          reject(new Error('Assistant stream failed.'));
        };
      });
    }

    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      if (!content.value.trim()) {
        status.textContent = 'Message is required.';
        return;
      }
      status.textContent = 'Saving message…';
      const resp = await fetch('/api/kbs/' + kbID + '/conversations/' + conversationID + '/messages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: content.value })
      });
      const payload = await resp.json().catch(() => null);
      if (!resp.ok) {
        status.textContent = payload?.message || 'Failed to save message.';
        return;
      }
      appendUserMessage(payload.message);
      content.value = '';
      status.textContent = 'Waiting for assistant…';
      try {
        await streamAssistant(payload.stream_url);
        await loadMessages();
      } catch (_) {
        // status text is already set in stream handling
      }
    });

    loadMessages();
  </script>
</body>
</html>`
