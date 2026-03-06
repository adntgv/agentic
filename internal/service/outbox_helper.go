package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
)

// writeOutboxEvent creates and persists an outbox event with the given type and payload.
func writeOutboxEvent(ctx context.Context, repo *repository.OutboxRepo, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	var jsonb models.JSONBData
	if err := json.Unmarshal(data, &jsonb); err != nil {
		return fmt.Errorf("unmarshal outbox payload: %w", err)
	}

	event := &models.OutboxEvent{
		EventType:      eventType,
		Payload:        jsonb,
		IdempotencyKey: fmt.Sprintf("%s-%s", eventType, uuid.New().String()),
		Status:         "pending",
		RetryCount:     0,
		CreatedAt:      time.Now(),
	}

	return repo.Create(ctx, event)
}
