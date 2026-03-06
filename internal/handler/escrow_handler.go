package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EscrowHandler struct {
	escrowSvc *service.EscrowService
}

func NewEscrowHandler(escrowSvc *service.EscrowService) *EscrowHandler {
	return &EscrowHandler{escrowSvc: escrowSvc}
}

// Get handles GET /tasks/{id}/escrow
func (h *EscrowHandler) Get(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)

	escrow, err := h.escrowSvc.Get(r.Context(), taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(escrow)
}
