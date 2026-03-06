package handler

import (
	"encoding/json"
	"net/http"
)

type WebhookHandler struct {
	// webhookSvc will be added when webhook service is implemented
}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// Create handles POST /webhooks
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	// TODO: Parse request body, create webhook
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// List handles GET /webhooks
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: List webhooks for authenticated worker
	json.NewEncoder(w).Encode([]interface{}{})
}
