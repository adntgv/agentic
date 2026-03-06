package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyRepo struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepo(pool *pgxpool.Pool) *IdempotencyRepo {
	return &IdempotencyRepo{pool: pool}
}

func (r *IdempotencyRepo) Create(ctx context.Context, key *models.IdempotencyKey) error {
	query := `
		INSERT INTO idempotency_keys (key, worker_id, endpoint, response_status, response_body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, expires_at`

	err := r.pool.QueryRow(ctx, query,
		key.Key,
		key.WorkerID,
		key.Endpoint,
		key.ResponseStatus,
		key.ResponseBody,
	).Scan(&key.CreatedAt, &key.ExpiresAt)

	if err != nil {
		return fmt.Errorf("create idempotency key: %w", err)
	}
	return nil
}

func (r *IdempotencyRepo) Get(ctx context.Context, workerID uuid.UUID, endpoint, key string) (*models.IdempotencyKey, error) {
	var idempotencyKey models.IdempotencyKey
	query := `
		SELECT key, worker_id, endpoint, response_status, response_body, created_at, expires_at
		FROM idempotency_keys
		WHERE worker_id = $1 AND endpoint = $2 AND key = $3 AND expires_at > $4`

	err := r.pool.QueryRow(ctx, query, workerID, endpoint, key, time.Now()).Scan(
		&idempotencyKey.Key,
		&idempotencyKey.WorkerID,
		&idempotencyKey.Endpoint,
		&idempotencyKey.ResponseStatus,
		&idempotencyKey.ResponseBody,
		&idempotencyKey.CreatedAt,
		&idempotencyKey.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get idempotency key: %w", err)
	}
	return &idempotencyKey, nil
}

func (r *IdempotencyRepo) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM idempotency_keys WHERE expires_at <= $1`
	result, err := r.pool.Exec(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("delete expired idempotency keys: %w", err)
	}
	return result.RowsAffected(), nil
}
