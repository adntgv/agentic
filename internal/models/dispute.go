package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Dispute reasons
const (
	DisputeReasonIncomplete       = "incomplete"
	DisputeReasonWrongRequirements = "wrong_requirements"
	DisputeReasonQuality          = "quality"
	DisputeReasonFraud            = "fraud"
)

// Dispute statuses
const (
	DisputeStatusRaised      = "raised"
	DisputeStatusEvidence    = "evidence"
	DisputeStatusArbitration = "arbitration"
	DisputeStatusResolved    = "resolved"
)

// Dispute outcomes
const (
	DisputeOutcomeAgentWins  = "agent_wins"
	DisputeOutcomePosterWins = "poster_wins"
	DisputeOutcomeSplit      = "split"
)

// Dispute represents a dispute on a task
type Dispute struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	TaskID                 uuid.UUID  `db:"task_id" json:"task_id"`
	PosterWorkerID         uuid.UUID  `db:"poster_worker_id" json:"poster_worker_id"`
	AssignedWorkerID       uuid.UUID  `db:"assigned_worker_id" json:"assigned_worker_id"`
	RaisedByWorkerID       uuid.UUID  `db:"raised_by_worker_id" json:"raised_by_worker_id"`
	Reason                 string     `db:"reason" json:"reason"`
	Status                 string     `db:"status" json:"status"`
	Outcome                *string    `db:"outcome" json:"outcome,omitempty"`
	AgentBps               *int16     `db:"agent_bps" json:"agent_bps,omitempty"`
	Rationale              *string    `db:"rationale" json:"rationale,omitempty"`
	EvidenceDeadline       *time.Time `db:"evidence_deadline" json:"evidence_deadline,omitempty"`
	BondResponseDeadline   *time.Time `db:"bond_response_deadline" json:"bond_response_deadline,omitempty"`
	SLAResponseDeadline    *time.Time `db:"sla_response_deadline" json:"sla_response_deadline,omitempty"`
	SLAResolutionDeadline  *time.Time `db:"sla_resolution_deadline" json:"sla_resolution_deadline,omitempty"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	ResolvedAt             *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
}

// Dispute bond roles
const (
	BondRoleRaiser    = "raiser"
	BondRoleResponder = "responder"
)

// Dispute bond statuses
const (
	BondStatusPosted   = "posted"
	BondStatusReturned = "returned"
	BondStatusRetained = "retained"
)

// DisputeBond represents a bond posted for a dispute
type DisputeBond struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	DisputeID    uuid.UUID       `db:"dispute_id" json:"dispute_id"`
	WorkerID     uuid.UUID       `db:"worker_id" json:"worker_id"`
	Role         string          `db:"role" json:"role"`
	Amount       decimal.Decimal `db:"amount" json:"amount"`
	TxHash       string          `db:"tx_hash" json:"tx_hash"`
	Status       string          `db:"status" json:"status"`
	ReturnTxHash *string         `db:"return_tx_hash" json:"return_tx_hash,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	SettledAt    *time.Time      `db:"settled_at" json:"settled_at,omitempty"`
}
