# Chain Integration Package

This package provides chain integration for the Agentic marketplace, including:

1. **Client** (`client.go`) - Ethereum RPC client wrapper with retry logic
2. **Relayer** (`relayer.go`) - Sends on-chain transactions for escrow operations
3. **Indexer** (`indexer.go`) - Event indexer with reorg handling and confirmations
4. **Bond Verifier** (`bond_verifier.go`) - Verifies USDC Transfer events for dispute bonds
5. **BidHash Utils** (`bidhash.go`) - TaskID and BidHash computation utilities
6. **Events** (`events.go`) - Contract event definitions

## Components

### 1. Client (`client.go`)

Ethereum RPC client wrapper with connection management and retry logic.

**Features:**
- Automatic retry with exponential backoff
- Connection health checking
- Reconnection on failure
- Configurable retry parameters

**Usage:**
```go
client, err := chain.NewClient(rpcURL, maxRetries, retryDelay)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

chainID, err := client.ChainID(ctx)
```

### 2. Relayer (`relayer.go`)

Sends on-chain transactions for: `release`, `refund`, `split`, `reassignAgent`, `markSubmitted`.

**Features:**
- Proper nonce management (sequential, synchronized)
- Automatic gas estimation with 20% buffer
- Gas price with 10% buffer for faster confirmation
- Transaction signing with EIP-155
- Nonce reset on send failure

**Usage:**
```go
relayer, err := chain.NewRelayer(client, privateKeyHex, contractAddress, chainID)
if err != nil {
    log.Fatal(err)
}

// Send release transaction
txHash, err := relayer.Release(ctx, taskIDBytes32)
if err != nil {
    log.Printf("Failed to send release: %v", err)
}
```

**Methods:**
- `Release(ctx, taskID)` - Release escrow to agent (5% fee)
- `Refund(ctx, taskID)` - Refund escrow to poster (0% fee)
- `Split(ctx, taskID, agentBps)` - Split escrow between agent and poster (0% fee)
- `ReassignAgent(ctx, taskID, newAgent)` - Reassign task to new agent (only if !submitted)
- `MarkSubmitted(ctx, taskID)` - Mark task as submitted on-chain

### 3. Indexer (`indexer.go`)

Polls for new blocks every 2s, parses contract events, handles confirmations and reorgs.

**Features:**
- Polls for new blocks at configurable interval (default: 2s)
- Parses all contract events: `Deposited`, `Released`, `Refunded`, `Split`, `ReassignedAgent`, `Submitted`
- Stores (block_number, block_hash) per event for reorg detection
- 12-block confirmation requirement before acting on events
- Reorg detection via block hash comparison
- On reorg: marks affected transactions as reorged, replays from last safe block
- Cross-references events against outbox for rogue transaction detection

**Usage:**
```go
deps := chain.IndexerDeps{
    EscrowRepo:  escrowRepo,
    TxRepo:      txRepo,
    OutboxRepo:  outboxRepo,
    TaskService: taskService,
}

indexer := chain.NewIndexer(
    client.Client(),
    contractAddress,
    deps,
    startBlock,
    12, // confirmations
    2*time.Second, // poll interval
)

go indexer.Run(ctx)
```

**Event Handling:**
- `Deposited` → Confirms deposit after 12 blocks, updates task to "assigned"
- `Released` → Updates escrow to "released", task to "completed"
- `Refunded` → Updates escrow to "refunded", task to "refunded"
- `Split` → Updates escrow to "split", task to "split"
- `ReassignedAgent` → Updates assigned agent
- `Submitted` → Records submission event

**Reorg Handling:**
- Stores block hashes for all processed blocks
- After 12 confirmations, compares stored hash with current chain
- On mismatch: rollback to last safe block, mark affected transactions as reorged
- Replays blocks from reorg point

### 4. Bond Verifier (`bond_verifier.go`)

Verifies USDC Transfer events for dispute bonds.

**Features:**
- Checks Transfer event in transaction receipt
- Verifies `to == bondOpsWallet address`
- Verifies `amount == floor(escrow_amount_raw * 100 / 10000)` (1% bond)
- Requires >= 12 block confirmations
- Bond amount computation utility

**Usage:**
```go
verifier := chain.NewBondVerifier(
    client.Client(),
    usdcContract,
    bondOpsWallet,
    12, // confirmations
)

// Verify bond transaction
err := verifier.VerifyBondWithEscrow(ctx, txHash, escrowAmount)
if err != nil {
    log.Printf("Invalid bond: %v", err)
}

// Or compute expected bond amount
bondAmount := chain.ComputeBondAmount(escrowAmount)
```

**Bond Formula:**
```
bond = floor(escrow_amount_raw × 100 / 10000)
     = floor(escrow_amount_raw / 100)
     = 1% of escrow amount
```

### 5. BidHash Utils (`bidhash.go`)

TaskID and BidHash computation per v0.1.5 spec.

**TaskID Derivation:**
```go
// UUID (16 bytes RFC 4122 binary) → keccak256 → bytes32
taskID := chain.TaskIDFromUUID(taskUUID)
```

**BidHash Computation:**
```go
bidHash := chain.ComputeBidHash(
    chainID,          // uint64
    contractAddr,     // address
    taskID,           // bytes32
    agent,            // address
    amount,           // *big.Int (uint96)
    etaHours,         // uint32
    createdAt,        // uint64 (unix timestamp)
    bidID,            // bytes32
)
```

**Formula:**
```
bidHash = keccak256(abi.encode(
    "AI_AGENT_MARKETPLACE_V0_1",  // domain separator
    chainId,
    escrowContractAddress,
    taskId,
    agent,
    amount,
    etaHours,
    createdAt,
    bidId
))
```

### 6. Events (`events.go`)

Contract event definitions with Go structs.

**Events:**
- `DepositedEvent` - Escrow deposit confirmed
- `ReleasedEvent` - Escrow released to agent
- `RefundedEvent` - Escrow refunded to poster
- `SplitEvent` - Escrow split between parties
- `ReassignedAgentEvent` - Agent reassigned
- `SubmittedEvent` - Task marked as submitted

**Block Info:**
- `BlockInfo` - Stores block metadata (number, hash, timestamp) for reorg detection

## Dependencies

```go
require (
    github.com/ethereum/go-ethereum v1.14.12
    github.com/google/uuid v1.6.0
    golang.org/x/crypto v0.48.0
)
```

## Integration with Other Components

### With Repository Layer (Agent 2):
- `EscrowRepository` - Update escrow status
- `TransactionRepository` - Store transaction records
- `OutboxRepository` - Cross-reference events for rogue tx detection

### With Service Layer (Agent 3):
- `TaskService.ConfirmDeposit()` - Called by indexer after 12 confirmations
- `TaskService.HandleEscrowSettled()` - Called on release/refund/split events

### With Contract (Agent 1):
- Event signatures must match contract ABI
- Function selectors computed from contract interface
- When Agent 1 provides ABI, update event signatures in `events.go`

## Configuration

Required environment variables:
- `BASE_RPC_URL` - Base L2 RPC endpoint
- `ESCROW_CONTRACT` - Deployed contract address
- `RELAYER_PRIVATE_KEY` - Relayer hot wallet private key (hex)
- `USDC_CONTRACT` - Base USDC contract (0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913)
- `BOND_OPS_WALLET` - Hot wallet for dispute bonds
- `CHAIN_ID` - 8453 for Base mainnet

## Testing

To test compilation (once Go is installed):
```bash
cd ~/workspace/agentic
go build ./internal/chain/...
```

## TODO / Notes

1. **Event Signatures** - Update when Agent 1 provides contract ABI
2. **Repository Integration** - Interfaces defined, implementations from Agent 2
3. **Service Integration** - TaskService methods called by indexer
4. **Monitoring** - Add metrics for:
   - Events processed per minute
   - Reorg detection frequency
   - Relayer nonce gaps
   - Bond verification failures
   - Rogue transaction alerts

## Security Notes

1. **Relayer Private Key** - Store securely, never commit to git
2. **Nonce Management** - Critical for transaction ordering, uses mutex
3. **Gas Buffer** - 20% gas limit + 10% gas price to avoid out-of-gas
4. **Reorg Handling** - 12-block confirmations prevent acting on unconfirmed events
5. **Rogue TX Detection** - Cross-referencing with outbox detects unauthorized relayer usage
