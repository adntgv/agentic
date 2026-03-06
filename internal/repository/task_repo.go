package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type TaskRepo struct {
	pool *pgxpool.Pool
}

func NewTaskRepo(pool *pgxpool.Pool) *TaskRepo {
	return &TaskRepo{pool: pool}
}

type TaskFilters struct {
	Status        *string
	Category      *string
	WorkerFilter  *string
	PosterID      *uuid.UUID
	AssignedWorkerID *uuid.UUID
	MinBudget     *decimal.Decimal
	MaxBudget     *decimal.Decimal
	Limit         int
	Offset        int
}

func (r *TaskRepo) Create(ctx context.Context, task *models.Task) error {
	query := `
		INSERT INTO tasks (
			poster_worker_id, title, description, category, budget, deadline, bid_deadline,
			worker_filter, max_revisions, revision_count, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		task.PosterWorkerID,
		task.Title,
		task.Description,
		task.Category,
		task.Budget,
		task.Deadline,
		task.BidDeadline,
		task.WorkerFilter,
		task.MaxRevisions,
		task.RevisionCount,
		task.Status,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	query := `
		SELECT id, poster_worker_id, title, description, category, budget, deadline, bid_deadline,
			   worker_filter, max_revisions, revision_count, status, assigned_worker_id,
			   accepted_bid_id, task_id_hash, created_at, updated_at
		FROM tasks
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.PosterWorkerID,
		&task.Title,
		&task.Description,
		&task.Category,
		&task.Budget,
		&task.Deadline,
		&task.BidDeadline,
		&task.WorkerFilter,
		&task.MaxRevisions,
		&task.RevisionCount,
		&task.Status,
		&task.AssignedWorkerID,
		&task.AcceptedBidID,
		&task.TaskIDHash,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &task, nil
}

func (r *TaskRepo) Update(ctx context.Context, task *models.Task) error {
	query := `
		UPDATE tasks
		SET title = $1, description = $2, category = $3, budget = $4, deadline = $5,
			bid_deadline = $6, worker_filter = $7, max_revisions = $8, revision_count = $9,
			status = $10, updated_at = now()
		WHERE id = $11
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		task.Title,
		task.Description,
		task.Category,
		task.Budget,
		task.Deadline,
		task.BidDeadline,
		task.WorkerFilter,
		task.MaxRevisions,
		task.RevisionCount,
		task.Status,
		task.ID,
	).Scan(&task.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus string) error {
	query := `UPDATE tasks SET status = $1, updated_at = now() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, newStatus, id)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	return nil
}

func (r *TaskRepo) SetAssigned(ctx context.Context, taskID, workerID, bidID uuid.UUID, taskIDHash string) error {
	query := `
		UPDATE tasks
		SET assigned_worker_id = $1, accepted_bid_id = $2, task_id_hash = $3,
			status = $4, updated_at = now()
		WHERE id = $5`

	_, err := r.pool.Exec(ctx, query, workerID, bidID, taskIDHash, models.TaskStatusAssigned, taskID)
	if err != nil {
		return fmt.Errorf("set task assigned: %w", err)
	}
	return nil
}

func (r *TaskRepo) IncrementRevisionCount(ctx context.Context, taskID uuid.UUID) error {
	query := `UPDATE tasks SET revision_count = revision_count + 1, updated_at = now() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("increment revision count: %w", err)
	}
	return nil
}

func (r *TaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func (r *TaskRepo) List(ctx context.Context, filters TaskFilters) ([]models.Task, error) {
	tasks := make([]models.Task, 0)

	query := `
		SELECT id, poster_worker_id, title, description, category, budget, deadline, bid_deadline,
			   worker_filter, max_revisions, revision_count, status, assigned_worker_id,
			   accepted_bid_id, task_id_hash, created_at, updated_at
		FROM tasks
		WHERE ($1::text IS NULL OR status = $1)
		  AND ($2::text IS NULL OR category = $2)
		  AND ($3::text IS NULL OR worker_filter = $3)
		  AND ($4::uuid IS NULL OR poster_worker_id = $4)
		  AND ($5::uuid IS NULL OR assigned_worker_id = $5)
		  AND ($6::numeric IS NULL OR budget >= $6)
		  AND ($7::numeric IS NULL OR budget <= $7)
		ORDER BY created_at DESC
		LIMIT $8 OFFSET $9`

	rows, err := r.pool.Query(ctx, query,
		filters.Status,
		filters.Category,
		filters.WorkerFilter,
		filters.PosterID,
		filters.AssignedWorkerID,
		filters.MinBudget,
		filters.MaxBudget,
		filters.Limit,
		filters.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.ID,
			&task.PosterWorkerID,
			&task.Title,
			&task.Description,
			&task.Category,
			&task.Budget,
			&task.Deadline,
			&task.BidDeadline,
			&task.WorkerFilter,
			&task.MaxRevisions,
			&task.RevisionCount,
			&task.Status,
			&task.AssignedWorkerID,
			&task.AcceptedBidID,
			&task.TaskIDHash,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
