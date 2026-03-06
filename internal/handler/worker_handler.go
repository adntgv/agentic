package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WorkerHandler struct {
	workerSvc *service.WorkerService
}

func NewWorkerHandler(workerSvc *service.WorkerService) *WorkerHandler {
	return &WorkerHandler{workerSvc: workerSvc}
}

// Get handles GET /workers/{id}
func (h *WorkerHandler) Get(w http.ResponseWriter, r *http.Request) {
	workerIDStr := chi.URLParam(r, "id")
	workerID, _ := uuid.Parse(workerIDStr)

	worker, err := h.workerSvc.GetProfile(r.Context(), workerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(worker)
}

// History handles GET /workers/{id}/history
func (h *WorkerHandler) History(w http.ResponseWriter, r *http.Request) {
	workerIDStr := chi.URLParam(r, "id")
	workerID, _ := uuid.Parse(workerIDStr)

	history, _ := h.workerSvc.GetHistory(r.Context(), workerID)
	json.NewEncoder(w).Encode(history)
}

// GetOperator handles GET /operators/{id}
func (h *WorkerHandler) GetOperator(w http.ResponseWriter, r *http.Request) {
	operatorIDStr := chi.URLParam(r, "id")
	operatorID, _ := uuid.Parse(operatorIDStr)

	operator, err := h.workerSvc.GetOperator(r.Context(), operatorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(operator)
}
