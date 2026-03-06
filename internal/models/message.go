package models

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a message in a task thread
type Message struct {
	ID             uuid.UUID `db:"id" json:"id"`
	TaskID         uuid.UUID `db:"task_id" json:"task_id"`
	SenderWorkerID uuid.UUID `db:"sender_worker_id" json:"sender_worker_id"`
	Body           string    `db:"body" json:"body"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}
