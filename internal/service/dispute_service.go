package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DisputeService handles dispute lifecycle and bond verification
type DisputeService struct {
	disputeRepo *repository.DisputeRepo
	bondRepo    *repository.DisputeBondRepo
	taskRepo    *repository.TaskRepo
	escrowRepo  *repository.EscrowRepo
	outboxRepo  *repository.OutboxRepo
	// bondVerifier from Agent 4 (chain.BondVerifier)
}

// NewDisputeService creates a new DisputeService instance
func NewDisputeService(
	disputeRepo *repository.DisputeRepo,
	bondRepo *repository.DisputeBondRepo,
	taskRepo *repository.TaskRepo,
	escrowRepo *repository.EscrowRepo,
	outboxRepo *repository.OutboxRepo,
) *DisputeService {
	return &DisputeService{
		disputeRepo: disputeRepo,
		bondRepo:    bondRepo,
		taskRepo:    taskRepo,
		escrowRepo:  escrowRepo,
		outboxRepo:  outboxRepo,
	}
}

// Raise creates a new dispute (raiser posts 1% bond)
func (s *DisputeService) Raise(ctx context.Context, taskID, raiserWorkerID uuid.UUID, reason, bondTxHash string) (*models.Dispute, error) {
	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Validate task status (can dispute from review or in_progress)
	if task.Status != models.TaskStatusReview && task.Status != models.TaskStatusInProgress {
		return nil, fmt.Errorf("can only dispute task in review or in_progress status")
	}

	// Validate raiser is a party (poster or assigned worker)
	if task.PosterWorkerID != raiserWorkerID && (task.AssignedWorkerID == nil || *task.AssignedWorkerID != raiserWorkerID) {
		return nil, fmt.Errorf("unauthorized: only task parties can raise dispute")
	}

	// Get escrow to compute bond amount
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get escrow: %w", err)
	}

	// Compute bond amount (1% of escrow)
	bondAmount := escrow.Amount.Div(decimal.NewFromInt(100))

	// TODO: Verify bond tx using chain.BondVerifier (Agent 4)
	// if err := s.bondVerifier.VerifyBond(ctx, bondTxHash, bondAmount); err != nil {
	//     return nil, fmt.Errorf("invalid bond transaction: %w", err)
	// }

	// Create dispute
	now := time.Now()
	evidenceDeadline := now.Add(48 * time.Hour)
	bondResponseDeadline := now.Add(48 * time.Hour)
	slaResponseDeadline := now.Add(48 * time.Hour)
	slaResolutionDeadline := now.Add(7 * 24 * time.Hour)

	dispute := &models.Dispute{
		ID:                    uuid.New(),
		TaskID:                taskID,
		PosterWorkerID:        task.PosterWorkerID,
		AssignedWorkerID:      *task.AssignedWorkerID,
		RaisedByWorkerID:      raiserWorkerID,
		Reason:                reason,
		Status:                "raised",
		EvidenceDeadline:      &evidenceDeadline,
		BondResponseDeadline:  &bondResponseDeadline,
		SLAResponseDeadline:   &slaResponseDeadline,
		SLAResolutionDeadline: &slaResolutionDeadline,
		CreatedAt:             now,
	}

	if err := s.disputeRepo.Create(ctx, dispute); err != nil {
		return nil, fmt.Errorf("create dispute: %w", err)
	}

	// Create bond record
	bond := &models.DisputeBond{
		ID:         uuid.New(),
		DisputeID:  dispute.ID,
		WorkerID:   raiserWorkerID,
		Role:       "raiser",
		Amount:     bondAmount,
		TxHash:     bondTxHash,
		Status:     "posted",
		CreatedAt:  time.Now(),
	}

	if err := s.bondRepo.Create(ctx, bond); err != nil {
		return nil, fmt.Errorf("create bond: %w", err)
	}

	// Update task status
	task.Status = models.TaskStatusDisputed
	if err := s.taskRepo.UpdateStatus(ctx, taskID, models.TaskStatusDisputed); err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}

	// Write outbox event
	payloadBytes, _ := json.Marshal(dispute)
	var payload models.JSONBData
	json.Unmarshal(payloadBytes, &payload)

	event := &models.OutboxEvent{
		EventType:      "task.disputed",
		Payload:        payload,
		IdempotencyKey: fmt.Sprintf("dispute-%s", dispute.ID.String()),
		Status:         "pending",
		RetryCount:     0,
		CreatedAt:      time.Now(),
	}

	if err := s.outboxRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return dispute, nil
}

// Respond responds to a dispute (responder posts 1% bond within 48h)
func (s *DisputeService) Respond(ctx context.Context, disputeID, responderWorkerID uuid.UUID, bondTxHash string) (*models.Dispute, error) {
	// Get dispute
	dispute, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, fmt.Errorf("get dispute: %w", err)
	}

	// Validate dispute status
	if dispute.Status != "raised" {
		return nil, fmt.Errorf("can only respond to raised dispute")
	}

	// Validate responder is the other party
	if dispute.RaisedByWorkerID == responderWorkerID {
		return nil, fmt.Errorf("raiser cannot respond to their own dispute")
	}

	if dispute.PosterWorkerID != responderWorkerID && dispute.AssignedWorkerID != responderWorkerID {
		return nil, fmt.Errorf("unauthorized: only task parties can respond")
	}

	// Check response deadline
	if dispute.BondResponseDeadline != nil && time.Now().After(*dispute.BondResponseDeadline) {
		return nil, fmt.Errorf("bond response deadline has passed")
	}

	// Get escrow to compute bond amount
	escrow, err := s.escrowRepo.GetByTaskID(ctx, dispute.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get escrow: %w", err)
	}

	bondAmount := escrow.Amount.Div(decimal.NewFromInt(100))

	// TODO: Verify bond tx using chain.BondVerifier (Agent 4)

	// Create bond record
	bond := &models.DisputeBond{
		ID:         uuid.New(),
		DisputeID:  dispute.ID,
		WorkerID:   responderWorkerID,
		Role:       "responder",
		Amount:     bondAmount,
		TxHash:     bondTxHash,
		Status:     "posted",
		CreatedAt:  time.Now(),
	}

	if err := s.bondRepo.Create(ctx, bond); err != nil {
		return nil, fmt.Errorf("create bond: %w", err)
	}

	// Update dispute status
	dispute.Status = "evidence"
	if err := s.disputeRepo.UpdateStatus(ctx, disputeID, "evidence"); err != nil {
		return nil, fmt.Errorf("update dispute status: %w", err)
	}

	return dispute, nil
}

// SubmitEvidence submits evidence for a dispute
func (s *DisputeService) SubmitEvidence(ctx context.Context, disputeID, workerID uuid.UUID, artifactIDs []uuid.UUID) error {
	// Get dispute
	dispute, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return fmt.Errorf("get dispute: %w", err)
	}

	// Validate dispute status
	if dispute.Status != "evidence" {
		return fmt.Errorf("can only submit evidence during evidence phase")
	}

	// Validate caller is a party
	if dispute.PosterWorkerID != workerID && dispute.AssignedWorkerID != workerID {
		return fmt.Errorf("unauthorized: only task parties can submit evidence")
	}

	// Validate evidence deadline
	if dispute.EvidenceDeadline != nil && time.Now().After(*dispute.EvidenceDeadline) {
		return fmt.Errorf("evidence deadline has passed")
	}

	// TODO: Validate all artifacts are finalized (Agent 2)
	// TODO: Link artifacts to dispute (Agent 2)

	return nil
}

// Ruling issues an admin ruling on a dispute
func (s *DisputeService) Ruling(ctx context.Context, disputeID, adminWorkerID uuid.UUID, outcome string, agentBps *int16, rationale string) (*models.Dispute, error) {
	// TODO: Validate admin role

	// Get dispute
	dispute, err := s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, fmt.Errorf("get dispute: %w", err)
	}

	// Validate dispute status
	if dispute.Status != "evidence" && dispute.Status != "arbitration" {
		return nil, fmt.Errorf("can only issue ruling during evidence/arbitration phase")
	}

	// Validate outcome
	switch outcome {
	case "agent_wins", "poster_wins", "split":
		// valid
	default:
		return nil, fmt.Errorf("invalid outcome: %s", outcome)
	}

	if outcome == "split" && agentBps == nil {
		return nil, fmt.Errorf("agentBps required for split outcome")
	}

	// Update dispute
	if err := s.disputeRepo.Resolve(ctx, disputeID, outcome, agentBps, rationale); err != nil {
		return nil, fmt.Errorf("resolve dispute: %w", err)
	}

	// Refresh dispute object after resolving
	dispute, err = s.disputeRepo.GetByID(ctx, disputeID)
	if err != nil {
		return nil, fmt.Errorf("get updated dispute: %w", err)
	}

	// TODO: Execute on-chain action via EscrowService
	// TODO: Settle bonds (winner returned, loser retained)

	// Write outbox event
	payloadBytes, _ := json.Marshal(dispute)
	var payload models.JSONBData
	json.Unmarshal(payloadBytes, &payload)

	event := &models.OutboxEvent{
		EventType:      "dispute.resolved",
		Payload:        payload,
		IdempotencyKey: fmt.Sprintf("resolve-%s", dispute.ID.String()),
		Status:         "pending",
		RetryCount:     0,
		CreatedAt:      time.Now(),
	}

	if err := s.outboxRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("write outbox: %w", err)
	}

	return dispute, nil
}
