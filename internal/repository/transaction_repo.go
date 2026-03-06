package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

func (r *TransactionRepo) Create(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (escrow_id, tx_hash, tx_type, block_number, block_hash, status, gas_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		tx.EscrowID,
		tx.TxHash,
		tx.TxType,
		tx.BlockNumber,
		tx.BlockHash,
		tx.Status,
		tx.GasUsed,
	).Scan(&tx.ID, &tx.CreatedAt)

	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepo) GetByTxHash(ctx context.Context, txHash string) (*models.Transaction, error) {
	var tx models.Transaction
	query := `
		SELECT id, escrow_id, tx_hash, tx_type, block_number, block_hash, status, gas_used, created_at, confirmed_at
		FROM transactions
		WHERE tx_hash = $1`

	err := r.pool.QueryRow(ctx, query, txHash).Scan(
		&tx.ID,
		&tx.EscrowID,
		&tx.TxHash,
		&tx.TxType,
		&tx.BlockNumber,
		&tx.BlockHash,
		&tx.Status,
		&tx.GasUsed,
		&tx.CreatedAt,
		&tx.ConfirmedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get transaction by tx hash: %w", err)
	}
	return &tx, nil
}

func (r *TransactionRepo) ListByEscrow(ctx context.Context, escrowID uuid.UUID) ([]models.Transaction, error) {
	txs := make([]models.Transaction, 0)

	query := `
		SELECT id, escrow_id, tx_hash, tx_type, block_number, block_hash, status, gas_used, created_at, confirmed_at
		FROM transactions
		WHERE escrow_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, escrowID)
	if err != nil {
		return nil, fmt.Errorf("list transactions by escrow: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.EscrowID,
			&tx.TxHash,
			&tx.TxType,
			&tx.BlockNumber,
			&tx.BlockHash,
			&tx.Status,
			&tx.GasUsed,
			&tx.CreatedAt,
			&tx.ConfirmedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

func (r *TransactionRepo) MarkConfirmed(ctx context.Context, txHash string) error {
	query := `UPDATE transactions SET status = $1, confirmed_at = now() WHERE tx_hash = $2`
	_, err := r.pool.Exec(ctx, query, models.TxStatusConfirmed, txHash)
	if err != nil {
		return fmt.Errorf("mark transaction confirmed: %w", err)
	}
	return nil
}

func (r *TransactionRepo) MarkReorged(ctx context.Context, txHash string) error {
	query := `UPDATE transactions SET status = $1 WHERE tx_hash = $2`
	_, err := r.pool.Exec(ctx, query, models.TxStatusReorged, txHash)
	if err != nil {
		return fmt.Errorf("mark transaction reorged: %w", err)
	}
	return nil
}

func (r *TransactionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM transactions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	return nil
}
