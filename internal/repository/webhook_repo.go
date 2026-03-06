package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookRepo(pool *pgxpool.Pool) *WebhookRepo {
	return &WebhookRepo{pool: pool}
}

func (r *WebhookRepo) Create(ctx context.Context, webhook *models.Webhook) error {
	query := `
		INSERT INTO webhooks (owner_worker_id, url, secret, events, status, consecutive_failures)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		webhook.OwnerWorkerID,
		webhook.URL,
		webhook.Secret,
		webhook.Events,
		webhook.Status,
		webhook.ConsecutiveFailures,
	).Scan(&webhook.ID, &webhook.CreatedAt)

	if err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}
	return nil
}

func (r *WebhookRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	var webhook models.Webhook
	query := `
		SELECT id, owner_worker_id, url, secret, events, status, consecutive_failures, created_at
		FROM webhooks
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID,
		&webhook.OwnerWorkerID,
		&webhook.URL,
		&webhook.Secret,
		&webhook.Events,
		&webhook.Status,
		&webhook.ConsecutiveFailures,
		&webhook.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get webhook by id: %w", err)
	}
	return &webhook, nil
}

func (r *WebhookRepo) ListByWorker(ctx context.Context, workerID uuid.UUID) ([]models.Webhook, error) {
	webhooks := make([]models.Webhook, 0)

	query := `
		SELECT id, owner_worker_id, url, secret, events, status, consecutive_failures, created_at
		FROM webhooks
		WHERE owner_worker_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, workerID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks by worker: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var webhook models.Webhook
		if err := rows.Scan(
			&webhook.ID,
			&webhook.OwnerWorkerID,
			&webhook.URL,
			&webhook.Secret,
			&webhook.Events,
			&webhook.Status,
			&webhook.ConsecutiveFailures,
			&webhook.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

func (r *WebhookRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE webhooks SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update webhook status: %w", err)
	}
	return nil
}

func (r *WebhookRepo) IncrementFailures(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE webhooks SET consecutive_failures = consecutive_failures + 1 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("increment webhook failures: %w", err)
	}
	return nil
}

func (r *WebhookRepo) ResetFailures(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE webhooks SET consecutive_failures = 0 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("reset webhook failures: %w", err)
	}
	return nil
}

func (r *WebhookRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	return nil
}
