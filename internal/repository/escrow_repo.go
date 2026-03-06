package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EscrowRepo struct {
	pool *pgxpool.Pool
}

func NewEscrowRepo(pool *pgxpool.Pool) *EscrowRepo {
	return &EscrowRepo{pool: pool}
}

func (r *EscrowRepo) Create(ctx context.Context, escrow *models.Escrow) error {
	query := `
		INSERT INTO escrows (task_id, poster_address, payee_address, amount, bid_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		escrow.TaskID,
		escrow.PosterAddress,
		escrow.PayeeAddress,
		escrow.Amount,
		escrow.BidHash,
		escrow.Status,
	).Scan(&escrow.ID, &escrow.CreatedAt)

	if err != nil {
		return fmt.Errorf("create escrow: %w", err)
	}
	return nil
}

func (r *EscrowRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Escrow, error) {
	var escrow models.Escrow
	query := `
		SELECT id, task_id, poster_address, payee_address, amount, bid_hash, status,
			   deposit_tx_hash, release_tx_hash, refund_tx_hash, split_tx_hash,
			   deposited_at, resolved_at, created_at
		FROM escrows
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&escrow.ID,
		&escrow.TaskID,
		&escrow.PosterAddress,
		&escrow.PayeeAddress,
		&escrow.Amount,
		&escrow.BidHash,
		&escrow.Status,
		&escrow.DepositTxHash,
		&escrow.ReleaseTxHash,
		&escrow.RefundTxHash,
		&escrow.SplitTxHash,
		&escrow.DepositedAt,
		&escrow.ResolvedAt,
		&escrow.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get escrow by id: %w", err)
	}
	return &escrow, nil
}

func (r *EscrowRepo) GetByTaskID(ctx context.Context, taskID uuid.UUID) (*models.Escrow, error) {
	var escrow models.Escrow
	query := `
		SELECT id, task_id, poster_address, payee_address, amount, bid_hash, status,
			   deposit_tx_hash, release_tx_hash, refund_tx_hash, split_tx_hash,
			   deposited_at, resolved_at, created_at
		FROM escrows
		WHERE task_id = $1`

	err := r.pool.QueryRow(ctx, query, taskID).Scan(
		&escrow.ID,
		&escrow.TaskID,
		&escrow.PosterAddress,
		&escrow.PayeeAddress,
		&escrow.Amount,
		&escrow.BidHash,
		&escrow.Status,
		&escrow.DepositTxHash,
		&escrow.ReleaseTxHash,
		&escrow.RefundTxHash,
		&escrow.SplitTxHash,
		&escrow.DepositedAt,
		&escrow.ResolvedAt,
		&escrow.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get escrow by task id: %w", err)
	}
	return &escrow, nil
}

func (r *EscrowRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE escrows SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update escrow status: %w", err)
	}
	return nil
}

func (r *EscrowRepo) SetDepositTx(ctx context.Context, id uuid.UUID, txHash string) error {
	query := `UPDATE escrows SET deposit_tx_hash = $1, deposited_at = now(), status = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, txHash, models.EscrowStatusLocked, id)
	if err != nil {
		return fmt.Errorf("set deposit tx: %w", err)
	}
	return nil
}

func (r *EscrowRepo) SetReleaseTx(ctx context.Context, id uuid.UUID, txHash string) error {
	query := `UPDATE escrows SET release_tx_hash = $1, resolved_at = now(), status = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, txHash, models.EscrowStatusReleased, id)
	if err != nil {
		return fmt.Errorf("set release tx: %w", err)
	}
	return nil
}

func (r *EscrowRepo) SetRefundTx(ctx context.Context, id uuid.UUID, txHash string) error {
	query := `UPDATE escrows SET refund_tx_hash = $1, resolved_at = now(), status = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, txHash, models.EscrowStatusRefunded, id)
	if err != nil {
		return fmt.Errorf("set refund tx: %w", err)
	}
	return nil
}

func (r *EscrowRepo) SetSplitTx(ctx context.Context, id uuid.UUID, txHash string) error {
	query := `UPDATE escrows SET split_tx_hash = $1, resolved_at = now(), status = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, txHash, models.EscrowStatusSplit, id)
	if err != nil {
		return fmt.Errorf("set split tx: %w", err)
	}
	return nil
}

func (r *EscrowRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM escrows WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete escrow: %w", err)
	}
	return nil
}
