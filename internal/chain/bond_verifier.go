package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BondVerifier verifies USDC Transfer events for dispute bonds
type BondVerifier struct {
	client         *ethclient.Client
	usdcContract   common.Address
	bondOpsWallet  common.Address
	confirmations  uint64
}

// NewBondVerifier creates a new bond verifier
func NewBondVerifier(
	client *ethclient.Client,
	usdcContract common.Address,
	bondOpsWallet common.Address,
	confirmations uint64,
) *BondVerifier {
	return &BondVerifier{
		client:         client,
		usdcContract:   usdcContract,
		bondOpsWallet:  bondOpsWallet,
		confirmations:  confirmations,
	}
}

// ERC20 Transfer event signature
var transferEventSignature = []byte("Transfer(address,address,uint256)")

// VerifyBond checks that a Transfer event exists in the given tx:
// - to == bondOpsWallet
// - amount == expectedAmount (floor(escrow_amount_raw * 100 / 10000))
// - >= 12 block confirmations
func (v *BondVerifier) VerifyBond(ctx context.Context, txHash common.Hash, expectedAmount *big.Int) error {
	// Get transaction receipt
	receipt, err := v.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Check confirmations
	latestBlock, err := v.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	blockNumber := receipt.BlockNumber.Uint64()
	if latestBlock < blockNumber+v.confirmations {
		return fmt.Errorf("insufficient confirmations: have %d, need %d", latestBlock-blockNumber, v.confirmations)
	}

	// Parse Transfer events from receipt
	transferEventTopic := common.BytesToHash(transferEventSignature)

	found := false
	for _, log := range receipt.Logs {
		// Check if this is a Transfer event from the USDC contract
		if log.Address != v.usdcContract {
			continue
		}
		if len(log.Topics) < 3 {
			continue
		}
		if log.Topics[0] != transferEventTopic {
			continue
		}

		// Parse Transfer event: indexed from, indexed to, uint256 value
		to := common.BytesToAddress(log.Topics[2].Bytes())
		if to != v.bondOpsWallet {
			continue
		}

		// Parse amount from log data
		if len(log.Data) != 32 {
			continue
		}
		amount := new(big.Int).SetBytes(log.Data)

		// Check amount matches
		if amount.Cmp(expectedAmount) == 0 {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no valid Transfer event found: expected %s USDC to %s", expectedAmount.String(), v.bondOpsWallet.Hex())
	}

	return nil
}

// ComputeBondAmount computes the bond amount: floor(escrow_amount_raw * 100 / 10000)
// escrowAmount is in USDC (6 decimals)
func ComputeBondAmount(escrowAmount *big.Int) *big.Int {
	// bond = floor(escrow * 100 / 10000) = floor(escrow / 100)
	// This is 1% of the escrow amount
	hundred := big.NewInt(100)
	bondAmount := new(big.Int).Div(escrowAmount, hundred)
	return bondAmount
}

// VerifyBondWithEscrow is a convenience method that computes expected bond and verifies
func (v *BondVerifier) VerifyBondWithEscrow(ctx context.Context, txHash common.Hash, escrowAmount *big.Int) error {
	expectedBond := ComputeBondAmount(escrowAmount)
	return v.VerifyBond(ctx, txHash, expectedBond)
}

// ParseTransferEvents parses all Transfer events from a receipt (for debugging/analysis)
func ParseTransferEvents(receipt *types.Receipt, usdcContract common.Address) ([]TransferEvent, error) {
	transferABI := `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

	parsedABI, err := abi.JSON(strings.NewReader(transferABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	var events []TransferEvent
	for _, log := range receipt.Logs {
		if log.Address != usdcContract {
			continue
		}

		event := TransferEvent{}
		err := parsedABI.UnpackIntoInterface(&event, "Transfer", log.Data)
		if err != nil {
			continue
		}

		if len(log.Topics) >= 3 {
			event.From = common.BytesToAddress(log.Topics[1].Bytes())
			event.To = common.BytesToAddress(log.Topics[2].Bytes())
		}

		events = append(events, event)
	}

	return events, nil
}

// TransferEvent represents a USDC Transfer event
type TransferEvent struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}
