package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TaskHandler handles task-related HTTP endpoints
type TaskHandler struct {
	taskSvc *service.TaskService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(taskSvc *service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

// Create handles POST /tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract worker_id from auth context
	workerID := uuid.New() // placeholder

	var req service.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.taskSvc.Create(r.Context(), workerID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// List handles GET /tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse query filters (status, category, budget, deadline, worker_type)
	// TODO: Call taskSvc.List with filters

	json.NewEncoder(w).Encode([]interface{}{})
}

// Get handles GET /tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	// TODO: Call taskSvc.Get(taskID)

	json.NewEncoder(w).Encode(map[string]string{"id": taskID.String()})
}

// Update handles PATCH /tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	workerID := uuid.New() // from auth context

	var req struct {
		Status *string `json:"status"`
		// other updatable fields
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If status update to "published", call Publish
	if req.Status != nil && *req.Status == "published" {
		task, err := h.taskSvc.Publish(r.Context(), taskID, workerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(task)
		return
	}

	// TODO: Handle other updates

	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// Delete handles DELETE /tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// TODO: Only allow delete of draft tasks
	w.WriteHeader(http.StatusNoContent)
}

// Cancel handles POST /tasks/{id}/cancel
func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	workerID := uuid.New() // from auth context

	task, err := h.taskSvc.Cancel(r.Context(), taskID, workerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(task)
}
