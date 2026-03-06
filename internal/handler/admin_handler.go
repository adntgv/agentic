package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	taskSvc *service.TaskService
}

func NewAdminHandler(taskSvc *service.TaskService) *AdminHandler {
	return &AdminHandler{taskSvc: taskSvc}
}

// Unassign handles POST /tasks/{id}/unassign (admin only)
func (h *AdminHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	adminWorkerID := uuid.New() // from auth

	var req struct {
		NewWorkerID *uuid.UUID `json:"new_worker_id"`
		Action      string     `json:"action"` // reassign | refund
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Action == "reassign" && req.NewWorkerID != nil {
		task, err := h.taskSvc.AdminReassign(r.Context(), taskID, *req.NewWorkerID, adminWorkerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(task)
		return
	}

	if req.Action == "refund" {
		task, err := h.taskSvc.AdminRefund(r.Context(), taskID, adminWorkerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(task)
		return
	}

	http.Error(w, "invalid action", http.StatusBadRequest)
}
