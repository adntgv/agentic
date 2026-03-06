package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ArtifactHandler struct {
	artifactSvc *service.ArtifactService
}

func NewArtifactHandler(artifactSvc *service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{artifactSvc: artifactSvc}
}

// UploadURL handles POST /tasks/{id}/artifacts/upload-url
func (h *ArtifactHandler) UploadURL(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	var req service.ArtifactUploadRequest
	json.NewDecoder(r.Body).Decode(&req)

	resp, err := h.artifactSvc.RequestUploadURL(r.Context(), taskID, workerID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// List handles GET /tasks/{id}/artifacts
func (h *ArtifactHandler) List(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	contextFilter := r.URL.Query().Get("context")

	artifacts, _ := h.artifactSvc.List(r.Context(), taskID, contextFilter)
	json.NewEncoder(w).Encode(artifacts)
}
