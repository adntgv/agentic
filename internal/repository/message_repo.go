package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

func (r *MessageRepo) Create(ctx context.Context, message *models.Message) error {
	query := `
		INSERT INTO messages (task_id, sender_worker_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		message.TaskID,
		message.SenderWorkerID,
		message.Body,
	).Scan(&message.ID, &message.CreatedAt)

	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

func (r *MessageRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error) {
	var message models.Message
	query := `
		SELECT id, task_id, sender_worker_id, body, created_at
		FROM messages
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&message.ID,
		&message.TaskID,
		&message.SenderWorkerID,
		&message.Body,
		&message.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	return &message, nil
}

func (r *MessageRepo) ListByTask(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]models.Message, error) {
	messages := make([]models.Message, 0)

	query := `
		SELECT id, task_id, sender_worker_id, body, created_at
		FROM messages
		WHERE task_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list messages by task: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var message models.Message
		if err := rows.Scan(
			&message.ID,
			&message.TaskID,
			&message.SenderWorkerID,
			&message.Body,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func (r *MessageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM messages WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}
