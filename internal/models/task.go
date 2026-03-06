package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Task statuses
const (
	TaskStatusDraft          = "draft"
	TaskStatusPublished      = "published"
	TaskStatusBidding        = "bidding"
	TaskStatusPendingDeposit = "pending_deposit"
	TaskStatusAssigned       = "assigned"
	TaskStatusInProgress     = "in_progress"
	TaskStatusReview         = "review"
	TaskStatusCompleted      = "completed"
	TaskStatusRefunded       = "refunded"
	TaskStatusSplit          = "split"
	TaskStatusDisputed       = "disputed"
	TaskStatusAbandoned      = "abandoned"
	TaskStatusOverdue        = "overdue"
	TaskStatusExpired        = "expired"
	TaskStatusCancelled      = "cancelled"
	TaskStatusDeleted        = "deleted"
)

// Worker filters
const (
	WorkerFilterHumanOnly = "human_only"
	WorkerFilterAIOnly    = "ai_only"
	WorkerFilterBoth      = "both"
)

// Task represents a work task
type Task struct {
	ID               uuid.UUID        `db:"id" json:"id"`
	PosterWorkerID   uuid.UUID        `db:"poster_worker_id" json:"poster_worker_id"`
	Title            string           `db:"title" json:"title"`
	Description      string           `db:"description" json:"description"`
	Category         *string          `db:"category" json:"category,omitempty"`
	Budget           decimal.Decimal  `db:"budget" json:"budget"`
	Deadline         *time.Time       `db:"deadline" json:"deadline,omitempty"`
	BidDeadline      *time.Time       `db:"bid_deadline" json:"bid_deadline,omitempty"`
	WorkerFilter     string           `db:"worker_filter" json:"worker_filter"`
	MaxRevisions     int16            `db:"max_revisions" json:"max_revisions"`
	RevisionCount    int16            `db:"revision_count" json:"revision_count"`
	Status           string           `db:"status" json:"status"`
	AssignedWorkerID *uuid.UUID       `db:"assigned_worker_id" json:"assigned_worker_id,omitempty"`
	AcceptedBidID    *uuid.UUID       `db:"accepted_bid_id" json:"accepted_bid_id,omitempty"`
	TaskIDHash       *string          `db:"task_id_hash" json:"task_id_hash,omitempty"`
	CreatedAt        time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time        `db:"updated_at" json:"updated_at"`
}

// Bid statuses
const (
	BidStatusPending   = "pending"
	BidStatusAccepted  = "accepted"
	BidStatusRejected  = "rejected"
	BidStatusWithdrawn = "withdrawn"
)

// Bid represents a worker's bid on a task
type Bid struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	TaskID      uuid.UUID       `db:"task_id" json:"task_id"`
	WorkerID    uuid.UUID       `db:"worker_id" json:"worker_id"`
	Amount      decimal.Decimal `db:"amount" json:"amount"`
	EtaHours    *int            `db:"eta_hours" json:"eta_hours,omitempty"`
	CoverLetter *string         `db:"cover_letter" json:"cover_letter,omitempty"`
	BidHash     *string         `db:"bid_hash" json:"bid_hash,omitempty"`
	Status      string          `db:"status" json:"status"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}
