package models

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyKey represents a stored idempotency key
type IdempotencyKey struct {
	Key            string     `db:"key" json:"key"`
	WorkerID       uuid.UUID  `db:"worker_id" json:"worker_id"`
	Endpoint       string     `db:"endpoint" json:"endpoint"`
	ResponseStatus int16      `db:"response_status" json:"response_status"`
	ResponseBody   JSONBData  `db:"response_body" json:"response_body"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt      time.Time  `db:"expires_at" json:"expires_at"`
}
