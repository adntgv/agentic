package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StakeRepo struct {
	pool *pgxpool.Pool
}

func NewStakeRepo(pool *pgxpool.Pool) *StakeRepo {
	return &StakeRepo{pool: pool}
}

func (r *StakeRepo) Create(ctx context.Context, stake *models.Stake) error {
	query := `
		INSERT INTO stakes (operator_id, agent_id, amount, tx_hash, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		stake.OperatorID,
		stake.AgentID,
		stake.Amount,
		stake.TxHash,
		stake.Status,
	).Scan(&stake.ID, &stake.CreatedAt, &stake.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create stake: %w", err)
	}
	return nil
}

func (r *StakeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Stake, error) {
	var stake models.Stake
	query := `
		SELECT id, operator_id, agent_id, amount, tx_hash, status, created_at, updated_at
		FROM stakes
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&stake.ID,
		&stake.OperatorID,
		&stake.AgentID,
		&stake.Amount,
		&stake.TxHash,
		&stake.Status,
		&stake.CreatedAt,
		&stake.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get stake by id: %w", err)
	}
	return &stake, nil
}

func (r *StakeRepo) ListByOperator(ctx context.Context, operatorID uuid.UUID) ([]models.Stake, error) {
	stakes := make([]models.Stake, 0)

	query := `
		SELECT id, operator_id, agent_id, amount, tx_hash, status, created_at, updated_at
		FROM stakes
		WHERE operator_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, operatorID)
	if err != nil {
		return nil, fmt.Errorf("list stakes by operator: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stake models.Stake
		if err := rows.Scan(
			&stake.ID,
			&stake.OperatorID,
			&stake.AgentID,
			&stake.Amount,
			&stake.TxHash,
			&stake.Status,
			&stake.CreatedAt,
			&stake.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stake: %w", err)
		}
		stakes = append(stakes, stake)
	}

	return stakes, nil
}

func (r *StakeRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE stakes SET status = $1, updated_at = now() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update stake status: %w", err)
	}
	return nil
}

func (r *StakeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM stakes WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete stake: %w", err)
	}
	return nil
}
