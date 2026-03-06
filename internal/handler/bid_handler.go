package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BidHandler handles bid-related HTTP endpoints
type BidHandler struct {
	bidSvc *service.BidService
}

func NewBidHandler(bidSvc *service.BidService) *BidHandler {
	return &BidHandler{bidSvc: bidSvc}
}

// Create handles POST /tasks/{id}/bids
func (h *BidHandler) Create(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)
	workerID := uuid.New() // from auth

	var req struct {
		Amount      decimal.Decimal `json:"amount"`
		EtaHours    int             `json:"eta_hours"`
		CoverLetter string          `json:"cover_letter"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	bid, err := h.bidSvc.PlaceBid(r.Context(), taskID, workerID, req.Amount, req.EtaHours, req.CoverLetter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bid)
}

// List handles GET /tasks/{id}/bids
func (h *BidHandler) List(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, _ := uuid.Parse(taskIDStr)

	bids, _ := h.bidSvc.List(r.Context(), taskID)
	json.NewEncoder(w).Encode(bids)
}

// Accept handles POST /tasks/{id}/bids/{bidId}/accept
func (h *BidHandler) Accept(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	bidIDStr := chi.URLParam(r, "bidId")
	taskID, _ := uuid.Parse(taskIDStr)
	bidID, _ := uuid.Parse(bidIDStr)
	workerID := uuid.New() // from auth

	depositParams, err := h.bidSvc.AcceptBid(r.Context(), taskID, bidID, workerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(depositParams)
}
