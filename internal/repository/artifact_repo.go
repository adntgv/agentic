package repository

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArtifactRepo struct {
	pool *pgxpool.Pool
}

func NewArtifactRepo(pool *pgxpool.Pool) *ArtifactRepo {
	return &ArtifactRepo{pool: pool}
}

func (r *ArtifactRepo) Create(ctx context.Context, artifact *models.Artifact) error {
	query := `
		INSERT INTO artifacts (task_id, worker_id, context, kind, sha256, url, text_body, mime, bytes, status, av_scan_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		artifact.TaskID,
		artifact.WorkerID,
		artifact.Context,
		artifact.Kind,
		artifact.SHA256,
		artifact.URL,
		artifact.TextBody,
		artifact.MIME,
		artifact.Bytes,
		artifact.Status,
		artifact.AVScanStatus,
	).Scan(&artifact.ID, &artifact.CreatedAt)

	if err != nil {
		return fmt.Errorf("create artifact: %w", err)
	}
	return nil
}

func (r *ArtifactRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Artifact, error) {
	var artifact models.Artifact
	query := `
		SELECT id, task_id, worker_id, context, kind, sha256, url, text_body, mime, bytes, status, av_scan_status, created_at, finalized_at
		FROM artifacts
		WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&artifact.ID,
		&artifact.TaskID,
		&artifact.WorkerID,
		&artifact.Context,
		&artifact.Kind,
		&artifact.SHA256,
		&artifact.URL,
		&artifact.TextBody,
		&artifact.MIME,
		&artifact.Bytes,
		&artifact.Status,
		&artifact.AVScanStatus,
		&artifact.CreatedAt,
		&artifact.FinalizedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get artifact by id: %w", err)
	}
	return &artifact, nil
}

func (r *ArtifactRepo) ListByTask(ctx context.Context, taskID uuid.UUID, contextFilter *string) ([]models.Artifact, error) {
	artifacts := make([]models.Artifact, 0)

	query := `
		SELECT id, task_id, worker_id, context, kind, sha256, url, text_body, mime, bytes, status, av_scan_status, created_at, finalized_at
		FROM artifacts
		WHERE task_id = $1
		  AND ($2::text IS NULL OR context = $2)
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, taskID, contextFilter)
	if err != nil {
		return nil, fmt.Errorf("list artifacts by task: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var artifact models.Artifact
		if err := rows.Scan(
			&artifact.ID,
			&artifact.TaskID,
			&artifact.WorkerID,
			&artifact.Context,
			&artifact.Kind,
			&artifact.SHA256,
			&artifact.URL,
			&artifact.TextBody,
			&artifact.MIME,
			&artifact.Bytes,
			&artifact.Status,
			&artifact.AVScanStatus,
			&artifact.CreatedAt,
			&artifact.FinalizedAt,
		); err != nil {
			return nil, fmt.Errorf("scan artifact: %w", err)
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func (r *ArtifactRepo) Finalize(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE artifacts SET status = $1, finalized_at = now() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, models.ArtifactStatusFinalized, id)
	if err != nil {
		return fmt.Errorf("finalize artifact: %w", err)
	}
	return nil
}

func (r *ArtifactRepo) UpdateAVScanStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE artifacts SET av_scan_status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update av scan status: %w", err)
	}
	return nil
}

func (r *ArtifactRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM artifacts WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	return nil
}
