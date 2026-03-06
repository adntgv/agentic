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

// TaskService handles all task lifecycle operations and state transitions
type TaskService struct {
	taskRepo     *repository.TaskRepo
	bidRepo      *repository.BidRepo
	escrowRepo   *repository.EscrowRepo
	outboxRepo   *repository.OutboxRepo
	escrowSvc    *EscrowService
	schedulerSvc *SchedulerService
}

// NewTaskService creates a new TaskService instance
func NewTaskService(
	taskRepo *repository.TaskRepo,
	bidRepo *repository.BidRepo,
	escrowRepo *repository.EscrowRepo,
	outboxRepo *repository.OutboxRepo,
	escrowSvc *EscrowService,
	schedulerSvc *SchedulerService,
) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		bidRepo:      bidRepo,
		escrowRepo:   escrowRepo,
		outboxRepo:   outboxRepo,
		escrowSvc:    escrowSvc,
		schedulerSvc: schedulerSvc,
	}
}

// Create creates a new task in draft status
func (s *TaskService) Create(ctx context.Context, posterWorkerID uuid.UUID, req CreateTaskRequest) (*models.Task, error) {
	// Validate minimum budget ($10 USDC)
	minBudget := decimal.NewFromFloat(10.0)
	if req.Budget.LessThan(minBudget) {
		return nil, fmt.Errorf("budget must be at least $10 USDC")
	}

	task := &models.Task{
		ID:              uuid.New(),
		PosterWorkerID:  posterWorkerID,
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		Budget:          req.Budget,
		Deadline:        req.Deadline,
		BidDeadline:     req.BidDeadline,
		WorkerFilter:    req.WorkerFilter,
		MaxRevisions:    2, // default
		RevisionCount:   0,
		Status:          models.TaskStatusDraft,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	return task, nil
}

// Publish transitions task from draft to published
func (s *TaskService) Publish(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is poster
	if task.PosterWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only poster can publish")
	}

	// Validate current status
	if task.Status != models.TaskStatusDraft {
		return nil, fmt.Errorf("can only publish draft tasks")
	}

	// Validate required fields
	if task.Title == "" || task.Description == "" || task.Budget.IsZero() {
		return nil, fmt.Errorf("missing required fields")
	}

	// Update status
	task.Status = models.TaskStatusPublished
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusPublished); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.published", task); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// Cancel cancels a task (before agent ACK only)
func (s *TaskService) Cancel(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is poster
	if task.PosterWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only poster can cancel")
	}

	// Check current status - can cancel before deposit, before ACK
	switch task.Status {
	case models.TaskStatusDraft, models.TaskStatusPublished, models.TaskStatusBidding:
		// Can cancel - no on-chain action needed
		task.Status = models.TaskStatusCancelled
	case models.TaskStatusAssigned:
		// Can cancel only if not yet ACK'd - trigger refund
		task.Status = models.TaskStatusCancelled
		// Trigger on-chain refund (0% fee)
		if _, err := s.escrowSvc.Refund(ctx, taskID); err != nil {
			return nil, fmt.Errorf("refund escrow: %w", err)
		}
	case models.TaskStatusInProgress:
		// Cannot cancel after agent ACK
		return nil, fmt.Errorf("cannot cancel after agent has acknowledged - must wait for submission/dispute/admin action")
	default:
		return nil, fmt.Errorf("cannot cancel task in status: %s", task.Status)
	}

	task.UpdatedAt = time.Now()
	if err := s.taskRepo.UpdateStatus(ctx, taskID, task.Status); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.cancelled", task); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// AcceptBid accepts a bid and returns deposit parameters for the poster
func (s *TaskService) AcceptBid(ctx context.Context, taskID, bidID, callerWorkerID uuid.UUID) (*DepositParams, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is poster
	if task.PosterWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only poster can accept bid")
	}

	// Validate current status
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

	// Update task status to pending_deposit
	task.Status = models.TaskStatusPendingDeposit
	task.AcceptedBidID = &bidID
	task.AssignedWorkerID = &bid.WorkerID
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.SetAssigned(ctx, taskID, bid.WorkerID, bidID, "0x..."); err != nil {
		return nil, fmt.Errorf("set assigned: %w", err)
	}

	// Return deposit parameters (BidService will compute bidHash and taskIDBytes32)
	return nil, fmt.Errorf("call BidService.AcceptBid for deposit params")
}

// ConfirmDeposit transitions task from pending_deposit to assigned (called by indexer)
func (s *TaskService) ConfirmDeposit(ctx context.Context, taskID uuid.UUID, txHash string) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	if task.Status != models.TaskStatusPendingDeposit {
		return nil, fmt.Errorf("task is not in pending_deposit status")
	}

	// Update status to assigned
	task.Status = models.TaskStatusAssigned
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusAssigned); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.assigned", task); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// Ack acknowledges assignment (worker ACKs)
func (s *TaskService) Ack(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is assigned worker
	if task.AssignedWorkerID == nil || *task.AssignedWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only assigned worker can acknowledge")
	}

	// Validate current status
	if task.Status != models.TaskStatusAssigned {
		return nil, fmt.Errorf("can only acknowledge assigned task")
	}

	// Update status to in_progress
	task.Status = models.TaskStatusInProgress
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusInProgress); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	return task, nil
}

// Submit submits work with artifacts
func (s *TaskService) Submit(ctx context.Context, taskID, callerWorkerID uuid.UUID, artifactIDs []uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is assigned worker
	if task.AssignedWorkerID == nil || *task.AssignedWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only assigned worker can submit")
	}

	// Validate current status
	if task.Status != models.TaskStatusInProgress {
		return nil, fmt.Errorf("can only submit in_progress task")
	}

	// Validate at least 1 artifact
	if len(artifactIDs) == 0 {
		return nil, fmt.Errorf("at least one artifact required")
	}

	// TODO: Validate all artifacts are finalized (Agent 2 will add artifact validation)

	// Update status to review
	task.Status = models.TaskStatusReview
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusReview); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Call on-chain markSubmitted (relayer)
	if _, err := s.escrowSvc.MarkSubmitted(ctx, taskID); err != nil {
		return nil, fmt.Errorf("mark submitted on-chain: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.submitted", task); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// Approve approves work and triggers release
func (s *TaskService) Approve(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is poster
	if task.PosterWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only poster can approve")
	}

	// Validate current status
	if task.Status != models.TaskStatusReview {
		return nil, fmt.Errorf("can only approve task in review")
	}

	// Update status to completed
	task.Status = models.TaskStatusCompleted
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusCompleted); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Trigger on-chain release (5% fee)
	if _, err := s.escrowSvc.Release(ctx, taskID); err != nil {
		return nil, fmt.Errorf("release escrow: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.approved", task); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// RequestRevision requests revisions (max 2)
func (s *TaskService) RequestRevision(ctx context.Context, taskID, callerWorkerID uuid.UUID, feedback []RevisionFeedback) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate caller is poster
	if task.PosterWorkerID != callerWorkerID {
		return nil, fmt.Errorf("unauthorized: only poster can request revision")
	}

	// Validate current status
	if task.Status != models.TaskStatusReview {
		return nil, fmt.Errorf("can only request revision for task in review")
	}

	// Check revision count
	if task.RevisionCount >= task.MaxRevisions {
		return nil, fmt.Errorf("max revisions (%d) reached", task.MaxRevisions)
	}

	// Validate feedback is provided
	if len(feedback) == 0 {
		return nil, fmt.Errorf("structured feedback required")
	}

	// Increment revision count
	task.RevisionCount++
	task.Status = models.TaskStatusInProgress
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusInProgress); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.revision_requested", map[string]interface{}{
		"task":     task,
		"feedback": feedback,
	}); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// RaiseDispute initiates a dispute
func (s *TaskService) RaiseDispute(ctx context.Context, taskID, callerWorkerID uuid.UUID, reason, bondTxHash string) (*models.Dispute, error) {
	// Delegate to DisputeService
	return nil, fmt.Errorf("use DisputeService.Raise")
}

// AdminReassign reassigns task to a new worker (admin only, submitted==false)
func (s *TaskService) AdminReassign(ctx context.Context, taskID, newWorkerID, adminWorkerID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// TODO: Validate admin role

	// Validate task can be reassigned (must not be submitted)
	// Check on-chain submitted flag via escrow
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get escrow: %w", err)
	}
	_ = escrow // TODO: Check on-chain submitted flag

	// NOTE: Agent 2 will add "submitted" field to escrow model
	// For now, assume we can reassign if not in completed/refunded/split status

	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusRefunded || task.Status == models.TaskStatusSplit {
		return nil, fmt.Errorf("cannot reassign completed/refunded/split task")
	}

	// Update assigned worker
	task.AssignedWorkerID = &newWorkerID
	task.Status = models.TaskStatusAssigned
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.SetAssigned(ctx, taskID, newWorkerID, *task.AcceptedBidID, "0x..."); err != nil {
		return nil, fmt.Errorf("set assigned: %w", err)
	}

	// Trigger on-chain reassignAgent
	// Get new worker's payee address (TODO: Agent 2 will provide WorkerService)
	newPayeeAddr := "0x..." // placeholder
	if _, err := s.escrowSvc.Reassign(ctx, taskID, newPayeeAddr); err != nil {
		return nil, fmt.Errorf("reassign on-chain: %w", err)
	}

	// Write outbox event
	if err := writeOutboxEvent(ctx, s.outboxRepo, "task.unassigned", task); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return task, nil
}

// AdminRefund refunds a task (admin only)
func (s *TaskService) AdminRefund(ctx context.Context, taskID, adminWorkerID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// TODO: Validate admin role

	// Trigger on-chain refund
	if _, err := s.escrowSvc.Refund(ctx, taskID); err != nil {
		return nil, fmt.Errorf("refund escrow: %w", err)
	}

	task.Status = models.TaskStatusRefunded
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusRefunded); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	return task, nil
}

// DepositParams contains parameters for poster to deposit escrow on-chain
type DepositParams struct {
	TaskIDBytes32         string          `json:"task_id_bytes32"`
	PayeeAddress          string          `json:"payee_address"`
	Amount                decimal.Decimal `json:"amount"`
	BidHash               string          `json:"bid_hash"`
	USDCAddress           string          `json:"usdc_address"`
	EscrowContractAddress string          `json:"escrow_contract_address"`
	DepositDeadline       time.Time       `json:"deposit_deadline"`
}

// CreateTaskRequest contains fields for creating a task
type CreateTaskRequest struct {
	Title        string
	Description  string
	Category     *string
	Budget       decimal.Decimal
	Deadline     *time.Time
	BidDeadline  *time.Time
	WorkerFilter string // human_only | ai_only | both
}

// RevisionFeedback contains structured feedback for revision requests
type RevisionFeedback struct {
	Section  string `json:"section"`
	Issue    string `json:"issue"`
	Expected string `json:"expected"`
}
