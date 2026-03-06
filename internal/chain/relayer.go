package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Relayer sends on-chain transactions for escrow operations
type Relayer struct {
	client      *ethclient.Client
	privateKey  *ecdsa.PrivateKey
	fromAddress common.Address
	contract    common.Address
	chainID     *big.Int
	nonceMu     sync.Mutex
	nonce       uint64
	nonceSet    bool
}

// NewRelayer creates a new relayer instance
func NewRelayer(
	client *ethclient.Client,
	privateKeyHex string,
	contractAddress common.Address,
	chainID *big.Int,
) (*Relayer, error) {
	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Derive address from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &Relayer{
		client:      client,
		privateKey:  privateKey,
		fromAddress: fromAddress,
		contract:    contractAddress,
		chainID:     chainID,
	}, nil
}

// getNonce gets the next nonce with proper synchronization
func (r *Relayer) getNonce(ctx context.Context) (uint64, error) {
	r.nonceMu.Lock()
	defer r.nonceMu.Unlock()

	if !r.nonceSet {
		// First time - fetch from chain
		nonce, err := r.client.PendingNonceAt(ctx, r.fromAddress)
		if err != nil {
			return 0, fmt.Errorf("failed to get nonce: %w", err)
		}
		r.nonce = nonce
		r.nonceSet = true
	}

	nonce := r.nonce
	r.nonce++ // Increment for next tx
	return nonce, nil
}

// resetNonce resets the nonce (e.g., after a failed tx)
func (r *Relayer) resetNonce() {
	r.nonceMu.Lock()
	defer r.nonceMu.Unlock()
	r.nonceSet = false
}

// buildAndSendTx builds a transaction with proper gas estimation and sends it
func (r *Relayer) buildAndSendTx(ctx context.Context, data []byte, gasLimit uint64) (common.Hash, error) {
	nonce, err := r.getNonce(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	// Get gas price
	gasPrice, err := r.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// Add 10% buffer to gas price for faster confirmation
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(110))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	// Estimate gas if not provided
	if gasLimit == 0 {
		msg := ethereum.CallMsg{
			From:     r.fromAddress,
			To:       &r.contract,
			GasPrice: gasPrice,
			Data:     data,
		}
		gasLimit, err = r.client.EstimateGas(ctx, msg)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to estimate gas: %w", err)
		}
		// Add 20% buffer
		gasLimit = gasLimit * 120 / 100
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		r.contract,
		big.NewInt(0), // No ETH transfer
		gasLimit,
		gasPrice,
		data,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(r.chainID), r.privateKey)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = r.client.SendTransaction(ctx, signedTx)
	if err != nil {
		// Reset nonce on send failure
		r.resetNonce()
		return common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash(), nil
}

// waitForConfirmation waits for a transaction to be confirmed
func (r *Relayer) waitForConfirmation(ctx context.Context, txHash common.Hash, confirmations uint64) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			receipt, err := r.client.TransactionReceipt(ctx, txHash)
			if err != nil {
				continue // Not yet mined
			}

			if receipt.Status == 0 {
				return fmt.Errorf("transaction failed on-chain")
			}

			// Check confirmations
			latestBlock, err := r.client.BlockNumber(ctx)
			if err != nil {
				return err
			}

			blockNumber := receipt.BlockNumber.Uint64()
			if latestBlock >= blockNumber+confirmations {
				return nil
			}
		}
	}
}

// Release sends a release transaction
func (r *Relayer) Release(ctx context.Context, taskIDBytes32 [32]byte) (common.Hash, error) {
	// ABI encode: release(bytes32)
	// function selector: first 4 bytes of keccak256("release(bytes32)")
	selector := crypto.Keccak256([]byte("release(bytes32)"))[:4]
	data := append(selector, taskIDBytes32[:]...)

	return r.buildAndSendTx(ctx, data, 0)
}

// Refund sends a refund transaction
func (r *Relayer) Refund(ctx context.Context, taskIDBytes32 [32]byte) (common.Hash, error) {
	// ABI encode: refund(bytes32)
	selector := crypto.Keccak256([]byte("refund(bytes32)"))[:4]
	data := append(selector, taskIDBytes32[:]...)

	return r.buildAndSendTx(ctx, data, 0)
}

// Split sends a split transaction
func (r *Relayer) Split(ctx context.Context, taskIDBytes32 [32]byte, agentBps uint16) (common.Hash, error) {
	// ABI encode: split(bytes32, uint16)
	selector := crypto.Keccak256([]byte("split(bytes32,uint16)"))[:4]
	
	// Encode parameters
	data := make([]byte, 4+32+32) // selector + taskId + agentBps
	copy(data[0:4], selector)
	copy(data[4:36], taskIDBytes32[:])
	
	// agentBps is uint16, needs to be padded to 32 bytes
	agentBpsBig := big.NewInt(int64(agentBps))
	agentBpsBytes := common.LeftPadBytes(agentBpsBig.Bytes(), 32)
	copy(data[36:68], agentBpsBytes)

	return r.buildAndSendTx(ctx, data, 0)
}

// ReassignAgent sends a reassign agent transaction
func (r *Relayer) ReassignAgent(ctx context.Context, taskIDBytes32 [32]byte, newAgent common.Address) (common.Hash, error) {
	// ABI encode: reassignAgent(bytes32, address)
	selector := crypto.Keccak256([]byte("reassignAgent(bytes32,address)"))[:4]
	
	data := make([]byte, 4+32+32) // selector + taskId + address
	copy(data[0:4], selector)
	copy(data[4:36], taskIDBytes32[:])
	
	// Address needs to be padded to 32 bytes (left-padded)
	addressBytes := common.LeftPadBytes(newAgent.Bytes(), 32)
	copy(data[36:68], addressBytes)

	return r.buildAndSendTx(ctx, data, 0)
}

// MarkSubmitted sends a markSubmitted transaction
func (r *Relayer) MarkSubmitted(ctx context.Context, taskIDBytes32 [32]byte) (common.Hash, error) {
	// ABI encode: markSubmitted(bytes32)
	selector := crypto.Keccak256([]byte("markSubmitted(bytes32)"))[:4]
	data := append(selector, taskIDBytes32[:]...)

	return r.buildAndSendTx(ctx, data, 0)
}

// GetTransactionOpts creates transaction options for contract interaction
func (r *Relayer) GetTransactionOpts(ctx context.Context) (*bind.TransactOpts, error) {
	nonce, err := r.getNonce(ctx)
	if err != nil {
		return nil, err
	}

	gasPrice, err := r.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// Add 10% buffer
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(110))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	auth, err := bind.NewKeyedTransactorWithChainID(r.privateKey, r.chainID)
	if err != nil {
		return nil, err
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasPrice = gasPrice
	auth.Context = ctx

	return auth, nil
}

// Address returns the relayer's address
func (r *Relayer) Address() common.Address {
	return r.fromAddress
}
