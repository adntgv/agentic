package chain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Repository interfaces that the indexer depends on
type IndexerDeps struct {
	EscrowRepo  EscrowRepository
	TxRepo      TransactionRepository
	OutboxRepo  OutboxRepository
	TaskService TaskService
}

// Placeholder interfaces - these will be implemented by other agents
type EscrowRepository interface {
	UpdateEscrowStatus(ctx context.Context, taskID [32]byte, status string) error
	GetEscrowByTaskID(ctx context.Context, taskID [32]byte) (interface{}, error)
}

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, tx interface{}) error
	GetTransactionByHash(ctx context.Context, txHash common.Hash) (interface{}, error)
	MarkConfirmed(ctx context.Context, txHash common.Hash, blockNumber uint64, blockHash common.Hash) error
	MarkReorged(ctx context.Context, txHash common.Hash) error
}

type OutboxRepository interface {
	MatchEvent(ctx context.Context, eventType string, taskID [32]byte, txHash common.Hash) (bool, error)
}

type TaskService interface {
	ConfirmDeposit(ctx context.Context, taskID [32]byte, txHash string) error
	HandleEscrowSettled(ctx context.Context, taskID [32]byte, outcome string) error
}

// Indexer polls for new blocks and processes contract events
type Indexer struct {
	client        *ethclient.Client
	contract      common.Address
	deps          IndexerDeps
	lastBlock     uint64
	confirmations uint64
	pollInterval  time.Duration
	
	mu            sync.RWMutex
	lastSafeBlock uint64
	blockHashes   map[uint64]common.Hash // For reorg detection
	
	stopCh        chan struct{}
	stoppedCh     chan struct{}
}

// NewIndexer creates a new chain indexer
func NewIndexer(
	client *ethclient.Client,
	contractAddress common.Address,
	deps IndexerDeps,
	startBlock uint64,
	confirmations uint64,
	pollInterval time.Duration,
) *Indexer {
	return &Indexer{
		client:        client,
		contract:      contractAddress,
		deps:          deps,
		lastBlock:     startBlock,
		confirmations: confirmations,
		pollInterval:  pollInterval,
		blockHashes:   make(map[uint64]common.Hash),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}
}

// Run starts the indexer loop
func (idx *Indexer) Run(ctx context.Context) error {
	defer close(idx.stoppedCh)
	
	ticker := time.NewTicker(idx.pollInterval)
	defer ticker.Stop()

	log.Printf("Starting indexer from block %d with %d confirmations", idx.lastBlock, idx.confirmations)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-idx.stopCh:
			return nil
		case <-ticker.C:
			if err := idx.processNewBlocks(ctx); err != nil {
				log.Printf("Error processing blocks: %v", err)
				// Continue on error, don't stop the indexer
			}
		}
	}
}

// Stop stops the indexer
func (idx *Indexer) Stop() {
	close(idx.stopCh)
	<-idx.stoppedCh
}

// processNewBlocks fetches and processes new blocks
func (idx *Indexer) processNewBlocks(ctx context.Context) error {
	latestBlock, err := idx.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	// Process blocks from lastBlock+1 to latestBlock
	for blockNum := idx.lastBlock + 1; blockNum <= latestBlock; blockNum++ {
		if err := idx.processBlock(ctx, blockNum); err != nil {
			return fmt.Errorf("failed to process block %d: %w", blockNum, err)
		}

		idx.mu.Lock()
		idx.lastBlock = blockNum
		idx.mu.Unlock()
	}

	// Check for confirmations and reorgs
	if err := idx.checkConfirmations(ctx); err != nil {
		return fmt.Errorf("failed to check confirmations: %w", err)
	}

	return nil
}

// processBlock fetches and processes a single block
func (idx *Indexer) processBlock(ctx context.Context, blockNum uint64) error {
	block, err := idx.client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	// Store block hash for reorg detection
	idx.mu.Lock()
	idx.blockHashes[blockNum] = block.Hash()
	idx.mu.Unlock()

	// Query logs for this block
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(blockNum)),
		ToBlock:   big.NewInt(int64(blockNum)),
		Addresses: []common.Address{idx.contract},
	}

	logs, err := idx.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to filter logs: %w", err)
	}

	// Process each log
	for _, vLog := range logs {
		if err := idx.processLog(ctx, vLog, block); err != nil {
			log.Printf("Error processing log: %v", err)
			// Continue processing other logs
		}
	}

	return nil
}

// processLog processes a single log entry
func (idx *Indexer) processLog(ctx context.Context, vLog types.Log, block *types.Block) error {
	if len(vLog.Topics) == 0 {
		return nil
	}

	eventSig := vLog.Topics[0]

	blockInfo := BlockInfo{
		Number:    vLog.BlockNumber,
		Hash:      vLog.BlockHash,
		Timestamp: block.Time(),
	}

	// Route to appropriate handler based on event signature
	switch eventSig {
	case DepositedEventSig:
		return idx.handleDeposited(ctx, vLog, blockInfo)
	case ReleasedEventSig:
		return idx.handleReleased(ctx, vLog, blockInfo)
	case RefundedEventSig:
		return idx.handleRefunded(ctx, vLog, blockInfo)
	case SplitEventSig:
		return idx.handleSplit(ctx, vLog, blockInfo)
	case ReassignedAgentEventSig:
		return idx.handleReassigned(ctx, vLog, blockInfo)
	case SubmittedEventSig:
		return idx.handleSubmitted(ctx, vLog, blockInfo)
	default:
		// Unknown event, skip
		return nil
	}
}

// handleDeposited processes a Deposited event
func (idx *Indexer) handleDeposited(ctx context.Context, vLog types.Log, blockInfo BlockInfo) error {
	if len(vLog.Topics) < 4 {
		return fmt.Errorf("invalid Deposited event: insufficient topics")
	}

	event := DepositedEvent{
		TaskID: [32]byte(vLog.Topics[1]),
		Poster: common.BytesToAddress(vLog.Topics[2].Bytes()),
		Agent:  common.BytesToAddress(vLog.Topics[3].Bytes()),
	}

	// Parse amount and bidHash from data
	if len(vLog.Data) >= 64 {
		event.Amount = new(big.Int).SetBytes(vLog.Data[0:32])
		copy(event.BidHash[:], vLog.Data[32:64])
	}

	log.Printf("Deposited event: taskID=%x poster=%s agent=%s amount=%s",
		event.TaskID, event.Poster.Hex(), event.Agent.Hex(), event.Amount.String())

	// Store in pending confirmations (not yet confirmed)
	// After 12 blocks, call idx.deps.TaskService.ConfirmDeposit()
	
	// Cross-reference with outbox
	matched, err := idx.deps.OutboxRepo.MatchEvent(ctx, "escrow.deposited", event.TaskID, vLog.TxHash)
	if err != nil {
		return fmt.Errorf("failed to match outbox event: %w", err)
	}
	if !matched {
		log.Printf("WARNING: Rogue Deposited tx detected: taskID=%x txHash=%s", event.TaskID, vLog.TxHash.Hex())
		// Alert admin
	}

	return nil
}

// handleReleased processes a Released event
func (idx *Indexer) handleReleased(ctx context.Context, vLog types.Log, blockInfo BlockInfo) error {
	if len(vLog.Topics) < 3 {
		return fmt.Errorf("invalid Released event: insufficient topics")
	}

	event := ReleasedEvent{
		TaskID: [32]byte(vLog.Topics[1]),
		Agent:  common.BytesToAddress(vLog.Topics[2].Bytes()),
	}

	if len(vLog.Data) >= 64 {
		event.AgentAmount = new(big.Int).SetBytes(vLog.Data[0:32])
		event.FeeAmount = new(big.Int).SetBytes(vLog.Data[32:64])
	}

	log.Printf("Released event: taskID=%x agent=%s agentAmount=%s fee=%s",
		event.TaskID, event.Agent.Hex(), event.AgentAmount.String(), event.FeeAmount.String())

	// After confirmations, update escrow status to "released" and task to "completed"
	return nil
}

// handleRefunded processes a Refunded event
func (idx *Indexer) handleRefunded(ctx context.Context, vLog types.Log, blockInfo BlockInfo) error {
	if len(vLog.Topics) < 3 {
		return fmt.Errorf("invalid Refunded event: insufficient topics")
	}

	event := RefundedEvent{
		TaskID: [32]byte(vLog.Topics[1]),
		Poster: common.BytesToAddress(vLog.Topics[2].Bytes()),
	}

	if len(vLog.Data) >= 32 {
		event.Amount = new(big.Int).SetBytes(vLog.Data[0:32])
	}

	log.Printf("Refunded event: taskID=%x poster=%s amount=%s",
		event.TaskID, event.Poster.Hex(), event.Amount.String())

	return nil
}

// handleSplit processes a Split event
func (idx *Indexer) handleSplit(ctx context.Context, vLog types.Log, blockInfo BlockInfo) error {
	if len(vLog.Topics) < 4 {
		return fmt.Errorf("invalid Split event: insufficient topics")
	}

	event := SplitEvent{
		TaskID: [32]byte(vLog.Topics[1]),
		Agent:  common.BytesToAddress(vLog.Topics[2].Bytes()),
		Poster: common.BytesToAddress(vLog.Topics[3].Bytes()),
	}

	if len(vLog.Data) >= 64 {
		event.AgentAmount = new(big.Int).SetBytes(vLog.Data[0:32])
		event.PosterAmount = new(big.Int).SetBytes(vLog.Data[32:64])
	}

	log.Printf("Split event: taskID=%x agent=%s agentAmt=%s poster=%s posterAmt=%s",
		event.TaskID, event.Agent.Hex(), event.AgentAmount.String(),
		event.Poster.Hex(), event.PosterAmount.String())

	return nil
}

// handleReassigned processes a ReassignedAgent event
func (idx *Indexer) handleReassigned(ctx context.Context, vLog types.Log, blockInfo BlockInfo) error {
	if len(vLog.Topics) < 4 {
		return fmt.Errorf("invalid ReassignedAgent event: insufficient topics")
	}

	event := ReassignedAgentEvent{
		TaskID:   [32]byte(vLog.Topics[1]),
		OldAgent: common.BytesToAddress(vLog.Topics[2].Bytes()),
		NewAgent: common.BytesToAddress(vLog.Topics[3].Bytes()),
	}

	log.Printf("ReassignedAgent event: taskID=%x old=%s new=%s",
		event.TaskID, event.OldAgent.Hex(), event.NewAgent.Hex())

	return nil
}

// handleSubmitted processes a Submitted event
func (idx *Indexer) handleSubmitted(ctx context.Context, vLog types.Log, blockInfo BlockInfo) error {
	if len(vLog.Topics) < 2 {
		return fmt.Errorf("invalid Submitted event: insufficient topics")
	}

	event := SubmittedEvent{
		TaskID: [32]byte(vLog.Topics[1]),
	}

	log.Printf("Submitted event: taskID=%x", event.TaskID)

	// Cross-reference with outbox
	matched, err := idx.deps.OutboxRepo.MatchEvent(ctx, "escrow.submitted", event.TaskID, vLog.TxHash)
	if err != nil {
		return fmt.Errorf("failed to match outbox event: %w", err)
	}
	if !matched {
		log.Printf("WARNING: Rogue markSubmitted tx detected: taskID=%x txHash=%s", event.TaskID, vLog.TxHash.Hex())
	}

	return nil
}

// checkConfirmations checks for confirmed blocks and detects reorgs
func (idx *Indexer) checkConfirmations(ctx context.Context) error {
	idx.mu.RLock()
	lastBlock := idx.lastBlock
	idx.mu.RUnlock()

	if lastBlock < idx.confirmations {
		return nil // Not enough blocks yet
	}

	safeBlock := lastBlock - idx.confirmations

	// Check for reorgs by comparing stored hashes with current chain
	idx.mu.RLock()
	for blockNum := idx.lastSafeBlock + 1; blockNum <= safeBlock; blockNum++ {
		storedHash, exists := idx.blockHashes[blockNum]
		idx.mu.RUnlock()

		if !exists {
			idx.mu.RLock()
			continue
		}

		// Fetch current block hash
		block, err := idx.client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", blockNum, err)
		}

		if block.Hash() != storedHash {
			// Reorg detected!
			log.Printf("REORG DETECTED at block %d: stored=%s current=%s",
				blockNum, storedHash.Hex(), block.Hash().Hex())
			return idx.handleReorg(ctx, blockNum)
		}

		idx.mu.RLock()
	}
	idx.mu.RUnlock()

	idx.mu.Lock()
	idx.lastSafeBlock = safeBlock
	idx.mu.Unlock()

	return nil
}

// handleReorg handles a blockchain reorganization
func (idx *Indexer) handleReorg(ctx context.Context, reorgBlock uint64) error {
	log.Printf("Handling reorg from block %d", reorgBlock)

	idx.mu.Lock()
	// Reset to last safe block before reorg
	idx.lastBlock = reorgBlock - 1
	
	// Clear block hashes after reorg point
	for blockNum := range idx.blockHashes {
		if blockNum >= reorgBlock {
			delete(idx.blockHashes, blockNum)
		}
	}
	idx.mu.Unlock()

	// Mark transactions as reorged in database
	// This will be implemented when transaction repo is available
	
	log.Printf("Reorg handled, resuming from block %d", reorgBlock-1)
	return nil
}

// GetLastProcessedBlock returns the last processed block number
func (idx *Indexer) GetLastProcessedBlock() uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.lastBlock
}

// GetLastSafeBlock returns the last safe (confirmed) block number
func (idx *Indexer) GetLastSafeBlock() uint64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.lastSafeBlock
}
