package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReputationRepo struct {
	pool *pgxpool.Pool
}

func NewReputationRepo(pool *pgxpool.Pool) *ReputationRepo {
	return &ReputationRepo{pool: pool}
}

func (r *ReputationRepo) Create(ctx context.Context, reputation *models.Reputation) error {
	query := `
		INSERT INTO reputation (rated_worker_id, rater_worker_id, task_id, rating, role, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		reputation.RatedWorkerID,
		reputation.RaterWorkerID,
		reputation.TaskID,
		reputation.Rating,
		reputation.Role,
		reputation.Comment,
	).Scan(&reputation.ID, &reputation.CreatedAt)

	if err != nil {
		return fmt.Errorf("create reputation: %w", err)
	}
	return nil
}

func (r *ReputationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Reputation, error) {
	var reputation models.Reputation
	query := `
		SELECT id, rated_worker_id, rater_worker_id, task_id, rating, role, comment, created_at
		FROM reputation
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&reputation.ID,
		&reputation.RatedWorkerID,
		&reputation.RaterWorkerID,
		&reputation.TaskID,
		&reputation.Rating,
		&reputation.Role,
		&reputation.Comment,
		&reputation.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get reputation by id: %w", err)
	}
	return &reputation, nil
}

func (r *ReputationRepo) ListByRatedWorker(ctx context.Context, workerID uuid.UUID, limit, offset int) ([]models.Reputation, error) {
	reputations := make([]models.Reputation, 0)

	query := `
		SELECT id, rated_worker_id, rater_worker_id, task_id, rating, role, comment, created_at
		FROM reputation
		WHERE rated_worker_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, workerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list reputations by rated worker: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reputation models.Reputation
		if err := rows.Scan(
			&reputation.ID,
			&reputation.RatedWorkerID,
			&reputation.RaterWorkerID,
			&reputation.TaskID,
			&reputation.Rating,
			&reputation.Role,
			&reputation.Comment,
			&reputation.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reputation: %w", err)
		}
		reputations = append(reputations, reputation)
	}

	return reputations, nil
}

func (r *ReputationRepo) GetSummary(ctx context.Context, workerID uuid.UUID) (*models.ReputationSummary, error) {
	var summary models.ReputationSummary
	query := `
		SELECT rated_worker_id, total_ratings, avg_rating, positive_count, negative_count
		FROM reputation_summary
		WHERE rated_worker_id = $1`

	err := r.pool.QueryRow(ctx, query, workerID).Scan(
		&summary.RatedWorkerID,
		&summary.TotalRatings,
		&summary.AvgRating,
		&summary.PositiveCount,
		&summary.NegativeCount,
	)

	if err != nil {
		return nil, fmt.Errorf("get reputation summary: %w", err)
	}
	return &summary, nil
}

func (r *ReputationRepo) RefreshSummary(ctx context.Context) error {
	query := `REFRESH MATERIALIZED VIEW CONCURRENTLY reputation_summary`
	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("refresh reputation summary: %w", err)
	}
	return nil
}

func (r *ReputationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reputation WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete reputation: %w", err)
	}
	return nil
}
