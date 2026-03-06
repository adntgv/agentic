package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DisputeHandler struct {
	disputeSvc *service.DisputeService
}

func NewDisputeHandler(disputeSvc *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{disputeSvc: disputeSvc}
}

// Raise handles POST /tasks/{id}/disputes
func (h *DisputeHandler) Raise(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	var req struct {
		Reason     string `json:"reason"`
		BondTxHash string `json:"bond_tx_hash"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	dispute, err := h.disputeSvc.Raise(r.Context(), taskID, workerID, req.Reason, req.BondTxHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dispute)
}

// Respond handles POST /disputes/{id}/respond
func (h *DisputeHandler) Respond(w http.ResponseWriter, r *http.Request) {
	disputeIDStr := chi.URLParam(r, "id")
	disputeID, _ := uuid.Parse(disputeIDStr)
	workerID := uuid.New() // from auth

	var req struct {
		BondTxHash string `json:"bond_tx_hash"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	dispute, err := h.disputeSvc.Respond(r.Context(), disputeID, workerID, req.BondTxHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(dispute)
}

// Evidence handles POST /disputes/{id}/evidence
func (h *DisputeHandler) Evidence(w http.ResponseWriter, r *http.Request) {
	disputeIDStr := chi.URLParam(r, "id")
	disputeID, _ := uuid.Parse(disputeIDStr)
	workerID := uuid.New() // from auth

	var req struct {
		ArtifactIDs []uuid.UUID `json:"artifact_ids"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	err := h.disputeSvc.SubmitEvidence(r.Context(), disputeID, workerID, req.ArtifactIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Ruling handles POST /disputes/{id}/ruling
func (h *DisputeHandler) Ruling(w http.ResponseWriter, r *http.Request) {
	disputeIDStr := chi.URLParam(r, "id")
	disputeID, _ := uuid.Parse(disputeIDStr)
	adminWorkerID := uuid.New() // from auth

	var req struct {
		Outcome   string `json:"outcome"`
		AgentBps  *int16 `json:"agent_bps"`
		Rationale string `json:"rationale"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	dispute, err := h.disputeSvc.Ruling(r.Context(), disputeID, adminWorkerID, req.Outcome, req.AgentBps, req.Rationale)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(dispute)
}
