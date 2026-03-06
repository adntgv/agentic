package service

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
)

// ReputationService handles reputation tracking and materialized view refresh
type ReputationService struct {
	repRepo *repository.ReputationRepo
}

// NewReputationService creates a new ReputationService instance
func NewReputationService(repRepo *repository.ReputationRepo) *ReputationService {
	return &ReputationService{
		repRepo: repRepo,
	}
}

// Rate creates a new rating for a worker
func (s *ReputationService) Rate(ctx context.Context, ratedWorkerID, raterWorkerID, taskID uuid.UUID, rating int16, role, comment string) (*models.Reputation, error) {
	// Validate rating range
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	// TODO: Validate that rater is a party to the task
	// TODO: Validate that task is completed
	// TODO: Prevent duplicate ratings for same task

	rep := &models.Reputation{
		ID:             uuid.New(),
		RatedWorkerID:  ratedWorkerID,
		RaterWorkerID:  raterWorkerID,
		TaskID:         &taskID,
		Rating:         rating,
		Role:           role,
		Comment:        &comment,
	}

	if err := s.repRepo.Create(ctx, rep); err != nil {
		return nil, fmt.Errorf("create reputation: %w", err)
	}

	// Refresh materialized view
	if err := s.RefreshSummary(ctx); err != nil {
		// Log error but don't fail the rating
		fmt.Printf("failed to refresh reputation summary: %v\n", err)
	}

	return rep, nil
}

// GetSummary retrieves reputation summary for a worker
func (s *ReputationService) GetSummary(ctx context.Context, workerID uuid.UUID) (*models.ReputationSummary, error) {
	return s.repRepo.GetSummary(ctx, workerID)
}

// RefreshSummary refreshes the reputation_summary materialized view
func (s *ReputationService) RefreshSummary(ctx context.Context) error {
	return s.repRepo.RefreshSummary(ctx)
}
