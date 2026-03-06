package models

import (
	"time"
)

// Outbox event statuses
const (
	OutboxStatusPending    = "pending"
	OutboxStatusDispatched = "dispatched"
	OutboxStatusFailed     = "failed"
)

// OutboxEvent represents an event to be dispatched
type OutboxEvent struct {
	ID             int64      `db:"id" json:"id"`
	EventType      string     `db:"event_type" json:"event_type"`
	Payload        JSONBData  `db:"payload" json:"payload"`
	IdempotencyKey string     `db:"idempotency_key" json:"idempotency_key"`
	SequenceNumber int64      `db:"sequence_number" json:"sequence_number"`
	Status         string     `db:"status" json:"status"`
	RetryCount     int16      `db:"retry_count" json:"retry_count"`
	NextRetryAt    *time.Time `db:"next_retry_at" json:"next_retry_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	DispatchedAt   *time.Time `db:"dispatched_at" json:"dispatched_at,omitempty"`
}
