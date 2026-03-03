package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"
)

// CryptoConfig holds Base chain and USDC configuration
type CryptoConfig struct {
	PlatformPrivateKey string // hex-encoded private key for sending withdrawals
	BaseRPCURL         string // Base chain RPC endpoint
	USDCContract       string // USDC contract address on Base
	UseTestnet         bool   // use Base Sepolia testnet
	PlatformAddress    string // derived from private key, the main deposit address
}

const (
	// Base Mainnet
	BaseMainnetRPC     = "https://mainnet.base.org"
	BaseMainnetChainID = 8453
	BaseMainnetUSDC    = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"

	// Base Sepolia Testnet
	BaseSepoliaRPC     = "https://sepolia.base.org"
	BaseSepoliaChainID = 84532
	BaseSepoliaUSDC    = "0x036CbD53842c5426634e7929541eC2318f3dCF7e" // USDC on Base Sepolia

	// USDC has 6 decimals
	USDCDecimals = 6

	// ERC-20 Transfer event signature: Transfer(address,address,uint256)
	TransferEventSig = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	// Minimum deposit amount (in USDC, human-readable)
	MinDepositAmount = 1.0

	// Deposit polling interval
	DepositPollInterval = 15 * time.Second
)

// DefaultCryptoConfig returns config with mainnet defaults
func DefaultCryptoConfig() CryptoConfig {
	return CryptoConfig{
		BaseRPCURL:   BaseMainnetRPC,
		USDCContract: BaseMainnetUSDC,
	}
}

// ApplyTestnet switches config to Base Sepolia
func (c *CryptoConfig) ApplyTestnet() {
	c.BaseRPCURL = BaseSepoliaRPC
	c.USDCContract = BaseSepoliaUSDC
	c.UseTestnet = true
}

// ChainID returns the appropriate chain ID
func (c *CryptoConfig) ChainID() int64 {
	if c.UseTestnet {
		return BaseSepoliaChainID
	}
	return BaseMainnetChainID
}

// ChainName returns human-readable chain name
func (c *CryptoConfig) ChainName() string {
	if c.UseTestnet {
		return "Base Sepolia (testnet)"
	}
	return "Base"
}

// --- Address Validation ---

// IsValidAddress checks if a string is a valid Ethereum address
func IsValidAddress(addr string) bool {
	if !strings.HasPrefix(addr, "0x") && !strings.HasPrefix(addr, "0X") {
		return false
	}
	if len(addr) != 42 {
		return false
	}
	// Check hex chars
	_, err := hex.DecodeString(addr[2:])
	return err == nil
}

// --- USDC Amount Conversions ---

// USDCToRaw converts human-readable USDC (e.g. 10.50) to raw units (10500000)
func USDCToRaw(amount float64) *big.Int {
	// Multiply by 10^6 (USDC decimals)
	raw := new(big.Float).Mul(
		new(big.Float).SetFloat64(amount),
		new(big.Float).SetFloat64(1_000_000),
	)
	result, _ := raw.Int(nil)
	return result
}

// RawToUSDC converts raw USDC units to human-readable float
func RawToUSDC(raw *big.Int) float64 {
	if raw == nil {
		return 0
	}
	f := new(big.Float).SetInt(raw)
	f.Quo(f, new(big.Float).SetFloat64(1_000_000))
	result, _ := f.Float64()
	return result
}

// --- Deposit Address ---
// For MVP: all users deposit to the platform address. The platform detects
// deposits by monitoring Transfer events to the platform address, matching
// by sender address (users must register their deposit source address).
//
// Future: generate per-user deposit addresses using HD wallets (BIP-44 derivation).

// GetDepositAddress returns the platform's USDC deposit address.
// Users send USDC to this address and get credited after on-chain confirmation.
func (c *CryptoConfig) GetDepositAddress() string {
	return c.PlatformAddress
}

// --- Fiat On-Ramp ---

// CoinbaseOnrampURL generates a Coinbase Onramp widget URL for a user
// The user pays with fiat and receives USDC directly to the platform address.
func (c *CryptoConfig) CoinbaseOnrampURL(userID string, amount float64) string {
	dest := c.GetDepositAddress()
	chainID := c.ChainID()
	// Coinbase Onramp widget URL format
	return fmt.Sprintf(
		"https://pay.coinbase.com/buy/select-asset?appId=agentic&destinationWallets=[{\"address\":\"%s\",\"assets\":[\"USDC\"],\"supportedNetworks\":[\"base\"]}]&defaultAsset=USDC&defaultPaymentMethod=CARD&presetFiatAmount=%.0f&fiatCurrency=USD&partnerUserId=%s&chainId=%d",
		dest, amount, userID, chainID,
	)
}

// TransakOnrampURL generates a Transak widget URL as fallback on-ramp
func (c *CryptoConfig) TransakOnrampURL(userID string, amount float64) string {
	dest := c.GetDepositAddress()
	network := "base"
	if c.UseTestnet {
		network = "base_sepolia"
	}
	return fmt.Sprintf(
		"https://global.transak.com/?apiKey=TRANSAK_API_KEY&cryptoCurrencyCode=USDC&network=%s&walletAddress=%s&fiatAmount=%.0f&fiatCurrency=USD&partnerCustomerId=%s&disableWalletAddressForm=true",
		network, dest, amount, userID,
	)
}

// --- Deposit Monitor (Lightweight EVM RPC) ---
// Instead of importing go-ethereum (heavy dependency), we use raw JSON-RPC calls
// to poll for USDC Transfer events to our platform address.

// DepositMonitor polls for incoming USDC transfers
type DepositMonitor struct {
	config    *CryptoConfig
	store     *Store
	lastBlock uint64
	stopCh    chan struct{}
}

// NewDepositMonitor creates a new deposit monitor
func NewDepositMonitor(config *CryptoConfig, store *Store) *DepositMonitor {
	return &DepositMonitor{
		config: config,
		store:  store,
		stopCh: make(chan struct{}),
	}
}

// Start begins polling for deposits in a background goroutine
func (dm *DepositMonitor) Start() {
	if dm.config.PlatformAddress == "" {
		log.Println("crypto: no platform address configured, deposit monitoring disabled")
		return
	}

	log.Printf("crypto: starting deposit monitor on %s", dm.config.ChainName())
	log.Printf("crypto: monitoring USDC transfers to %s", dm.config.PlatformAddress)
	log.Printf("crypto: USDC contract: %s", dm.config.USDCContract)

	go dm.pollLoop()
}

// Stop halts the deposit monitor
func (dm *DepositMonitor) Stop() {
	close(dm.stopCh)
}

func (dm *DepositMonitor) pollLoop() {
	ticker := time.NewTicker(DepositPollInterval)
	defer ticker.Stop()

	// Get current block on first run
	currentBlock, err := dm.getBlockNumber()
	if err != nil {
		log.Printf("crypto: failed to get initial block number: %v", err)
		dm.lastBlock = 0
	} else {
		dm.lastBlock = currentBlock
		log.Printf("crypto: starting from block %d", dm.lastBlock)
	}

	for {
		select {
		case <-dm.stopCh:
			log.Println("crypto: deposit monitor stopped")
			return
		case <-ticker.C:
			dm.checkDeposits()
		}
	}
}

func (dm *DepositMonitor) checkDeposits() {
	currentBlock, err := dm.getBlockNumber()
	if err != nil {
		log.Printf("crypto: failed to get block number: %v", err)
		return
	}

	if currentBlock <= dm.lastBlock {
		return
	}

	// Query Transfer events to our platform address
	deposits, err := dm.queryTransferEvents(dm.lastBlock+1, currentBlock)
	if err != nil {
		log.Printf("crypto: failed to query events: %v", err)
		return
	}

	for _, dep := range deposits {
		log.Printf("crypto: detected deposit of %.6f USDC from %s (tx: %s)",
			dep.Amount, dep.From, dep.TxHash)

		// Find user by their registered deposit source address
		userID := dm.store.FindUserByDepositAddress(dep.From)
		if userID == "" {
			log.Printf("crypto: unknown sender %s, skipping (users must register source address)", dep.From)
			continue
		}

		// Check if this tx was already processed
		if dm.store.IsTransactionProcessed(dep.TxHash) {
			log.Printf("crypto: tx %s already processed, skipping", dep.TxHash)
			continue
		}

		// Credit the user
		dm.store.GetOrCreateWallet(userID)
		dm.store.AddTransaction(userID, "deposit", dep.Amount, dep.TxHash,
			fmt.Sprintf("USDC deposit %.6f from %s", dep.Amount, dep.From))

		log.Printf("crypto: credited %.6f USDC to user %s", dep.Amount, userID)
	}

	dm.lastBlock = currentBlock
}

// Deposit represents a detected USDC transfer
type Deposit struct {
	From    string
	To      string
	Amount  float64
	TxHash  string
	Block   uint64
}

// getBlockNumber calls eth_blockNumber via JSON-RPC
func (dm *DepositMonitor) getBlockNumber() (uint64, error) {
	result, err := ethRPCCall(dm.config.BaseRPCURL, "eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}
	return parseHexUint64(result)
}

// queryTransferEvents calls eth_getLogs for USDC Transfer events to platform address
func (dm *DepositMonitor) queryTransferEvents(fromBlock, toBlock uint64) ([]Deposit, error) {
	// Pad the platform address to 32 bytes for topic matching
	paddedAddr := "0x000000000000000000000000" + strings.ToLower(dm.config.PlatformAddress[2:])

	params := []interface{}{
		map[string]interface{}{
			"fromBlock": fmt.Sprintf("0x%x", fromBlock),
			"toBlock":   fmt.Sprintf("0x%x", toBlock),
			"address":   dm.config.USDCContract,
			"topics": []interface{}{
				TransferEventSig, // Transfer event
				nil,              // any sender
				paddedAddr,       // to our platform address
			},
		},
	}

	result, err := ethRPCCallArray(dm.config.BaseRPCURL, "eth_getLogs", params)
	if err != nil {
		return nil, err
	}

	var deposits []Deposit
	for _, logEntry := range result {
		logMap, ok := logEntry.(map[string]interface{})
		if !ok {
			continue
		}

		topics, _ := logMap["topics"].([]interface{})
		if len(topics) < 3 {
			continue
		}

		data, _ := logMap["data"].(string)
		txHash, _ := logMap["transactionHash"].(string)
		blockHex, _ := logMap["blockNumber"].(string)

		// Parse sender from topic[1] (padded address)
		fromTopic, _ := topics[1].(string)
		from := "0x" + fromTopic[26:] // strip 0x + 24 zero chars

		// Parse amount from data (uint256, 6 decimals for USDC)
		rawAmount := new(big.Int)
		if len(data) > 2 {
			rawAmount.SetString(data[2:], 16)
		}
		amount := RawToUSDC(rawAmount)

		block, _ := parseHexUint64(blockHex)

		deposits = append(deposits, Deposit{
			From:   from,
			To:     dm.config.PlatformAddress,
			Amount: amount,
			TxHash: txHash,
			Block:  block,
		})
	}

	return deposits, nil
}

// --- Withdrawal Sender ---
// For MVP: withdrawals are queued and require admin approval.
// The actual on-chain send uses raw transaction construction with JSON-RPC.

// ProcessWithdrawal sends USDC to the agent's wallet address.
// This is called by the admin approval flow.
func ProcessWithdrawal(config *CryptoConfig, wr WithdrawalRequest) (string, error) {
	if config.PlatformPrivateKey == "" {
		return "", fmt.Errorf("platform private key not configured")
	}

	_ = context.Background()
	_ = (*ecdsa.PrivateKey)(nil) // placeholder for actual crypto signing

	// For MVP: log the withdrawal for manual processing
	// In production, this would:
	// 1. Build ERC-20 transfer(to, amount) calldata
	// 2. Sign with platform private key
	// 3. Send via eth_sendRawTransaction
	// 4. Return tx hash

	log.Printf("crypto: withdrawal queued - %.6f USDC to %s (ID: %s)",
		wr.Amount, wr.WalletAddress, wr.ID)

	// TODO: Implement actual on-chain sending when ready
	// For now, return empty hash (admin processes manually)
	return "", fmt.Errorf("automated withdrawals not yet enabled; process manually and update status via admin API")
}

// --- JSON-RPC Helpers (lightweight, no go-ethereum dependency) ---

func ethRPCCall(rpcURL, method string, params []interface{}) (string, error) {
	return ethRPCCallGeneric[string](rpcURL, method, params)
}

func ethRPCCallArray(rpcURL, method string, params []interface{}) ([]interface{}, error) {
	return ethRPCCallGeneric[[]interface{}](rpcURL, method, params)
}

func ethRPCCallGeneric[T any](rpcURL, method string, params []interface{}) (T, error) {
	var zero T

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	body, err := jsonMarshal(reqBody)
	if err != nil {
		return zero, err
	}

	resp, err := httpPost(rpcURL, "application/json", body)
	if err != nil {
		return zero, fmt.Errorf("RPC request failed: %w", err)
	}

	var rpcResp struct {
		Result T              `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := jsonUnmarshal(resp, &rpcResp); err != nil {
		return zero, fmt.Errorf("RPC parse failed: %w", err)
	}

	if rpcResp.Error != nil {
		return zero, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func parseHexUint64(hexStr string) (uint64, error) {
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		hexStr = hexStr[2:]
	}
	val := new(big.Int)
	if _, ok := val.SetString(hexStr, 16); !ok {
		return 0, fmt.Errorf("invalid hex: %s", hexStr)
	}
	return val.Uint64(), nil
}
