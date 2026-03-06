package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DisputeBondRepo struct {
	pool *pgxpool.Pool
}

func NewDisputeBondRepo(pool *pgxpool.Pool) *DisputeBondRepo {
	return &DisputeBondRepo{pool: pool}
}

func (r *DisputeBondRepo) Create(ctx context.Context, bond *models.DisputeBond) error {
	query := `
		INSERT INTO dispute_bonds (dispute_id, worker_id, role, amount, tx_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		bond.DisputeID,
		bond.WorkerID,
		bond.Role,
		bond.Amount,
		bond.TxHash,
		bond.Status,
	).Scan(&bond.ID, &bond.CreatedAt)

	if err != nil {
		return fmt.Errorf("create dispute bond: %w", err)
	}
	return nil
}

func (r *DisputeBondRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DisputeBond, error) {
	var bond models.DisputeBond
	query := `
		SELECT id, dispute_id, worker_id, role, amount, tx_hash, status, return_tx_hash, created_at, settled_at
		FROM dispute_bonds
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&bond.ID,
		&bond.DisputeID,
		&bond.WorkerID,
		&bond.Role,
		&bond.Amount,
		&bond.TxHash,
		&bond.Status,
		&bond.ReturnTxHash,
		&bond.CreatedAt,
		&bond.SettledAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get dispute bond by id: %w", err)
	}
	return &bond, nil
}

func (r *DisputeBondRepo) ListByDispute(ctx context.Context, disputeID uuid.UUID) ([]models.DisputeBond, error) {
	bonds := make([]models.DisputeBond, 0)

	query := `
		SELECT id, dispute_id, worker_id, role, amount, tx_hash, status, return_tx_hash, created_at, settled_at
		FROM dispute_bonds
		WHERE dispute_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, disputeID)
	if err != nil {
		return nil, fmt.Errorf("list dispute bonds: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bond models.DisputeBond
		if err := rows.Scan(
			&bond.ID,
			&bond.DisputeID,
			&bond.WorkerID,
			&bond.Role,
			&bond.Amount,
			&bond.TxHash,
			&bond.Status,
			&bond.ReturnTxHash,
			&bond.CreatedAt,
			&bond.SettledAt,
		); err != nil {
			return nil, fmt.Errorf("scan dispute bond: %w", err)
		}
		bonds = append(bonds, bond)
	}

	return bonds, nil
}

func (r *DisputeBondRepo) MarkReturned(ctx context.Context, id uuid.UUID, returnTxHash string) error {
	query := `UPDATE dispute_bonds SET status = $1, return_tx_hash = $2, settled_at = now() WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, models.BondStatusReturned, returnTxHash, id)
	if err != nil {
		return fmt.Errorf("mark bond returned: %w", err)
	}
	return nil
}

func (r *DisputeBondRepo) MarkRetained(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE dispute_bonds SET status = $1, settled_at = now() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, models.BondStatusRetained, id)
	if err != nil {
		return fmt.Errorf("mark bond retained: %w", err)
	}
	return nil
}

func (r *DisputeBondRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dispute_bonds WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete dispute bond: %w", err)
	}
	return nil
}
