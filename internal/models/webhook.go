package models

import (
	"time"

	"github.com/google/uuid"
)

// Webhook statuses
const (
	WebhookStatusActive   = "active"
	WebhookStatusDisabled = "disabled"
)

// Webhook represents a registered webhook endpoint
type Webhook struct {
	ID                  uuid.UUID `db:"id" json:"id"`
	OwnerWorkerID       uuid.UUID `db:"owner_worker_id" json:"owner_worker_id"`
	URL                 string    `db:"url" json:"url"`
	Secret              string    `db:"secret" json:"-"` // Never expose in JSON
	Events              []string  `db:"events" json:"events"`
	Status              string    `db:"status" json:"status"`
	ConsecutiveFailures int       `db:"consecutive_failures" json:"consecutive_failures"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}
