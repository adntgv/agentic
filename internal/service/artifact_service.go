package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
)

// ArtifactService handles artifact upload and finalization
type ArtifactService struct {
	artifactRepo *repository.ArtifactRepo
	// s3Client will be injected (from internal/s3)
}

// NewArtifactService creates a new ArtifactService instance
func NewArtifactService(artifactRepo *repository.ArtifactRepo) *ArtifactService {
	return &ArtifactService{
		artifactRepo: artifactRepo,
	}
}

// RequestUploadURL generates a pre-signed S3 URL for artifact upload
func (s *ArtifactService) RequestUploadURL(ctx context.Context, taskID, workerID uuid.UUID, req ArtifactUploadRequest) (*ArtifactUploadResponse, error) {
	// Validate kind-specific fields
	switch req.Kind {
	case "file":
		if req.SHA256 == "" || req.Bytes == 0 || req.Mime == "" {
			return nil, fmt.Errorf("sha256, bytes, and mime required for file kind")
		}
	case "text":
		if req.Text == "" {
			return nil, fmt.Errorf("text required for text kind")
		}
		// Compute SHA256 server-side for text
		// req.SHA256 = computeSHA256(req.Text)
	case "url":
		if req.URL == "" {
			return nil, fmt.Errorf("url required for url kind")
		}
	default:
		return nil, fmt.Errorf("invalid kind: %s", req.Kind)
	}

	// Create artifact record with status=pending
	artifact := &models.Artifact{
		ID:           uuid.New(),
		TaskID:       taskID,
		WorkerID:     workerID,
		Context:      req.Context,
		Kind:         req.Kind,
		Status:       "pending",
		AVScanStatus: "pending",
		CreatedAt:    time.Now(),
	}

	// Set pointer fields only if non-empty
	if req.SHA256 != "" {
		artifact.SHA256 = &req.SHA256
	}
	if req.URL != "" {
		artifact.URL = &req.URL
	}
	if req.Text != "" {
		artifact.TextBody = &req.Text
	}
	if req.Mime != "" {
		artifact.MIME = &req.Mime
	}
	if req.Bytes > 0 {
		artifact.Bytes = &req.Bytes
	}

	if err := s.artifactRepo.Create(ctx, artifact); err != nil {
		return nil, fmt.Errorf("create artifact: %w", err)
	}

	var uploadURL string
	if req.Kind == "file" {
		// TODO: Generate pre-signed S3 URL (Agent 3 - s3/client.go)
		// uploadURL = s.s3Client.GeneratePresignedUploadURL(artifact.ID, req.Mime, 15*time.Minute)
		uploadURL = "https://s3.example.com/upload?signature=..." // placeholder
	}

	// For text/url kinds, finalize immediately
	if req.Kind == "text" || req.Kind == "url" {
		if err := s.artifactRepo.Finalize(ctx, artifact.ID); err != nil {
			return nil, fmt.Errorf("finalize artifact: %w", err)
		}
		artifact.Status = "finalized"
		now := time.Now()
		artifact.FinalizedAt = &now
	}

	return &ArtifactUploadResponse{
		UploadURL:  uploadURL,
		ArtifactID: artifact.ID,
	}, nil
}

// Finalize marks an artifact as finalized (called after S3 upload completes)
func (s *ArtifactService) Finalize(ctx context.Context, artifactID uuid.UUID) error {
	artifact, err := s.artifactRepo.GetByID(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("get artifact: %w", err)
	}

	if artifact.Status != "pending" {
		return fmt.Errorf("artifact is not pending")
	}

	// TODO: Verify upload (sha256 match, size match, AV scan clean)

	return s.artifactRepo.Finalize(ctx, artifactID)
}

// List lists artifacts for a task with optional context filter
func (s *ArtifactService) List(ctx context.Context, taskID uuid.UUID, contextFilter string) ([]models.Artifact, error) {
	var filter *string
	if contextFilter != "" {
		filter = &contextFilter
	}
	return s.artifactRepo.ListByTask(ctx, taskID, filter)
}

// ArtifactUploadRequest contains fields for requesting an upload URL
type ArtifactUploadRequest struct {
	Context string `json:"context"` // submission|evidence|revision_request
	Kind    string `json:"kind"`    // file|url|text
	SHA256  string `json:"sha256"`  // required for file/text
	Bytes   int64  `json:"bytes"`   // required for file
	Mime    string `json:"mime"`    // required for file
	URL     string `json:"url"`     // for kind=url
	Text    string `json:"text"`    // for kind=text
}

// ArtifactUploadResponse contains the pre-signed URL and artifact ID
type ArtifactUploadResponse struct {
	UploadURL  string    `json:"upload_url,omitempty"` // for file kind only
	ArtifactID uuid.UUID `json:"artifact_id"`
}
