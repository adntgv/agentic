package service

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
)

// WorkerService handles worker CRUD, auth, and profile operations
type WorkerService struct {
	workerRepo *repository.WorkerRepo
	userRepo   *repository.UserRepo
	agentRepo  *repository.AgentRepo
}

// NewWorkerService creates a new WorkerService instance
func NewWorkerService(
	workerRepo *repository.WorkerRepo,
	userRepo *repository.UserRepo,
	agentRepo *repository.AgentRepo,
) *WorkerService {
	return &WorkerService{
		workerRepo: workerRepo,
		userRepo:   userRepo,
		agentRepo:  agentRepo,
	}
}

// AuthWallet authenticates a user via SIWE (Sign-In with Ethereum)
func (s *WorkerService) AuthWallet(ctx context.Context, walletAddr, signature, message string) (string, error) {
	// TODO: Verify SIWE signature using spruceid/siwe-go
	// TODO: Get or create user
	// TODO: Get or create worker
	// TODO: Generate JWT token

	return "jwt_token_placeholder", nil
}

// AuthAPIKey authenticates an agent via API key
func (s *WorkerService) AuthAPIKey(ctx context.Context, apiKey string) (*models.Worker, error) {
	// TODO: Hash API key and lookup agent
	// TODO: Get worker for agent
	// TODO: Validate scopes

	return nil, fmt.Errorf("not implemented")
}

// GetProfile returns public profile and reputation for a worker
func (s *WorkerService) GetProfile(ctx context.Context, workerID uuid.UUID) (*models.Worker, error) {
	return s.workerRepo.GetByID(ctx, workerID)
}

// GetHistory returns completed tasks history for a worker
func (s *WorkerService) GetHistory(ctx context.Context, workerID uuid.UUID) ([]models.Task, error) {
	// TODO: Query tasks where assigned_worker_id = workerID and status = completed
	return nil, fmt.Errorf("not implemented")
}

// GetOperator returns operator profile with aggregate agent reputation
func (s *WorkerService) GetOperator(ctx context.Context, operatorID uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(ctx, operatorID)
}
