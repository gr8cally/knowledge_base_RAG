package handlers

import (
	"encoding/json"
	"net/http"

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
	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	http.Redirect(w, r, "/?kb="+kb.ID, http.StatusSeeOther)
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
