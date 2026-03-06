package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BidService handles bid operations
type BidService struct {
	bidRepo    *repository.BidRepo
	taskRepo   *repository.TaskRepo
	outboxRepo *repository.OutboxRepo
}

// NewBidService creates a new BidService instance
func NewBidService(
	bidRepo *repository.BidRepo,
	taskRepo *repository.TaskRepo,
	outboxRepo *repository.OutboxRepo,
) *BidService {
	return &BidService{
		bidRepo:    bidRepo,
		taskRepo:   taskRepo,
		outboxRepo: outboxRepo,
	}
}

// PlaceBid creates a new bid on a task
func (s *BidService) PlaceBid(ctx context.Context, taskID, workerID uuid.UUID, amount decimal.Decimal, etaHours int, coverLetter string) (*models.Bid, error) {
	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate task status
	if task.Status != models.TaskStatusPublished && task.Status != models.TaskStatusBidding {
		return nil, fmt.Errorf("cannot bid on task in status: %s", task.Status)
	}

	// Validate bid deadline
	if task.BidDeadline != nil && time.Now().After(*task.BidDeadline) {
		return nil, fmt.Errorf("bid deadline has passed")
	}

	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("bid amount must be positive")
	}

	// Create bid
	bid := &models.Bid{
		ID:        uuid.New(),
		TaskID:    taskID,
		WorkerID:  workerID,
		Amount:    amount,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Set optional pointer fields
	if etaHours > 0 {
		bid.EtaHours = &etaHours
	}
	if coverLetter != "" {
		bid.CoverLetter = &coverLetter
	}

	// TODO: Compute bidHash (requires chain.BidHash utility from Agent 4)
	// bidHash := chain.ComputeBidHash(chainID, contractAddr, taskID, agent, amount, etaHours, createdAt, bidID)
	// bid.BidHash = bidHash

	if err := s.bidRepo.Create(ctx, bid); err != nil {
		return nil, fmt.Errorf("create bid: %w", err)
	}

	// Update task status to bidding if first bid
	if task.Status == models.TaskStatusPublished {
		task.Status = models.TaskStatusBidding
		if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusBidding); err != nil {
			return nil, fmt.Errorf("update task status: %w", err)
		}
	}

	return bid, nil
}

// AcceptBid accepts a bid and returns deposit parameters
func (s *BidService) AcceptBid(ctx context.Context, taskID, bidID, posterWorkerID uuid.UUID) (*DepositParams, error) {
	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is poster
	if task.PosterWorkerID != posterWorkerID {
		return nil, fmt.Errorf("unauthorized: only poster can accept bid")
	}

	// Validate task status
	if task.Status != models.TaskStatusBidding && task.Status != models.TaskStatusPublished {
		return nil, fmt.Errorf("can only accept bid on published/bidding task")
	}

	// Get bid
	bid, err := s.bidRepo.GetByID(ctx, bidID)
	if err != nil {
		return nil, fmt.Errorf("get bid: %w", err)
	}

	if bid.TaskID != taskID {
		return nil, fmt.Errorf("bid does not belong to this task")
	}

	// Update bid status
	if err := s.bidRepo.UpdateStatus(ctx, bidID, "accepted"); err != nil {
		return nil, fmt.Errorf("update bid: %w", err)
	}
	bid.Status = "accepted"

	// Update task
	task.Status = models.TaskStatusPendingDeposit
	task.AcceptedBidID = &bidID
	task.AssignedWorkerID = &bid.WorkerID
	task.UpdatedAt = time.Now()

	// TODO: Compute taskIDHash (requires chain utilities from Agent 4)
	taskIDHash := "0x..." // placeholder

	if err := s.taskRepo.SetAssigned(ctx, taskID, bid.WorkerID, bidID, taskIDHash); err != nil {
		return nil, fmt.Errorf("set assigned: %w", err)
	}

	// TODO: Get worker's payee address (requires WorkerService from Agent 2)
	payeeAddr := "0x..." // placeholder

	// TODO: Compute taskIDBytes32 and bidHash (requires chain utilities from Agent 4)
	// taskIDBytes32 := chain.TaskIDFromUUID(taskID)
	// bidHash := chain.ComputeBidHash(...)

	bidHash := ""
	if bid.BidHash != nil {
		bidHash = *bid.BidHash
	}

	// Return deposit parameters
	depositParams := &DepositParams{
		TaskIDBytes32:         "0x...", // placeholder
		PayeeAddress:          payeeAddr,
		Amount:                bid.Amount,
		BidHash:               bidHash,
		USDCAddress:           "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // Base USDC
		EscrowContractAddress: "0x...",                                       // from config
		DepositDeadline:       time.Now().Add(24 * time.Hour),
	}

	return depositParams, nil
}

// List lists bids for a task
func (s *BidService) List(ctx context.Context, taskID uuid.UUID) ([]models.Bid, error) {
	return s.bidRepo.ListByTask(ctx, taskID, nil)
}
