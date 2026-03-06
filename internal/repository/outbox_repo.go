package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aid/agentic/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepo struct {
	pool *pgxpool.Pool
}

func NewOutboxRepo(pool *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{pool: pool}
}

func (r *OutboxRepo) Create(ctx context.Context, event *models.OutboxEvent) error {
	query := `
		INSERT INTO outbox (event_type, payload, idempotency_key, status, retry_count, next_retry_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, sequence_number, created_at`

	err := r.pool.QueryRow(ctx, query,
		event.EventType,
		event.Payload,
		event.IdempotencyKey,
		event.Status,
		event.RetryCount,
		event.NextRetryAt,
	).Scan(&event.ID, &event.SequenceNumber, &event.CreatedAt)

	if err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}
	return nil
}

func (r *OutboxRepo) GetByID(ctx context.Context, id int64) (*models.OutboxEvent, error) {
	var event models.OutboxEvent
	query := `
		SELECT id, event_type, payload, idempotency_key, sequence_number, status, retry_count, next_retry_at, created_at, dispatched_at
		FROM outbox
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.EventType,
		&event.Payload,
		&event.IdempotencyKey,
		&event.SequenceNumber,
		&event.Status,
		&event.RetryCount,
		&event.NextRetryAt,
		&event.CreatedAt,
		&event.DispatchedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get outbox event by id: %w", err)
	}
	return &event, nil
}

func (r *OutboxRepo) GetPending(ctx context.Context, limit int) ([]models.OutboxEvent, error) {
	events := make([]models.OutboxEvent, 0)

	query := `
		SELECT id, event_type, payload, idempotency_key, sequence_number, status, retry_count, next_retry_at, created_at, dispatched_at
		FROM outbox
		WHERE status = $1
		  AND (next_retry_at IS NULL OR next_retry_at <= $2)
		ORDER BY id ASC
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, models.OutboxStatusPending, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("get pending outbox events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event models.OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Payload,
			&event.IdempotencyKey,
			&event.SequenceNumber,
			&event.Status,
			&event.RetryCount,
			&event.NextRetryAt,
			&event.CreatedAt,
			&event.DispatchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *OutboxRepo) MarkDispatched(ctx context.Context, id int64) error {
	query := `UPDATE outbox SET status = $1, dispatched_at = now() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, models.OutboxStatusDispatched, id)
	if err != nil {
		return fmt.Errorf("mark outbox event dispatched: %w", err)
	}
	return nil
}

func (r *OutboxRepo) MarkFailed(ctx context.Context, id int64, nextRetryAt time.Time) error {
	query := `UPDATE outbox SET status = $1, retry_count = retry_count + 1, next_retry_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, models.OutboxStatusFailed, nextRetryAt, id)
	if err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}
	return nil
}

func (r *OutboxRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM outbox WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete outbox event: %w", err)
	}
	return nil
}
