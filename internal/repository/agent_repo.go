package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentRepo struct {
	pool *pgxpool.Pool
}

func NewAgentRepo(pool *pgxpool.Pool) *AgentRepo {
	return &AgentRepo{pool: pool}
}

func (r *AgentRepo) Create(ctx context.Context, agent *models.Agent) error {
	query := `
		INSERT INTO agents (operator_id, wallet_address, display_name, api_key_hash, is_ai, skill_manifest, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		agent.OperatorID,
		agent.WalletAddress,
		agent.DisplayName,
		agent.APIKeyHash,
		agent.IsAI,
		agent.SkillManifest,
		agent.Status,
	).Scan(&agent.ID, &agent.CreatedAt, &agent.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

func (r *AgentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var agent models.Agent
	query := `
		SELECT id, operator_id, wallet_address, display_name, api_key_hash, is_ai, skill_manifest, status, created_at, updated_at
		FROM agents
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&agent.ID,
		&agent.OperatorID,
		&agent.WalletAddress,
		&agent.DisplayName,
		&agent.APIKeyHash,
		&agent.IsAI,
		&agent.SkillManifest,
		&agent.Status,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get agent by id: %w", err)
	}
	return &agent, nil
}

func (r *AgentRepo) GetByWallet(ctx context.Context, walletAddress string) (*models.Agent, error) {
	var agent models.Agent
	query := `
		SELECT id, operator_id, wallet_address, display_name, api_key_hash, is_ai, skill_manifest, status, created_at, updated_at
		FROM agents
		WHERE wallet_address = $1`

	err := r.pool.QueryRow(ctx, query, walletAddress).Scan(
		&agent.ID,
		&agent.OperatorID,
		&agent.WalletAddress,
		&agent.DisplayName,
		&agent.APIKeyHash,
		&agent.IsAI,
		&agent.SkillManifest,
		&agent.Status,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get agent by wallet: %w", err)
	}
	return &agent, nil
}

func (r *AgentRepo) ListByOperator(ctx context.Context, operatorID uuid.UUID) ([]models.Agent, error) {
	agents := make([]models.Agent, 0)

	query := `
		SELECT id, operator_id, wallet_address, display_name, api_key_hash, is_ai, skill_manifest, status, created_at, updated_at
		FROM agents
		WHERE operator_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, operatorID)
	if err != nil {
		return nil, fmt.Errorf("list agents by operator: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agent models.Agent
		if err := rows.Scan(
			&agent.ID,
			&agent.OperatorID,
			&agent.WalletAddress,
			&agent.DisplayName,
			&agent.APIKeyHash,
			&agent.IsAI,
			&agent.SkillManifest,
			&agent.Status,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func (r *AgentRepo) Update(ctx context.Context, agent *models.Agent) error {
	query := `
		UPDATE agents
		SET display_name = $1, api_key_hash = $2, skill_manifest = $3, status = $4, updated_at = now()
		WHERE id = $5
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		agent.DisplayName,
		agent.APIKeyHash,
		agent.SkillManifest,
		agent.Status,
		agent.ID,
	).Scan(&agent.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	return nil
}

func (r *AgentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM agents WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	return nil
}
