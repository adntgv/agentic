package api

import (
	"encoding/json"
	"net/http"
)

// UserHandler handles user-related requests
type UserHandler struct {
	// TODO: Add dependencies
}

// GetUser returns a user by ID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.WriteHeader(http.StatusCreated)
}
