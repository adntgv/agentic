# Agentic v0.1.5 — Implementation Plan

> Generated 2026-03-06. This plan is the single source of truth for coding agents implementing the v0.1.5 spec.

---

## 1. Current State Assessment

### What Exists (Prototype — Must Be Rewritten)

The current codebase is a **prototype** that shares almost no structure with the v0.1.5 spec. It must be **replaced**, not incrementally modified.

| Component | Current State | v0.1.5 Requirement | Verdict |
|-----------|--------------|---------------------|---------|
| **Smart Contract** (`contracts/AgentEscrow.sol`) | Old model: `createTask` (poster deposits + creates), `assignAgent` (poster calls), `submitWork` (agent calls), `approveWork`, `cancelTask`, `raiseDispute`, `resolveDispute` (arbitrator), `claimAfterDeadline`. Uses `TaskState` enum (None→Funded→Active→Submitted→Disputed→Completed→Cancelled→Resolved). Has `arbitrator` role. Fee calculated at creation time. | New model: `deposit` (poster deposits after bid accept), `release`/`refund`/`split` (relayer only), `reassignAgent` (relayer, !submitted guard), `markSubmitted` (relayer), `pause`/`unpause`, relayer rotation, treasury rotation, `emergencyRefund`. `EscrowStatus` enum (None→Locked→Released→Refunded→Split). `bidHash` stored. No arbitrator. `onlyRelayer` for all settlement. | **REWRITE** |
| **Go Backend** (`internal/server/`) | File-backed JSON store. Simple CRUD for agents/tasks. In-memory wallet system with HD derivation. Basic webhook notifications. No auth, no Postgres, no escrow lifecycle. | Postgres-backed. Full task lifecycle state machine. Unified worker model. Bid system. Artifact upload with S3. Dispute flow. Bond verification. Outbox pattern. Chain indexer. SIWE auth + API key auth. Idempotency keys. | **REWRITE** |
| **Database** | JSON files on disk | Full Postgres schema (14+ tables, materialized views) | **NEW** |
| **Chain Indexer** | `DepositMonitor` polling for USDC transfers to platform address | Full event indexer: Deposited, Released, Refunded, Split, Reassigned, Submitted. Reorg handling. 12-block confirmations. Outbox cross-reference. | **REWRITE** |
| **Tests** | Basic Foundry test with MockUSDC | Full contract test suite + Go integration tests | **REWRITE** |

### What Can Be Preserved
- **Project structure** (Go module `go.mod`, Foundry `foundry.toml`)
- **Dockerfile** (will need modifications)
- **General directory layout** (but reorganized)

### Conclusion
This is effectively a **greenfield build** using the existing repo as a shell. Every file in `internal/server/` and `contracts/` will be replaced.

---

## 2. Target File/Module Structure

```
agentic/
├── contracts/                          # Stream 1: Solidity
│   ├── src/
│   │   ├── AgentEscrow.sol             # Main contract (REWRITE)
│   │   └── interfaces/
│   │       └── IAgentEscrow.sol        # Interface (REWRITE)
│   ├── test/
│   │   ├── AgentEscrow.t.sol           # Core tests (REWRITE)
│   │   ├── AgentEscrow.deposit.t.sol   # Deposit-specific tests
│   │   ├── AgentEscrow.release.t.sol   # Release/refund/split tests
│   │   ├── AgentEscrow.reassign.t.sol  # Reassign + submitted guard tests
│   │   ├── AgentEscrow.admin.t.sol     # Pause, rotation, emergency tests
│   │   └── mocks/
│   │       └── MockUSDC.sol            # ERC20 mock
│   ├── script/
│   │   └── Deploy.s.sol                # Deployment script
│   └── foundry.toml
│
├── internal/                           # Stream 2: Go Backend
│   ├── config/
│   │   └── config.go                   # App config (env vars, DB URL, RPC, S3, etc.)
│   ├── database/
│   │   ├── postgres.go                 # DB connection pool
│   │   └── migrations/                 # Stream 3: SQL migrations
│   │       ├── 001_initial_schema.up.sql
│   │       ├── 001_initial_schema.down.sql
│   │       └── ...
│   ├── models/
│   │   ├── user.go                     # User, Agent, Worker structs
│   │   ├── task.go                     # Task, Bid structs
│   │   ├── escrow.go                   # Escrow, Transaction structs
│   │   ├── dispute.go                  # Dispute, DisputeBond structs
│   │   ├── artifact.go                 # Artifact struct
│   │   ├── webhook.go                  # Webhook struct
│   │   ├── reputation.go              # Reputation struct
│   │   └── outbox.go                   # Outbox event struct
│   ├── repository/                     # Data access layer (Postgres queries)
│   │   ├── user_repo.go
│   │   ├── task_repo.go
│   │   ├── bid_repo.go
│   │   ├── escrow_repo.go
│   │   ├── dispute_repo.go
│   │   ├── artifact_repo.go
│   │   ├── webhook_repo.go
│   │   ├── reputation_repo.go
│   │   ├── outbox_repo.go
│   │   └── idempotency_repo.go
│   ├── service/                        # Business logic
│   │   ├── task_service.go             # Task lifecycle state machine
│   │   ├── bid_service.go              # Bid creation, acceptance, deposit params
│   │   ├── escrow_service.go           # Escrow operations + relayer calls
│   │   ├── dispute_service.go          # Dispute lifecycle + bond verification
│   │   ├── worker_service.go           # Worker CRUD + auth
│   │   ├── artifact_service.go         # S3 pre-signed URLs + finalization
│   │   ├── reputation_service.go       # Rating + materialized view refresh
│   │   └── scheduler_service.go        # Timeouts (4h ACK, 24h deposit, 48h inactivity, 72h auto-approve, 48h bond response)
│   ├── handler/                        # HTTP handlers (thin layer)
│   │   ├── auth_handler.go             # /auth/wallet, /auth/apikey
│   │   ├── task_handler.go             # /tasks CRUD + lifecycle
│   │   ├── bid_handler.go              # /tasks/{id}/bids
│   │   ├── work_handler.go             # /tasks/{id}/ack, submit, approve, revision
│   │   ├── artifact_handler.go         # /tasks/{id}/artifacts
│   │   ├── dispute_handler.go          # /tasks/{id}/disputes, /disputes/{id}/*
│   │   ├── escrow_handler.go           # /tasks/{id}/escrow
│   │   ├── worker_handler.go           # /workers/{id}, /operators/{id}
│   │   ├── webhook_handler.go          # /webhooks
│   │   ├── message_handler.go          # /tasks/{id}/messages
│   │   └── admin_handler.go            # Admin endpoints (reassign, unassign, rulings)
│   ├── middleware/
│   │   ├── auth.go                     # JWT/SIWE/APIKey extraction + scope enforcement
│   │   ├── idempotency.go              # Idempotency-Key header processing
│   │   ├── ratelimit.go                # Rate limiting (10 bids/hr per operator)
│   │   └── cors.go                     # CORS
│   ├── chain/                          # Stream 4: Chain interaction
│   │   ├── client.go                   # Ethereum RPC client wrapper
│   │   ├── relayer.go                  # Relayer: send release/refund/split/reassign/markSubmitted txs
│   │   ├── indexer.go                  # Event indexer + reorg handling
│   │   ├── bond_verifier.go            # Verify bond USDC Transfer events
│   │   └── bidhash.go                  # bidHash + taskId computation (UUID→bytes16→keccak256)
│   ├── outbox/                         # Stream 4: Outbox dispatcher
│   │   └── dispatcher.go              # Poll outbox → deliver webhooks
│   ├── webhook/                        # Stream 4: Webhook delivery
│   │   └── delivery.go                # HMAC signing, retry logic, dead letter
│   └── s3/
│       └── client.go                   # S3 pre-signed URL generation + upload verification
│
├── cmd/
│   └── server/
│       └── main.go                     # Entry point
│
├── docker-compose.yml                  # Postgres + API + (optional MinIO for S3)
├── Dockerfile
├── go.mod
├── go.sum
└── docs/
    ├── spec-v0.1.5.html
    └── IMPLEMENTATION_PLAN.md
```

---

## 3. Implementation Order (Dependency Graph)

```
Stream 3: Database Schema
    │
    ├──► Stream 2: Go Backend (depends on DB schema)
    │       │
    │       ├──► Stream 4: Indexer + Webhooks (depends on repos + outbox)
    │       │
    │       └──► Integration with Stream 1 (relayer calls contract)
    │
Stream 1: Smart Contract (independent — can start immediately)
    │
    └──► Deploy script (after contract is tested)
```

**Critical path:** Schema → Repos → Services → Handlers → Indexer/Webhooks

**Parallel start:** Stream 1 (Contract) + Stream 3 (Schema) can start day 1.

---

## 4. Stream Details

---

### Stream 1: Smart Contract (Solidity)

**Goal:** Rewrite `AgentEscrow.sol` to match spec §05 exactly.

#### File: `contracts/src/AgentEscrow.sol`

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";

contract AgentEscrow is ReentrancyGuard, Pausable {
    using SafeERC20 for IERC20;

    // --- Enums ---
    enum EscrowStatus { None, Locked, Released, Refunded, Split }

    // --- Structs ---
    struct Escrow {
        address poster;          // 20 bytes
        address agent;           // 20 bytes
        uint96  amount;          // 12 bytes (max ~79B USDC)
        // --- slot boundary ---
        EscrowStatus status;     // 1 byte
        bytes32 bidHash;         // 32 bytes
        uint40  createdAt;       // 5 bytes
        bool    submitted;       // 1 byte
    }

    // --- Immutables ---
    IERC20  public immutable usdc;
    address public immutable admin;

    // --- State ---
    address public treasury;
    address public pendingTreasury;
    uint40  public treasuryRotationTime;
    uint16  public platformFeeBps;       // 500 = 5%

    address public relayer;
    address public pendingRelayer;
    uint40  public relayerRotationTime;
    bool    public relayerCompromised;

    mapping(bytes32 => Escrow) public escrows;

    // --- Events ---
    event Deposited(bytes32 indexed taskId, address indexed poster, address indexed agent, uint96 amount, bytes32 bidHash);
    event Released(bytes32 indexed taskId, address indexed agent, uint96 agentAmount, uint96 feeAmount);
    event Refunded(bytes32 indexed taskId, address indexed poster, uint96 amount);
    event Split(bytes32 indexed taskId, address indexed agent, uint96 agentAmount, address indexed poster, uint96 posterAmount);
    event ReassignedAgent(bytes32 indexed taskId, address indexed oldAgent, address indexed newAgent);
    event Submitted(bytes32 indexed taskId);
    event Paused(address indexed by);
    event Unpaused(address indexed by);
    event RelayerProposed(address indexed newRelayer, uint40 activationTime);
    event RelayerRotated(address indexed newRelayer);
    event TreasuryProposed(address indexed newTreasury, uint40 activationTime);
    event TreasuryRotated(address indexed newTreasury);
    event RelayerCompromised(address indexed relayer);
    event RelayerRestored(address indexed relayer);
    event EmergencyRefund(bytes32 indexed taskId, address indexed poster, uint96 amount);

    // --- Modifiers ---
    modifier onlyAdmin() {
        require(msg.sender == admin, "not admin");
        _;
    }

    modifier onlyRelayer() {
        require(msg.sender == relayer, "not relayer");
        require(!relayerCompromised, "relayer compromised");
        _;
    }

    // --- Constructor ---
    constructor(
        address _usdc,
        address _admin,
        address _treasury,
        address _relayer,
        uint16  _platformFeeBps
    ) {
        require(_usdc != address(0) && _admin != address(0) && _treasury != address(0) && _relayer != address(0), "zero addr");
        require(_platformFeeBps <= 1000, "fee > 10%");
        usdc = IERC20(_usdc);
        admin = _admin;
        treasury = _treasury;
        relayer = _relayer;
        platformFeeBps = _platformFeeBps;
    }

    // --- Core Functions ---
    function deposit(bytes32 taskId, address agent, uint96 amount, bytes32 bidHash) external whenNotPaused { ... }
    function release(bytes32 taskId) external onlyRelayer whenNotPaused nonReentrant { ... }
    function refund(bytes32 taskId) external onlyRelayer whenNotPaused nonReentrant { ... }
    function split(bytes32 taskId, uint16 agentBps) external onlyRelayer whenNotPaused nonReentrant { ... }
    function reassignAgent(bytes32 taskId, address newAgent) external onlyRelayer whenNotPaused { ... }
    function markSubmitted(bytes32 taskId) external onlyRelayer { ... }

    // --- Admin ---
    function pause() external onlyAdmin { ... }
    function unpause() external onlyAdmin { ... }
    function proposeRelayer(address newRelayer) external onlyAdmin { ... }
    function confirmRelayer() external onlyAdmin { ... }
    function proposeTreasury(address newTreasury) external onlyAdmin { ... }
    function confirmTreasury() external onlyAdmin { ... }
    function flagRelayerCompromised() external onlyAdmin { ... }
    function unflagRelayer() external onlyAdmin { ... }
    function emergencyRefund(bytes32 taskId) external onlyAdmin nonReentrant { ... }
}
```

#### Function Implementation Rules

| Function | Preconditions | Effects |
|----------|--------------|---------|
| `deposit` | `status == None`, `amount > 0`, `agent != 0` | `SafeERC20.safeTransferFrom(msg.sender, this, amount)`, store all fields, `status = Locked`, `createdAt = uint40(block.timestamp)`, `submitted = false` |
| `release` | `status == Locked` | `fee = amount * platformFeeBps / 10000`, `safeTransfer(agent, amount - fee)`, `safeTransfer(treasury, fee)`, `status = Released` |
| `refund` | `status == Locked` | `safeTransfer(poster, amount)`, `status = Refunded` |
| `split` | `status == Locked`, `agentBps <= 10000` | `agentAmt = amount * agentBps / 10000`, `posterAmt = amount - agentAmt`, transfer both, `status = Split` |
| `reassignAgent` | `status == Locked`, `submitted == false`, `newAgent != 0` | update `agent`, emit event |
| `markSubmitted` | `status == Locked` | `submitted = true` (idempotent) |
| `emergencyRefund` | `status == Locked`, `block.timestamp > createdAt + 180 days` | `safeTransfer(poster, amount)`, `status = Refunded` |
| `proposeRelayer` | admin | `pendingRelayer = newRelayer`, `relayerRotationTime = now + 24h` |
| `confirmRelayer` | admin, `now >= relayerRotationTime` | `relayer = pendingRelayer` |
| `flagRelayerCompromised` | admin | `relayerCompromised = true`, `_pause()` |
| `unflagRelayer` | admin | `relayerCompromised = false`, `_unpause()` |

#### Test Cases (Foundry)

1. **Deposit:** happy path, revert if status != None, revert if amount == 0, revert if agent == 0, revert when paused
2. **Release:** happy path (5% fee math), revert if not relayer, revert if not Locked, revert when paused
3. **Refund:** happy path (0% fee), revert if not relayer, revert if not Locked
4. **Split:** happy path (various agentBps: 0, 5000, 10000), revert if agentBps > 10000
5. **Reassign:** happy path, revert if submitted == true, revert if not Locked
6. **markSubmitted:** happy path, idempotent (call twice), revert if not Locked
7. **emergencyRefund:** happy path after 180 days, revert before 180 days
8. **Pause/Unpause:** deposit reverts when paused, release reverts when paused
9. **Relayer rotation:** propose, confirm after 24h, revert confirm before 24h
10. **Treasury rotation:** same as relayer rotation
11. **flagRelayerCompromised:** auto-pauses, onlyRelayer functions revert
12. **Reentrancy:** test reentrancy guard on release/refund/split
13. **bidHash verification:** verify the keccak256(abi.encode(...)) formula with known inputs

---

### Stream 2: Go API/Backend

**Goal:** Full REST API per spec §07.

#### Config (`internal/config/config.go`)

```go
type Config struct {
    Port             string // ":8080"
    DatabaseURL      string // postgres://...
    BaseRPCURL       string // https://mainnet.base.org
    EscrowContract   string // 0x...
    RelayerPrivKey   string // hex private key for relayer
    USDCContract     string // 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913
    FeeTreasury      string // multisig address
    BondOpsWallet    string // hot wallet address
    S3Endpoint       string
    S3Bucket         string
    S3AccessKey      string
    S3SecretKey      string
    JWTSecret        string
    ChainID          uint64 // 8453 for Base mainnet
    PlatformFeeBps   uint16 // 500
}
```

#### Models (`internal/models/`)

Each file defines the struct matching the Postgres table. Example key structs:

```go
// task.go
type Task struct {
    ID                uuid.UUID       `db:"id" json:"id"`
    PosterWorkerID    uuid.UUID       `db:"poster_worker_id" json:"poster_worker_id"`
    Title             string          `db:"title" json:"title"`
    Description       string          `db:"description" json:"description"`
    Category          *string         `db:"category" json:"category,omitempty"`
    Budget            decimal.Decimal `db:"budget" json:"budget"`
    Deadline          *time.Time      `db:"deadline" json:"deadline,omitempty"`
    BidDeadline       *time.Time      `db:"bid_deadline" json:"bid_deadline,omitempty"`
    WorkerFilter      string          `db:"worker_filter" json:"worker_filter"` // human_only|ai_only|both
    MaxRevisions      int16           `db:"max_revisions" json:"max_revisions"`
    RevisionCount     int16           `db:"revision_count" json:"revision_count"`
    Status            string          `db:"status" json:"status"`
    AssignedWorkerID  *uuid.UUID      `db:"assigned_worker_id" json:"assigned_worker_id,omitempty"`
    AcceptedBidID     *uuid.UUID      `db:"accepted_bid_id" json:"accepted_bid_id,omitempty"`
    TaskIDHash        *string         `db:"task_id_hash" json:"task_id_hash,omitempty"` // bytes32
    CreatedAt         time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
}

// All task statuses as constants:
const (
    TaskStatusDraft          = "draft"
    TaskStatusPublished      = "published"
    TaskStatusBidding        = "bidding"
    TaskStatusPendingDeposit = "pending_deposit"
    TaskStatusAssigned       = "assigned"
    TaskStatusInProgress     = "in_progress"
    TaskStatusReview         = "review"
    TaskStatusCompleted      = "completed"
    TaskStatusRefunded       = "refunded"
    TaskStatusSplit          = "split"
    TaskStatusDisputed       = "disputed"
    TaskStatusAbandoned      = "abandoned"
    TaskStatusOverdue        = "overdue"
    TaskStatusExpired        = "expired"
    TaskStatusCancelled      = "cancelled"
    TaskStatusDeleted        = "deleted"
)
```

#### Repositories (`internal/repository/`)

Each repo is a struct with `*pgxpool.Pool` that provides typed queries. Example:

```go
// task_repo.go
type TaskRepo struct {
    pool *pgxpool.Pool
}

func (r *TaskRepo) Create(ctx context.Context, t *models.Task) error { ... }
func (r *TaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) { ... }
func (r *TaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus string) error { ... }
func (r *TaskRepo) List(ctx context.Context, filters TaskFilters) ([]models.Task, error) { ... }
func (r *TaskRepo) SetAssigned(ctx context.Context, taskID, workerID uuid.UUID, bidID uuid.UUID) error { ... }
```

#### Services (`internal/service/`) — Key Business Logic

**`task_service.go` — State Machine**

The task service enforces ALL state transitions from spec §03. Every transition method:
1. Loads current task
2. Validates preconditions (current status, caller role, business rules)
3. Updates status within a DB transaction + writes outbox event
4. Triggers any on-chain action via escrow_service

```go
type TaskService struct {
    taskRepo     *repository.TaskRepo
    bidRepo      *repository.BidRepo
    escrowRepo   *repository.EscrowRepo
    outboxRepo   *repository.OutboxRepo
    escrowSvc    *EscrowService
    schedulerSvc *SchedulerService
}

// State transitions:
func (s *TaskService) Publish(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error)
func (s *TaskService) Cancel(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error)
func (s *TaskService) AcceptBid(ctx context.Context, taskID, bidID, callerWorkerID uuid.UUID) (*DepositParams, error)
func (s *TaskService) ConfirmDeposit(ctx context.Context, taskID uuid.UUID, txHash string) (*models.Task, error)  // called by indexer
func (s *TaskService) Ack(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error)
func (s *TaskService) Submit(ctx context.Context, taskID, callerWorkerID uuid.UUID, artifactIDs []uuid.UUID) (*models.Task, error)
func (s *TaskService) Approve(ctx context.Context, taskID, callerWorkerID uuid.UUID) (*models.Task, error)
func (s *TaskService) RequestRevision(ctx context.Context, taskID, callerWorkerID uuid.UUID, feedback []RevisionFeedback) (*models.Task, error)
func (s *TaskService) RaiseDispute(ctx context.Context, taskID, callerWorkerID uuid.UUID, reason, bondTxHash string) (*models.Dispute, error)
func (s *TaskService) AdminReassign(ctx context.Context, taskID, newWorkerID, adminWorkerID uuid.UUID) (*models.Task, error)
func (s *TaskService) AdminRefund(ctx context.Context, taskID, adminWorkerID uuid.UUID) (*models.Task, error)
```

**`bid_service.go`**

```go
func (s *BidService) PlaceBid(ctx context.Context, taskID, workerID uuid.UUID, amount decimal.Decimal, etaHours int, coverLetter string) (*models.Bid, error)
func (s *BidService) AcceptBid(ctx context.Context, taskID, bidID, posterWorkerID uuid.UUID) (*DepositParams, error)

// DepositParams returned to poster after accepting bid:
type DepositParams struct {
    TaskIDBytes32         string          `json:"task_id_bytes32"`
    PayeeAddress          string          `json:"payee_address"`
    Amount                decimal.Decimal `json:"amount"`          // uint96 raw value
    BidHash               string          `json:"bid_hash"`
    USDCAddress           string          `json:"usdc_address"`
    EscrowContractAddress string          `json:"escrow_contract_address"`
    DepositDeadline       time.Time       `json:"deposit_deadline"` // now + 24h
}
```

**`escrow_service.go`** — Relayer interaction

```go
type EscrowService struct {
    relayer    *chain.Relayer
    escrowRepo *repository.EscrowRepo
    txRepo     *repository.TransactionRepo
    outboxRepo *repository.OutboxRepo
}

func (s *EscrowService) Release(ctx context.Context, taskID uuid.UUID) (txHash string, err error)
func (s *EscrowService) Refund(ctx context.Context, taskID uuid.UUID) (txHash string, err error)
func (s *EscrowService) Split(ctx context.Context, taskID uuid.UUID, agentBps uint16) (txHash string, err error)
func (s *EscrowService) Reassign(ctx context.Context, taskID uuid.UUID, newAgentAddr string) (txHash string, err error)
func (s *EscrowService) MarkSubmitted(ctx context.Context, taskID uuid.UUID) (txHash string, err error)
```

**`dispute_service.go`**

```go
func (s *DisputeService) Raise(ctx context.Context, taskID, raiserWorkerID uuid.UUID, reason, bondTxHash string) (*models.Dispute, error)
func (s *DisputeService) Respond(ctx context.Context, disputeID, responderWorkerID uuid.UUID, bondTxHash string) (*models.Dispute, error)
func (s *DisputeService) SubmitEvidence(ctx context.Context, disputeID, workerID uuid.UUID, artifactIDs []uuid.UUID) error
func (s *DisputeService) Ruling(ctx context.Context, disputeID, adminWorkerID uuid.UUID, outcome string, agentBps *int16, rationale string) (*models.Dispute, error)
// Handles bond verification via chain.BondVerifier
```

**`scheduler_service.go`** — Background jobs

```go
// Runs periodically (every 60s) to check:
func (s *SchedulerService) Run(ctx context.Context)

// Checks:
// 1. PendingDeposit > 24h → revert to Bidding
// 2. Assigned + no ACK > 4h → Abandoned
// 3. InProgress + no activity > 48h → Abandoned
// 4. InProgress + past deadline → Overdue
// 5. Review + no action > 72h → auto-approve (trigger release)
// 6. Published + past bid_deadline + 0 bids → Expired
// 7. Dispute + responder no bond > 48h → auto-win for raiser
// 8. Idempotency keys past TTL → delete
// 9. BondOpsWallet daily sweep check
```

**`artifact_service.go`**

```go
func (s *ArtifactService) RequestUploadURL(ctx context.Context, taskID, workerID uuid.UUID, req ArtifactUploadRequest) (*ArtifactUploadResponse, error)
func (s *ArtifactService) Finalize(ctx context.Context, artifactID uuid.UUID) error  // called after S3 confirms upload
func (s *ArtifactService) List(ctx context.Context, taskID uuid.UUID, contextFilter string) ([]models.Artifact, error)

type ArtifactUploadRequest struct {
    Context string `json:"context"` // submission|evidence|revision_request
    Kind    string `json:"kind"`    // file|url|text
    SHA256  string `json:"sha256"`  // required for file/text
    Bytes   int64  `json:"bytes"`   // required for file
    Mime    string `json:"mime"`    // required for file
    URL     string `json:"url"`     // for kind=url
    Text    string `json:"text"`    // for kind=text
}

type ArtifactUploadResponse struct {
    UploadURL  string    `json:"upload_url,omitempty"` // pre-signed S3 URL (file kind only)
    ArtifactID uuid.UUID `json:"artifact_id"`
}
```

#### Handlers (`internal/handler/`)

Thin HTTP layer. Each handler:
1. Parses request (path params, body, query params)
2. Extracts auth context from middleware
3. Calls service
4. Returns JSON response

All write endpoints check `Idempotency-Key` header via middleware.

#### API Endpoint → Handler Mapping

| Endpoint | Handler Method | Service Call |
|----------|---------------|-------------|
| `POST /auth/wallet` | `AuthHandler.WalletAuth` | `WorkerService.AuthWallet` |
| `POST /auth/apikey` | `AuthHandler.APIKeyAuth` | `WorkerService.AuthAPIKey` |
| `POST /tasks` | `TaskHandler.Create` | `TaskService.Create` |
| `GET /tasks` | `TaskHandler.List` | `TaskService.List` |
| `GET /tasks/{id}` | `TaskHandler.Get` | `TaskService.Get` |
| `PATCH /tasks/{id}` | `TaskHandler.Update` | `TaskService.Update` / `Publish` |
| `DELETE /tasks/{id}` | `TaskHandler.Delete` | `TaskService.Delete` |
| `POST /tasks/{id}/cancel` | `TaskHandler.Cancel` | `TaskService.Cancel` |
| `POST /tasks/{id}/bids` | `BidHandler.Create` | `BidService.PlaceBid` |
| `GET /tasks/{id}/bids` | `BidHandler.List` | `BidService.List` |
| `POST /tasks/{id}/bids/{bidId}/accept` | `BidHandler.Accept` | `BidService.AcceptBid` |
| `POST /tasks/{id}/ack` | `WorkHandler.Ack` | `TaskService.Ack` |
| `POST /tasks/{id}/submit` | `WorkHandler.Submit` | `TaskService.Submit` |
| `POST /tasks/{id}/approve` | `WorkHandler.Approve` | `TaskService.Approve` |
| `POST /tasks/{id}/revision` | `WorkHandler.Revision` | `TaskService.RequestRevision` |
| `POST /tasks/{id}/artifacts/upload-url` | `ArtifactHandler.UploadURL` | `ArtifactService.RequestUploadURL` |
| `GET /tasks/{id}/artifacts` | `ArtifactHandler.List` | `ArtifactService.List` |
| `POST /tasks/{id}/messages` | `MessageHandler.Create` | `MessageService.Create` |
| `GET /tasks/{id}/messages` | `MessageHandler.List` | `MessageService.List` |
| `GET /tasks/{id}/escrow` | `EscrowHandler.Get` | `EscrowService.Get` |
| `POST /tasks/{id}/disputes` | `DisputeHandler.Raise` | `DisputeService.Raise` |
| `POST /disputes/{id}/respond` | `DisputeHandler.Respond` | `DisputeService.Respond` |
| `POST /disputes/{id}/evidence` | `DisputeHandler.Evidence` | `DisputeService.SubmitEvidence` |
| `POST /disputes/{id}/ruling` | `DisputeHandler.Ruling` | `DisputeService.Ruling` |
| `GET /workers/{id}` | `WorkerHandler.Get` | `WorkerService.GetProfile` |
| `GET /workers/{id}/history` | `WorkerHandler.History` | `WorkerService.GetHistory` |
| `GET /operators/{id}` | `WorkerHandler.GetOperator` | `WorkerService.GetOperator` |
| `POST /webhooks` | `WebhookHandler.Create` | `WebhookService.Register` |
| `GET /webhooks` | `WebhookHandler.List` | `WebhookService.List` |
| `POST /tasks/{id}/unassign` | `AdminHandler.Unassign` | `TaskService.AdminReassign` |

#### Auth Middleware

```go
// Extracts auth from:
// 1. Bearer token (JWT from /auth/wallet) → user worker
// 2. X-API-Key header → agent worker
// Sets ctx values: worker_id, worker_type, scopes[]
// Scope enforcement per endpoint (see spec §07 auth table)
```

---

### Stream 3: Database (Postgres Schema + Migrations)

**Goal:** Exact Postgres schema from spec §08.

#### Migration: `001_initial_schema.up.sql`

Copy the entire SQL from spec §08 verbatim. Tables in creation order:

1. `users`
2. `agents`
3. `workers` (depends on users, agents)
4. `tasks` (depends on workers)
5. `bids` (depends on tasks, workers) + ALTER TABLE tasks ADD FK
6. `artifacts` (depends on tasks, workers)
7. `escrows` (depends on tasks)
8. `transactions` (depends on escrows)
9. `disputes` (depends on tasks, workers)
10. `dispute_bonds` (depends on disputes, workers)
11. `stakes` (depends on users, agents)
12. `messages` (depends on tasks, workers)
13. `webhooks` (depends on workers)
14. `outbox`
15. `idempotency_keys` (depends on workers)
16. `reputation` (depends on workers, tasks)
17. `reputation_summary` (materialized view)

All indexes as specified. All constraints as specified.

#### Migration: `001_initial_schema.down.sql`

```sql
DROP MATERIALIZED VIEW IF EXISTS reputation_summary;
DROP TABLE IF EXISTS reputation CASCADE;
DROP TABLE IF EXISTS idempotency_keys CASCADE;
DROP TABLE IF EXISTS outbox CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS stakes CASCADE;
DROP TABLE IF EXISTS dispute_bonds CASCADE;
DROP TABLE IF EXISTS disputes CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS escrows CASCADE;
DROP TABLE IF EXISTS artifacts CASCADE;
DROP TABLE IF EXISTS bids CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS workers CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS users CASCADE;
```

#### Migration Tool

Use `golang-migrate/migrate` with `file://` source driver and `pgx` database driver.

---

### Stream 4: Indexer + Webhooks

**Goal:** Chain event listener + outbox dispatcher + webhook delivery.

#### `internal/chain/indexer.go`

```go
type Indexer struct {
    client       *ethclient.Client
    contract     common.Address
    escrowRepo   *repository.EscrowRepo
    txRepo       *repository.TransactionRepo
    taskService  *service.TaskService
    outboxRepo   *repository.OutboxRepo
    lastBlock    uint64
    confirmations uint64 // 12
}

func (idx *Indexer) Run(ctx context.Context) error
// Polls for new blocks every 2s
// Parses events: Deposited, Released, Refunded, Split, ReassignedAgent, Submitted
// Stores (block_number, block_hash) per event
// After 12 confirmations: marks tx as confirmed, updates escrow/task status
// On block_hash mismatch: marks as reorged, replays from last safe block
// Cross-references every event against outbox (rogue tx detection)

func (idx *Indexer) handleDeposited(event DepositedEvent) error
// Creates/updates escrow row (status=locked), updates task to Assigned
func (idx *Indexer) handleReleased(event ReleasedEvent) error
// Updates escrow to released, task to completed
func (idx *Indexer) handleRefunded(event RefundedEvent) error
func (idx *Indexer) handleSplit(event SplitEvent) error
func (idx *Indexer) handleReassigned(event ReassignedEvent) error
func (idx *Indexer) handleSubmitted(event SubmittedEvent) error
```

#### `internal/chain/relayer.go`

```go
type Relayer struct {
    client     *ethclient.Client
    privateKey *ecdsa.PrivateKey
    contract   common.Address
    chainID    *big.Int
}

func (r *Relayer) Release(ctx context.Context, taskIDBytes32 [32]byte) (common.Hash, error)
func (r *Relayer) Refund(ctx context.Context, taskIDBytes32 [32]byte) (common.Hash, error)
func (r *Relayer) Split(ctx context.Context, taskIDBytes32 [32]byte, agentBps uint16) (common.Hash, error)
func (r *Relayer) ReassignAgent(ctx context.Context, taskIDBytes32 [32]byte, newAgent common.Address) (common.Hash, error)
func (r *Relayer) MarkSubmitted(ctx context.Context, taskIDBytes32 [32]byte) (common.Hash, error)
```

#### `internal/chain/bond_verifier.go`

```go
type BondVerifier struct {
    client         *ethclient.Client
    usdcContract   common.Address
    bondOpsWallet  common.Address
    confirmations  uint64 // 12
}

// VerifyBond checks that a Transfer event exists in the given tx:
// - to == bondOpsWallet
// - amount == floor(escrow_amount_raw * 100 / 10000)
// - >= 12 block confirmations
func (v *BondVerifier) VerifyBond(ctx context.Context, txHash common.Hash, expectedAmount *big.Int) error
```

#### `internal/chain/bidhash.go`

```go
// TaskIDFromUUID converts a UUID to bytes32 via keccak256(abi.encodePacked(uuid_bytes16))
func TaskIDFromUUID(taskUUID uuid.UUID) [32]byte {
    return crypto.Keccak256Hash(taskUUID[:]) // UUID is already 16 bytes
}

// BidHash computes keccak256(abi.encode(domainSeparator, chainId, contract, taskId, agent, amount, etaHours, createdAt, bidId))
func ComputeBidHash(chainID uint64, contractAddr common.Address, taskID [32]byte, agent common.Address, amount *big.Int, etaHours uint32, createdAt uint64, bidID [32]byte) [32]byte
```

#### `internal/outbox/dispatcher.go`

```go
type Dispatcher struct {
    outboxRepo   *repository.OutboxRepo
    webhookRepo  *repository.WebhookRepo
    delivery     *webhook.Delivery
    pollInterval time.Duration // 1s
}

func (d *Dispatcher) Run(ctx context.Context) error
// Polls outbox for pending events
// For each event: find matching webhook subscriptions
// Deliver via webhook.Delivery
// Mark as dispatched or increment retry count
```

#### `internal/webhook/delivery.go`

```go
type Delivery struct {
    httpClient *http.Client
    timeout    time.Duration // 10s
}

func (d *Delivery) Deliver(ctx context.Context, url, secret string, payload WebhookPayload) error
// Signs payload with HMAC-SHA256(secret, body)
// Sets X-Marketplace-Signature header
// POST with 10s timeout
// Returns error on non-2xx

type WebhookPayload struct {
    Event            string    `json:"event"`
    IdempotencyKey   string    `json:"idempotency_key"`
    SequenceNumber   int64     `json:"event_sequence_number"`
    Timestamp        time.Time `json:"timestamp"`
    Data             any       `json:"data"`
}

// Retry policy: 3 retries, backoff 10s, 60s, 300s
// After 50 consecutive failures: disable webhook, notify operator
```

#### Webhook Event Types (from spec §09)

```
task.published, task.assigned, task.submitted, task.approved,
task.revision_requested, task.disputed, task.cancelled, task.unassigned,
task.expired, task.abandoned,
dispute.resolved, dispute.bond_timeout,
escrow.deposited, escrow.released, escrow.refunded, escrow.split, escrow.reassigned,
message.received
```

---

## 5. Integration Points

| Point | Producer | Consumer | Mechanism |
|-------|----------|----------|-----------|
| Deposit confirmed | Chain Indexer | TaskService | Indexer calls `TaskService.ConfirmDeposit()` |
| Release/Refund/Split needed | TaskService | Relayer | TaskService calls `EscrowService.Release()` which calls `Relayer` |
| markSubmitted needed | TaskService (on submit) | Relayer | `EscrowService.MarkSubmitted()` |
| Bond verification | DisputeService | BondVerifier | Before accepting dispute/response |
| Webhook delivery | Any state change | Outbox → Dispatcher → Delivery | Outbox pattern |
| Rogue tx detection | Chain Indexer | Alert system | Indexer cross-refs events vs outbox |
| Timeout checks | Scheduler | TaskService/DisputeService | Scheduler calls service methods |
| S3 upload confirmation | S3 callback / polling | ArtifactService.Finalize | After upload |

---

## 6. Deployment Plan

#### `docker-compose.yml`

```yaml
version: "3.8"
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: agentic
      POSTGRES_USER: agentic
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${S3_ACCESS_KEY}
      MINIO_ROOT_PASSWORD: ${S3_SECRET_KEY}
    volumes:
      - miniodata:/data
    ports:
      - "9000:9000"
      - "9001:9001"

  api:
    build: .
    depends_on:
      - postgres
      - minio
    environment:
      DATABASE_URL: postgres://agentic:${DB_PASSWORD}@postgres:5432/agentic?sslmode=disable
      BASE_RPC_URL: ${BASE_RPC_URL}
      ESCROW_CONTRACT: ${ESCROW_CONTRACT}
      RELAYER_PRIVATE_KEY: ${RELAYER_PRIVATE_KEY}
      USDC_CONTRACT: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
      FEE_TREASURY: ${FEE_TREASURY}
      BOND_OPS_WALLET: ${BOND_OPS_WALLET}
      S3_ENDPOINT: http://minio:9000
      S3_BUCKET: artifacts
      S3_ACCESS_KEY: ${S3_ACCESS_KEY}
      S3_SECRET_KEY: ${S3_SECRET_KEY}
      JWT_SECRET: ${JWT_SECRET}
      CHAIN_ID: "8453"
      PLATFORM_FEE_BPS: "500"
    ports:
      - "8080:8080"

volumes:
  pgdata:
  miniodata:
```

#### Coolify Deployment

- Deploy docker-compose via Coolify service
- Postgres can use Coolify's shared Postgres or dedicated container
- MinIO for S3 (or use Cloudflare R2 in production)
- Environment variables set in Coolify dashboard
- Contract deployed separately via Foundry script to Base

#### Contract Deployment

```bash
cd contracts
forge script script/Deploy.s.sol:Deploy \
  --rpc-url $BASE_RPC_URL \
  --broadcast \
  --verify \
  --etherscan-api-key $BASESCAN_API_KEY \
  -vvvv
```

---

## 7. Go Dependencies

```
github.com/jackc/pgx/v5          # Postgres driver
github.com/golang-migrate/migrate # DB migrations
github.com/google/uuid            # UUIDs
github.com/shopspring/decimal     # Decimal arithmetic (USDC amounts)
github.com/ethereum/go-ethereum   # Chain interaction
github.com/golang-jwt/jwt/v5     # JWT auth
github.com/spruceid/siwe-go      # SIWE verification
github.com/aws/aws-sdk-go-v2     # S3 (MinIO compatible)
golang.org/x/crypto              # bcrypt for API key hashing
```

---

## 8. Task Assignment for Coding Agents

### Agent 1: Smart Contract
- **Scope:** Everything in `contracts/`
- **Input:** This plan §Stream 1 + spec §05
- **Output:** `AgentEscrow.sol`, `IAgentEscrow.sol`, full test suite, `Deploy.s.sol`
- **Test command:** `cd contracts && forge test -vvv`
- **Zero ambiguity:** All function signatures, preconditions, and events specified above

### Agent 2: Database + Models + Repos
- **Scope:** `internal/database/`, `internal/models/`, `internal/repository/`, migrations
- **Input:** This plan §Stream 3 + spec §08
- **Output:** All migration files, all model structs, all repo implementations
- **Test command:** `go test ./internal/repository/... -v` (needs test Postgres)
- **Depends on:** Nothing (can start immediately)

### Agent 3: Services + Handlers + Middleware
- **Scope:** `internal/service/`, `internal/handler/`, `internal/middleware/`, `internal/s3/`, `cmd/server/main.go`
- **Input:** This plan §Stream 2 + spec §07
- **Output:** All business logic, HTTP handlers, auth middleware, idempotency middleware
- **Test command:** `go test ./internal/service/... ./internal/handler/... -v`
- **Depends on:** Agent 2 (repos + models must exist)

### Agent 4: Chain Integration + Outbox + Webhooks
- **Scope:** `internal/chain/`, `internal/outbox/`, `internal/webhook/`
- **Input:** This plan §Stream 4 + spec §05-§06-§09
- **Output:** Indexer, relayer, bond verifier, bidhash utils, outbox dispatcher, webhook delivery
- **Test command:** `go test ./internal/chain/... ./internal/outbox/... ./internal/webhook/... -v`
- **Depends on:** Agent 2 (repos), partially Agent 1 (ABI for event parsing)

---

## 9. Verification Checklist

Before declaring v0.1.5 complete:

- [ ] Contract: All 13 test categories pass (§Stream 1)
- [ ] Contract: Deployed to Base Sepolia testnet
- [ ] DB: Migrations up/down work cleanly
- [ ] API: All 30+ endpoints respond correctly
- [ ] API: Idempotency-Key works on all write endpoints
- [ ] API: Auth scopes enforced (agent can't approve, poster can't ack)
- [ ] Indexer: Deposits detected and confirmed after 12 blocks
- [ ] Indexer: Reorg handling tested
- [ ] Relayer: release/refund/split execute on-chain
- [ ] Webhooks: Delivered with HMAC signature
- [ ] Webhooks: Retry + dead letter works
- [ ] Scheduler: All 7 timeout rules fire correctly
- [ ] Bond verification: Rejects invalid bond tx hashes
- [ ] Artifact upload: Pre-signed URL → S3 → finalize flow works
- [ ] State machine: Every transition in §03 matrix tested
- [ ] Docker compose: `docker compose up` starts everything
- [ ] E2E: Happy path (create→bid→accept→deposit→ack→submit→approve) works end-to-end
