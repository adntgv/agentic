package models

import (
	"time"

	"github.com/google/uuid"
)

// User types
const (
	UserTypeHuman    = "human"
	UserTypeOperator = "operator"
)

// User statuses
const (
	UserStatusRegistered = "registered"
	UserStatusVerified   = "verified"
	UserStatusActive     = "active"
	UserStatusSuspended  = "suspended"
	UserStatusBanned     = "banned"
	UserStatusInactive   = "inactive"
)

// User represents a human account (poster, human worker, or operator)
type User struct {
	ID            uuid.UUID `db:"id" json:"id"`
	WalletAddress string    `db:"wallet_address" json:"wallet_address"`
	Email         *string   `db:"email" json:"email,omitempty"`
	DisplayName   *string   `db:"display_name" json:"display_name,omitempty"`
	UserType      string    `db:"user_type" json:"user_type"`
	Status        string    `db:"status" json:"status"`
	KYCVerified   bool      `db:"kyc_verified" json:"kyc_verified"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// Agent represents an AI agent linked to an operator
type Agent struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	OperatorID     uuid.UUID  `db:"operator_id" json:"operator_id"`
	WalletAddress  string     `db:"wallet_address" json:"wallet_address"`
	DisplayName    string     `db:"display_name" json:"display_name"`
	APIKeyHash     string     `db:"api_key_hash" json:"-"` // Never expose in JSON
	IsAI           bool       `db:"is_ai" json:"is_ai"`
	SkillManifest  *JSONBData `db:"skill_manifest" json:"skill_manifest,omitempty"`
	Status         string     `db:"status" json:"status"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

// Agent statuses
const (
	AgentStatusActive    = "active"
	AgentStatusSuspended = "suspended"
	AgentStatusBanned    = "banned"
)

// Worker types
const (
	WorkerTypeUser  = "user"
	WorkerTypeAgent = "agent"
)

// Worker represents the unified identity model - can be user or agent
type Worker struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	WorkerType string     `db:"worker_type" json:"worker_type"`
	UserID     *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	AgentID    *uuid.UUID `db:"agent_id" json:"agent_id,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// JSONBData is a helper type for JSONB columns
type JSONBData map[string]interface{}
