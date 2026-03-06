package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// WorkHandler handles work-related actions (ack, submit, approve, revision)
type WorkHandler struct {
	taskSvc *service.TaskService
}

func NewWorkHandler(taskSvc *service.TaskService) *WorkHandler {
	return &WorkHandler{taskSvc: taskSvc}
}

// Ack handles POST /tasks/{id}/ack
func (h *WorkHandler) Ack(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	task, err := h.taskSvc.Ack(r.Context(), taskID, workerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// Submit handles POST /tasks/{id}/submit
func (h *WorkHandler) Submit(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	var req struct {
		ArtifactIDs []uuid.UUID `json:"artifact_ids"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if len(req.ArtifactIDs) == 0 {
		http.Error(w, "at least one artifact required", http.StatusUnprocessableEntity)
		return
	}

	task, err := h.taskSvc.Submit(r.Context(), taskID, workerID, req.ArtifactIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// Approve handles POST /tasks/{id}/approve
func (h *WorkHandler) Approve(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	task, err := h.taskSvc.Approve(r.Context(), taskID, workerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// Revision handles POST /tasks/{id}/revision
func (h *WorkHandler) Revision(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	var req struct {
		Feedback []service.RevisionFeedback `json:"feedback"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	task, err := h.taskSvc.RequestRevision(r.Context(), taskID, workerID, req.Feedback)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(task)
}
