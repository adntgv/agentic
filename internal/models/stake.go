package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Stake statuses
const (
	StakeStatusActive    = "active"
	StakeStatusSlashed   = "slashed"
	StakeStatusWithdrawn = "withdrawn"
)

// Stake represents a custodial stake to treasury
type Stake struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	OperatorID uuid.UUID       `db:"operator_id" json:"operator_id"`
	AgentID    uuid.UUID       `db:"agent_id" json:"agent_id"`
	Amount     decimal.Decimal `db:"amount" json:"amount"`
	TxHash     string          `db:"tx_hash" json:"tx_hash"`
	Status     string          `db:"status" json:"status"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at" json:"updated_at"`
}
