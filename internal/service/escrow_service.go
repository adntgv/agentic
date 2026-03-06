package service

import (
	"context"
	"fmt"

	"github.com/aid/agentic/internal/models"
	"github.com/aid/agentic/internal/repository"
	"github.com/google/uuid"
)

// EscrowService handles escrow operations and relayer interactions
type EscrowService struct {
	// relayer will be added by Agent 4 (chain.Relayer)
	// For now, placeholder methods that will call the relayer
	escrowRepo *repository.EscrowRepo
	txRepo     *repository.TransactionRepo
	outboxRepo *repository.OutboxRepo
}

// NewEscrowService creates a new EscrowService instance
func NewEscrowService(
	escrowRepo *repository.EscrowRepo,
	txRepo *repository.TransactionRepo,
	outboxRepo *repository.OutboxRepo,
) *EscrowService {
	return &EscrowService{
		escrowRepo: escrowRepo,
		txRepo:     txRepo,
		outboxRepo: outboxRepo,
	}
}

// Release triggers on-chain release (5% fee to agent, called by relayer)
func (s *EscrowService) Release(ctx context.Context, taskID uuid.UUID) (string, error) {
	// Get escrow
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get escrow: %w", err)
	}

	// Validate escrow status
	if escrow.Status != "locked" {
		return "", fmt.Errorf("escrow must be locked to release")
	}

	// TODO: Call relayer.Release(taskIDBytes32) - Agent 4 will provide this
	// txHash, err := s.relayer.Release(ctx, taskIDBytes32)
	// if err != nil {
	//     return "", fmt.Errorf("relayer release: %w", err)
	// }

	txHash := "0x..." // placeholder

	// Write to outbox for event tracking
	if err := writeOutboxEvent(ctx, s.outboxRepo, "escrow.release", map[string]interface{}{
		"task_id": taskID,
		"tx_hash": txHash,
	}); err != nil {
		return "", fmt.Errorf("write outbox: %w", err)
	}

	return txHash, nil
}

// Refund triggers on-chain refund (0% fee, full amount to poster)
func (s *EscrowService) Refund(ctx context.Context, taskID uuid.UUID) (string, error) {
	// Get escrow
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get escrow: %w", err)
	}

	// Validate escrow status
	if escrow.Status != "locked" {
		return "", fmt.Errorf("escrow must be locked to refund")
	}

	// TODO: Call relayer.Refund(taskIDBytes32) - Agent 4 will provide this
	// txHash, err := s.relayer.Refund(ctx, taskIDBytes32)
	// if err != nil {
	//     return "", fmt.Errorf("relayer refund: %w", err)
	// }

	txHash := "0x..." // placeholder

	// Write to outbox
	if err := writeOutboxEvent(ctx, s.outboxRepo, "escrow.refund", map[string]interface{}{
		"task_id": taskID,
		"tx_hash": txHash,
	}); err != nil {
		return "", fmt.Errorf("write outbox: %w", err)
	}

	return txHash, nil
}

// Split triggers on-chain split (0% fee from escrow, custom split percentage)
func (s *EscrowService) Split(ctx context.Context, taskID uuid.UUID, agentBps uint16) (string, error) {
	// Validate agentBps
	if agentBps > 10000 {
		return "", fmt.Errorf("agentBps must be <= 10000")
	}

	// Get escrow
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get escrow: %w", err)
	}

	// Validate escrow status
	if escrow.Status != "locked" {
		return "", fmt.Errorf("escrow must be locked to split")
	}

	// TODO: Call relayer.Split(taskIDBytes32, agentBps) - Agent 4 will provide this
	// txHash, err := s.relayer.Split(ctx, taskIDBytes32, agentBps)
	// if err != nil {
	//     return "", fmt.Errorf("relayer split: %w", err)
	// }

	txHash := "0x..." // placeholder

	// Write to outbox
	if err := writeOutboxEvent(ctx, s.outboxRepo, "escrow.split", map[string]interface{}{
		"task_id":   taskID,
		"tx_hash":   txHash,
		"agent_bps": agentBps,
	}); err != nil {
		return "", fmt.Errorf("write outbox: %w", err)
	}

	return txHash, nil
}

// Reassign triggers on-chain agent reassignment (only if submitted==false)
func (s *EscrowService) Reassign(ctx context.Context, taskID uuid.UUID, newAgentAddr string) (string, error) {
	// Get escrow
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get escrow: %w", err)
	}

	// Validate escrow status
	if escrow.Status != "locked" {
		return "", fmt.Errorf("escrow must be locked to reassign")
	}

	// TODO: Check submitted flag (Agent 2 will add this field to escrow model)
	// if escrow.Submitted {
	//     return "", fmt.Errorf("cannot reassign after submission")
	// }

	// TODO: Call relayer.ReassignAgent(taskIDBytes32, newAgentAddr) - Agent 4 will provide this
	// txHash, err := s.relayer.ReassignAgent(ctx, taskIDBytes32, newAgentAddr)
	// if err != nil {
	//     return "", fmt.Errorf("relayer reassign: %w", err)
	// }

	txHash := "0x..." // placeholder

	// Write to outbox
	if err := writeOutboxEvent(ctx, s.outboxRepo, "escrow.reassigned", map[string]interface{}{
		"task_id":        taskID,
		"tx_hash":        txHash,
		"new_agent_addr": newAgentAddr,
	}); err != nil {
		return "", fmt.Errorf("write outbox: %w", err)
	}

	return txHash, nil
}

// MarkSubmitted triggers on-chain markSubmitted (relayer calls after validating submission)
func (s *EscrowService) MarkSubmitted(ctx context.Context, taskID uuid.UUID) (string, error) {
	// Get escrow
	escrow, err := s.escrowRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get escrow: %w", err)
	}

	// Validate escrow status
	if escrow.Status != "locked" {
		return "", fmt.Errorf("escrow must be locked to mark submitted")
	}

	// TODO: Call relayer.MarkSubmitted(taskIDBytes32) - Agent 4 will provide this
	// txHash, err := s.relayer.MarkSubmitted(ctx, taskIDBytes32)
	// if err != nil {
	//     return "", fmt.Errorf("relayer mark submitted: %w", err)
	// }

	txHash := "0x..." // placeholder

	// Write to outbox (includes submission_id and submission_timestamp for audit per spec)
	if err := writeOutboxEvent(ctx, s.outboxRepo, "escrow.mark_submitted", map[string]interface{}{
		"task_id":              taskID,
		"tx_hash":              txHash,
		"submission_id":        taskID.String(), // placeholder
		"submission_timestamp": "2026-03-06T00:00:00Z",
	}); err != nil {
		return "", fmt.Errorf("write outbox: %w", err)
	}

	return txHash, nil
}

// Get retrieves escrow details for a task
func (s *EscrowService) Get(ctx context.Context, taskID uuid.UUID) (*models.Escrow, error) {
	return s.escrowRepo.GetByTaskID(ctx, taskID)
}
