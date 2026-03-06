package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Escrow statuses
const (
	EscrowStatusNone     = "none"
	EscrowStatusLocked   = "locked"
	EscrowStatusReleased = "released"
	EscrowStatusRefunded = "refunded"
	EscrowStatusSplit    = "split"
)

// Escrow mirrors the on-chain escrow state
type Escrow struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	TaskID         uuid.UUID       `db:"task_id" json:"task_id"`
	PosterAddress  string          `db:"poster_address" json:"poster_address"`
	PayeeAddress   string          `db:"payee_address" json:"payee_address"`
	Amount         decimal.Decimal `db:"amount" json:"amount"`
	BidHash        string          `db:"bid_hash" json:"bid_hash"`
	Status         string          `db:"status" json:"status"`
	DepositTxHash  *string         `db:"deposit_tx_hash" json:"deposit_tx_hash,omitempty"`
	ReleaseTxHash  *string         `db:"release_tx_hash" json:"release_tx_hash,omitempty"`
	RefundTxHash   *string         `db:"refund_tx_hash" json:"refund_tx_hash,omitempty"`
	SplitTxHash    *string         `db:"split_tx_hash" json:"split_tx_hash,omitempty"`
	DepositedAt    *time.Time      `db:"deposited_at" json:"deposited_at,omitempty"`
	ResolvedAt     *time.Time      `db:"resolved_at" json:"resolved_at,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

// Transaction types
const (
	TxTypeDeposit       = "deposit"
	TxTypeRelease       = "release"
	TxTypeRefund        = "refund"
	TxTypeSplit         = "split"
	TxTypeReassign      = "reassign"
	TxTypeMarkSubmitted = "mark_submitted"
)

// Transaction statuses
const (
	TxStatusPending   = "pending"
	TxStatusConfirmed = "confirmed"
	TxStatusFailed    = "failed"
	TxStatusReorged   = "reorged"
)

// Transaction represents an on-chain transaction
type Transaction struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	EscrowID    uuid.UUID  `db:"escrow_id" json:"escrow_id"`
	TxHash      string     `db:"tx_hash" json:"tx_hash"`
	TxType      string     `db:"tx_type" json:"tx_type"`
	BlockNumber *int64     `db:"block_number" json:"block_number,omitempty"`
	BlockHash   *string    `db:"block_hash" json:"block_hash,omitempty"`
	Status      string     `db:"status" json:"status"`
	GasUsed     *int64     `db:"gas_used" json:"gas_used,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	ConfirmedAt *time.Time `db:"confirmed_at" json:"confirmed_at,omitempty"`
}
