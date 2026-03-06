package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (wallet_address, email, display_name, user_type, status, kyc_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.WalletAddress,
		user.Email,
		user.DisplayName,
		user.UserType,
		user.Status,
		user.KYCVerified,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, wallet_address, email, display_name, user_type, status, kyc_verified, created_at, updated_at
		FROM users
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.WalletAddress,
		&user.Email,
		&user.DisplayName,
		&user.UserType,
		&user.Status,
		&user.KYCVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByWallet(ctx context.Context, walletAddress string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, wallet_address, email, display_name, user_type, status, kyc_verified, created_at, updated_at
		FROM users
		WHERE wallet_address = $1`

	err := r.pool.QueryRow(ctx, query, walletAddress).Scan(
		&user.ID,
		&user.WalletAddress,
		&user.Email,
		&user.DisplayName,
		&user.UserType,
		&user.Status,
		&user.KYCVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get user by wallet: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, display_name = $2, user_type = $3, status = $4, kyc_verified = $5, updated_at = now()
		WHERE id = $6
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.Email,
		user.DisplayName,
		user.UserType,
		user.Status,
		user.KYCVerified,
		user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// List returns users with optional status filter
func (r *UserRepo) List(ctx context.Context, status *string, limit, offset int) ([]models.User, error) {
	users := make([]models.User, 0)

	query := `
		SELECT id, wallet_address, email, display_name, user_type, status, kyc_verified, created_at, updated_at
		FROM users
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.WalletAddress,
			&user.Email,
			&user.DisplayName,
			&user.UserType,
			&user.Status,
			&user.KYCVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
