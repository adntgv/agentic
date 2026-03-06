package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BidRepo struct {
	pool *pgxpool.Pool
}

func NewBidRepo(pool *pgxpool.Pool) *BidRepo {
	return &BidRepo{pool: pool}
}

func (r *BidRepo) Create(ctx context.Context, bid *models.Bid) error {
	query := `
		INSERT INTO bids (task_id, worker_id, amount, eta_hours, cover_letter, bid_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		bid.TaskID,
		bid.WorkerID,
		bid.Amount,
		bid.EtaHours,
		bid.CoverLetter,
		bid.BidHash,
		bid.Status,
	).Scan(&bid.ID, &bid.CreatedAt)

	if err != nil {
		return fmt.Errorf("create bid: %w", err)
	}
	return nil
}

func (r *BidRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Bid, error) {
	var bid models.Bid
	query := `
		SELECT id, task_id, worker_id, amount, eta_hours, cover_letter, bid_hash, status, created_at
		FROM bids
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&bid.ID,
		&bid.TaskID,
		&bid.WorkerID,
		&bid.Amount,
		&bid.EtaHours,
		&bid.CoverLetter,
		&bid.BidHash,
		&bid.Status,
		&bid.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get bid by id: %w", err)
	}
	return &bid, nil
}

func (r *BidRepo) ListByTask(ctx context.Context, taskID uuid.UUID, status *string) ([]models.Bid, error) {
	bids := make([]models.Bid, 0)

	query := `
		SELECT id, task_id, worker_id, amount, eta_hours, cover_letter, bid_hash, status, created_at
		FROM bids
		WHERE task_id = $1
		  AND ($2::text IS NULL OR status = $2)
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, taskID, status)
	if err != nil {
		return nil, fmt.Errorf("list bids by task: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bid models.Bid
		if err := rows.Scan(
			&bid.ID,
			&bid.TaskID,
			&bid.WorkerID,
			&bid.Amount,
			&bid.EtaHours,
			&bid.CoverLetter,
			&bid.BidHash,
			&bid.Status,
			&bid.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bid: %w", err)
		}
		bids = append(bids, bid)
	}

	return bids, nil
}

func (r *BidRepo) ListByWorker(ctx context.Context, workerID uuid.UUID, status *string, limit, offset int) ([]models.Bid, error) {
	bids := make([]models.Bid, 0)

	query := `
		SELECT id, task_id, worker_id, amount, eta_hours, cover_letter, bid_hash, status, created_at
		FROM bids
		WHERE worker_id = $1
		  AND ($2::text IS NULL OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, workerID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list bids by worker: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bid models.Bid
		if err := rows.Scan(
			&bid.ID,
			&bid.TaskID,
			&bid.WorkerID,
			&bid.Amount,
			&bid.EtaHours,
			&bid.CoverLetter,
			&bid.BidHash,
			&bid.Status,
			&bid.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bid: %w", err)
		}
		bids = append(bids, bid)
	}

	return bids, nil
}

func (r *BidRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE bids SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update bid status: %w", err)
	}
	return nil
}

func (r *BidRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM bids WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete bid: %w", err)
	}
	return nil
}
