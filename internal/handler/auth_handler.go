package handler

import (
	"encoding/json"
	"net/http"

	"github.com/aid/agentic/internal/service"
)

type AuthHandler struct {
	workerSvc *service.WorkerService
}

func NewAuthHandler(workerSvc *service.WorkerService) *AuthHandler {
	return &AuthHandler{workerSvc: workerSvc}
}

// WalletAuth handles POST /auth/wallet (SIWE)
func (h *AuthHandler) WalletAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WalletAddr string `json:"wallet_address"`
		Signature  string `json:"signature"`
		Message    string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	token, err := h.workerSvc.AuthWallet(r.Context(), req.WalletAddr, req.Signature, req.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// APIKeyAuth handles POST /auth/apikey
func (h *AuthHandler) APIKeyAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"api_key"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	worker, err := h.workerSvc.AuthAPIKey(r.Context(), req.APIKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(worker)
}
