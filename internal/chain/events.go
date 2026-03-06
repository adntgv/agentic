package chain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Contract event signatures
var (
	// Event signatures (keccak256 of event signature string)
	DepositedEventSig       = common.HexToHash("0x...") // Will be updated when ABI is available
	ReleasedEventSig        = common.HexToHash("0x...")
	RefundedEventSig        = common.HexToHash("0x...")
	SplitEventSig           = common.HexToHash("0x...")
	ReassignedAgentEventSig = common.HexToHash("0x...")
	SubmittedEventSig       = common.HexToHash("0x...")
)

// DepositedEvent represents the Deposited event
// event Deposited(bytes32 indexed taskId, address indexed poster, address indexed agent, uint96 amount, bytes32 bidHash)
type DepositedEvent struct {
	TaskID  [32]byte
	Poster  common.Address
	Agent   common.Address
	Amount  *big.Int
	BidHash [32]byte
}

// ReleasedEvent represents the Released event
// event Released(bytes32 indexed taskId, address indexed agent, uint96 agentAmount, uint96 feeAmount)
type ReleasedEvent struct {
	TaskID       [32]byte
	Agent        common.Address
	AgentAmount  *big.Int
	FeeAmount    *big.Int
}

// RefundedEvent represents the Refunded event
// event Refunded(bytes32 indexed taskId, address indexed poster, uint96 amount)
type RefundedEvent struct {
	TaskID [32]byte
	Poster common.Address
	Amount *big.Int
}

// SplitEvent represents the Split event
// event Split(bytes32 indexed taskId, address indexed agent, uint96 agentAmount, address indexed poster, uint96 posterAmount)
type SplitEvent struct {
	TaskID       [32]byte
	Agent        common.Address
	AgentAmount  *big.Int
	Poster       common.Address
	PosterAmount *big.Int
}

// ReassignedAgentEvent represents the ReassignedAgent event
// event ReassignedAgent(bytes32 indexed taskId, address indexed oldAgent, address indexed newAgent)
type ReassignedAgentEvent struct {
	TaskID   [32]byte
	OldAgent common.Address
	NewAgent common.Address
}

// SubmittedEvent represents the Submitted event
// event Submitted(bytes32 indexed taskId)
type SubmittedEvent struct {
	TaskID [32]byte
}

// BlockInfo stores block metadata for reorg detection
type BlockInfo struct {
	Number    uint64
	Hash      common.Hash
	Timestamp uint64
}

// EventLog wraps an event with its block info
type EventLog struct {
	Event     interface{}
	BlockInfo BlockInfo
	TxHash    common.Hash
	LogIndex  uint
}
