package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DisputeRepo struct {
	pool *pgxpool.Pool
}

func NewDisputeRepo(pool *pgxpool.Pool) *DisputeRepo {
	return &DisputeRepo{pool: pool}
}

func (r *DisputeRepo) Create(ctx context.Context, dispute *models.Dispute) error {
	query := `
		INSERT INTO disputes (
			task_id, poster_worker_id, assigned_worker_id, raised_by_worker_id,
			reason, status, evidence_deadline, bond_response_deadline,
			sla_response_deadline, sla_resolution_deadline
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		dispute.TaskID,
		dispute.PosterWorkerID,
		dispute.AssignedWorkerID,
		dispute.RaisedByWorkerID,
		dispute.Reason,
		dispute.Status,
		dispute.EvidenceDeadline,
		dispute.BondResponseDeadline,
		dispute.SLAResponseDeadline,
		dispute.SLAResolutionDeadline,
	).Scan(&dispute.ID, &dispute.CreatedAt)

	if err != nil {
		return fmt.Errorf("create dispute: %w", err)
	}
	return nil
}

func (r *DisputeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Dispute, error) {
	var dispute models.Dispute
	query := `
		SELECT id, task_id, poster_worker_id, assigned_worker_id, raised_by_worker_id,
			   reason, status, outcome, agent_bps, rationale,
			   evidence_deadline, bond_response_deadline, sla_response_deadline, sla_resolution_deadline,
			   created_at, resolved_at
		FROM disputes
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&dispute.ID,
		&dispute.TaskID,
		&dispute.PosterWorkerID,
		&dispute.AssignedWorkerID,
		&dispute.RaisedByWorkerID,
		&dispute.Reason,
		&dispute.Status,
		&dispute.Outcome,
		&dispute.AgentBps,
		&dispute.Rationale,
		&dispute.EvidenceDeadline,
		&dispute.BondResponseDeadline,
		&dispute.SLAResponseDeadline,
		&dispute.SLAResolutionDeadline,
		&dispute.CreatedAt,
		&dispute.ResolvedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get dispute by id: %w", err)
	}
	return &dispute, nil
}

func (r *DisputeRepo) GetByTaskID(ctx context.Context, taskID uuid.UUID) (*models.Dispute, error) {
	var dispute models.Dispute
	query := `
		SELECT id, task_id, poster_worker_id, assigned_worker_id, raised_by_worker_id,
			   reason, status, outcome, agent_bps, rationale,
			   evidence_deadline, bond_response_deadline, sla_response_deadline, sla_resolution_deadline,
			   created_at, resolved_at
		FROM disputes
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.pool.QueryRow(ctx, query, taskID).Scan(
		&dispute.ID,
		&dispute.TaskID,
		&dispute.PosterWorkerID,
		&dispute.AssignedWorkerID,
		&dispute.RaisedByWorkerID,
		&dispute.Reason,
		&dispute.Status,
		&dispute.Outcome,
		&dispute.AgentBps,
		&dispute.Rationale,
		&dispute.EvidenceDeadline,
		&dispute.BondResponseDeadline,
		&dispute.SLAResponseDeadline,
		&dispute.SLAResolutionDeadline,
		&dispute.CreatedAt,
		&dispute.ResolvedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get dispute by task id: %w", err)
	}
	return &dispute, nil
}

func (r *DisputeRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE disputes SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update dispute status: %w", err)
	}
	return nil
}

func (r *DisputeRepo) Resolve(ctx context.Context, id uuid.UUID, outcome string, agentBps *int16, rationale string) error {
	query := `
		UPDATE disputes
		SET status = $1, outcome = $2, agent_bps = $3, rationale = $4, resolved_at = now()
		WHERE id = $5`

	_, err := r.pool.Exec(ctx, query, models.DisputeStatusResolved, outcome, agentBps, rationale, id)
	if err != nil {
		return fmt.Errorf("resolve dispute: %w", err)
	}
	return nil
}

func (r *DisputeRepo) List(ctx context.Context, status *string, limit, offset int) ([]models.Dispute, error) {
	disputes := make([]models.Dispute, 0)

	query := `
		SELECT id, task_id, poster_worker_id, assigned_worker_id, raised_by_worker_id,
			   reason, status, outcome, agent_bps, rationale,
			   evidence_deadline, bond_response_deadline, sla_response_deadline, sla_resolution_deadline,
			   created_at, resolved_at
		FROM disputes
		WHERE ($1::text IS NULL OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list disputes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dispute models.Dispute
		if err := rows.Scan(
			&dispute.ID,
			&dispute.TaskID,
			&dispute.PosterWorkerID,
			&dispute.AssignedWorkerID,
			&dispute.RaisedByWorkerID,
			&dispute.Reason,
			&dispute.Status,
			&dispute.Outcome,
			&dispute.AgentBps,
			&dispute.Rationale,
			&dispute.EvidenceDeadline,
			&dispute.BondResponseDeadline,
			&dispute.SLAResponseDeadline,
			&dispute.SLAResolutionDeadline,
			&dispute.CreatedAt,
			&dispute.ResolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dispute: %w", err)
		}
		disputes = append(disputes, dispute)
	}

	return disputes, nil
}

func (r *DisputeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM disputes WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete dispute: %w", err)
	}
	return nil
}
