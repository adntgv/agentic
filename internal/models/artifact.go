package models

import (
	"time"

	"github.com/google/uuid"
)

// Artifact contexts
const (
	ArtifactContextSubmission       = "submission"
	ArtifactContextEvidence         = "evidence"
	ArtifactContextRevisionRequest  = "revision_request"
)

// Artifact kinds
const (
	ArtifactKindFile = "file"
	ArtifactKindURL  = "url"
	ArtifactKindText = "text"
)

// Artifact statuses
const (
	ArtifactStatusPending   = "pending"
	ArtifactStatusFinalized = "finalized"
)

// AV scan statuses
const (
	AVScanStatusPending  = "pending"
	AVScanStatusClean    = "clean"
	AVScanStatusInfected = "infected"
	AVScanStatusSkipped  = "skipped"
)

// Artifact represents a file, URL, or text artifact
type Artifact struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	TaskID       uuid.UUID  `db:"task_id" json:"task_id"`
	WorkerID     uuid.UUID  `db:"worker_id" json:"worker_id"`
	Context      string     `db:"context" json:"context"`
	Kind         string     `db:"kind" json:"kind"`
	SHA256       *string    `db:"sha256" json:"sha256,omitempty"`
	URL          *string    `db:"url" json:"url,omitempty"`
	TextBody     *string    `db:"text_body" json:"text_body,omitempty"`
	MIME         *string    `db:"mime" json:"mime,omitempty"`
	Bytes        *int64     `db:"bytes" json:"bytes,omitempty"`
	Status       string     `db:"status" json:"status"`
	AVScanStatus string     `db:"av_scan_status" json:"av_scan_status"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	FinalizedAt  *time.Time `db:"finalized_at" json:"finalized_at,omitempty"`
}
