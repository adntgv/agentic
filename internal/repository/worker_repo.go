package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkerRepo struct {
	pool *pgxpool.Pool
}

func NewWorkerRepo(pool *pgxpool.Pool) *WorkerRepo {
	return &WorkerRepo{pool: pool}
}

func (r *WorkerRepo) Create(ctx context.Context, worker *models.Worker) error {
	query := `
		INSERT INTO workers (worker_type, user_id, agent_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		worker.WorkerType,
		worker.UserID,
		worker.AgentID,
	).Scan(&worker.ID, &worker.CreatedAt)

	if err != nil {
		return fmt.Errorf("create worker: %w", err)
	}
	return nil
}

func (r *WorkerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Worker, error) {
	var worker models.Worker
	query := `
		SELECT id, worker_type, user_id, agent_id, created_at
		FROM workers
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&worker.ID,
		&worker.WorkerType,
		&worker.UserID,
		&worker.AgentID,
		&worker.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get worker by id: %w", err)
	}
	return &worker, nil
}

func (r *WorkerRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.Worker, error) {
	var worker models.Worker
	query := `
		SELECT id, worker_type, user_id, agent_id, created_at
		FROM workers
		WHERE user_id = $1`

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&worker.ID,
		&worker.WorkerType,
		&worker.UserID,
		&worker.AgentID,
		&worker.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get worker by user id: %w", err)
	}
	return &worker, nil
}

func (r *WorkerRepo) GetByAgentID(ctx context.Context, agentID uuid.UUID) (*models.Worker, error) {
	var worker models.Worker
	query := `
		SELECT id, worker_type, user_id, agent_id, created_at
		FROM workers
		WHERE agent_id = $1`

	err := r.pool.QueryRow(ctx, query, agentID).Scan(
		&worker.ID,
		&worker.WorkerType,
		&worker.UserID,
		&worker.AgentID,
		&worker.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get worker by agent id: %w", err)
	}
	return &worker, nil
}

func (r *WorkerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workers WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete worker: %w", err)
	}
	return nil
}

func (r *WorkerRepo) List(ctx context.Context, workerType *string, limit, offset int) ([]models.Worker, error) {
	workers := make([]models.Worker, 0)

	query := `
		SELECT id, worker_type, user_id, agent_id, created_at
		FROM workers
		WHERE ($1::text IS NULL OR worker_type = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, workerType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var worker models.Worker
		if err := rows.Scan(
			&worker.ID,
			&worker.WorkerType,
			&worker.UserID,
			&worker.AgentID,
			&worker.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan worker: %w", err)
		}
		workers = append(workers, worker)
	}

	return workers, nil
}
