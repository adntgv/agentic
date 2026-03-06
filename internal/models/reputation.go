package models

import (
	"time"

	"github.com/google/uuid"
)

// Reputation roles
const (
	ReputationRolePoster = "poster"
	ReputationRoleWorker = "worker"
)

// Reputation represents a reputation rating
type Reputation struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	RatedWorkerID  uuid.UUID  `db:"rated_worker_id" json:"rated_worker_id"`
	RaterWorkerID  uuid.UUID  `db:"rater_worker_id" json:"rater_worker_id"`
	TaskID         *uuid.UUID `db:"task_id" json:"task_id,omitempty"`
	Rating         int16      `db:"rating" json:"rating"`
	Role           string     `db:"role" json:"role"`
	Comment        *string    `db:"comment" json:"comment,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// ReputationSummary represents aggregated reputation data
type ReputationSummary struct {
	RatedWorkerID  uuid.UUID `db:"rated_worker_id" json:"rated_worker_id"`
	TotalRatings   int64     `db:"total_ratings" json:"total_ratings"`
	AvgRating      float64   `db:"avg_rating" json:"avg_rating"`
	PositiveCount  int64     `db:"positive_count" json:"positive_count"`
	NegativeCount  int64     `db:"negative_count" json:"negative_count"`
}
