package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// --- Wallet Models ---

// Wallet holds a user's USDC credit balance
type Wallet struct {
	UserID            string  `json:"user_id"`
	Balance           float64 `json:"balance"`              // available USDC credits
	EscrowHeld        float64 `json:"escrow_held"`          // frozen in escrow
	WalletAddress     string  `json:"wallet_address"`       // Base chain address for withdrawals
	DepositAddress    string  `json:"deposit_address"`      // registered source address for deposit matching
	HDIndex           int     `json:"hd_index"`             // HD derivation index for per-user deposit address
	UniqueDepositAddr string  `json:"unique_deposit_addr"`  // derived HD deposit address for this user
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Transaction is a double-entry ledger record
type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"` // deposit, escrow_hold, escrow_release, escrow_return, withdrawal, withdrawal_failed
	Amount      float64   `json:"amount"`      // positive = credit, negative = debit
	Balance     float64   `json:"balance"`     // balance after this transaction
	RefID       string    `json:"ref_id"`      // reference: tx hash, task ID, withdrawal ID
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// WithdrawalRequest tracks USDC payout requests
type WithdrawalRequest struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Amount        float64   `json:"amount"`
	WalletAddress string    `json:"wallet_address"`
	Status        string    `json:"status"` // pending, processing, completed, failed
	TxHash        string    `json:"tx_hash,omitempty"` // on-chain tx hash once sent
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// --- Wallet Store Methods ---

func (s *Store) GetOrCreateWallet(userID string) Wallet {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := s.wallets[userID]; ok {
		return w
	}

	now := time.Now().UTC()

	// Assign next HD index
	hdIndex := s.nextHDIndex
	s.nextHDIndex++

	w := Wallet{
		UserID:    userID,
		Balance:   0,
		HDIndex:   hdIndex,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Derive unique deposit address if HD wallet is configured
	if s.hdDeriver != nil {
		addr, err := s.hdDeriver.DeriveAddress(hdIndex)
		if err != nil {
			log.Printf("wallet: failed to derive HD address for user %s index %d: %v", userID, hdIndex, err)
		} else {
			w.UniqueDepositAddr = addr
			// Index the derived address for deposit matching
			if s.hdAddressIndex == nil {
				s.hdAddressIndex = make(map[string]string)
			}
			s.hdAddressIndex[strings.ToLower(addr)] = userID
		}
	}

	s.wallets[userID] = w
	s.saveToDisk()
	return w
}

func (s *Store) GetWallet(userID string) (Wallet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.wallets[userID]
	return w, ok
}

func (s *Store) SetWalletAddress(userID, address string) Wallet {
	s.mu.Lock()
	defer s.mu.Unlock()

	w := s.wallets[userID]
	w.WalletAddress = address
	w.UpdatedAt = time.Now().UTC()
	s.wallets[userID] = w
	s.saveToDisk()
	return w
}

func (s *Store) SetDepositAddress(userID, address string) Wallet {
	s.mu.Lock()
	defer s.mu.Unlock()

	w := s.wallets[userID]
	w.DepositAddress = address
	w.UpdatedAt = time.Now().UTC()
	s.wallets[userID] = w

	// Also index for reverse lookup
	if s.depositAddressIndex == nil {
		s.depositAddressIndex = make(map[string]string)
	}
	s.depositAddressIndex[strings.ToLower(address)] = userID

	s.saveToDisk()
	return w
}

// FindUserByDepositAddress finds which user registered a given source address
// It checks both registered source addresses AND HD-derived deposit addresses
func (s *Store) FindUserByDepositAddress(addr string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lower := strings.ToLower(addr)

	// Check registered source addresses
	if s.depositAddressIndex != nil {
		if userID, ok := s.depositAddressIndex[lower]; ok {
			return userID
		}
	}

	// Check HD-derived deposit addresses (deposit TO this address)
	if s.hdAddressIndex != nil {
		if userID, ok := s.hdAddressIndex[lower]; ok {
			return userID
		}
	}

	// Fallback: scan wallets (slower)
	for _, w := range s.wallets {
		if strings.ToLower(w.DepositAddress) == lower {
			return w.UserID
		}
		if strings.ToLower(w.UniqueDepositAddr) == lower {
			return w.UserID
		}
	}
	return ""
}

// GetAllHDDepositAddresses returns all derived deposit addresses for monitoring
func (s *Store) GetAllHDDepositAddresses() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var addrs []string
	for _, w := range s.wallets {
		if w.UniqueDepositAddr != "" {
			addrs = append(addrs, w.UniqueDepositAddr)
		}
	}
	return addrs
}

// IsTransactionProcessed checks if a tx hash has already been credited
func (s *Store) IsTransactionProcessed(txHash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.processedTxs == nil {
		return false
	}
	return s.processedTxs[txHash]
}

func (s *Store) AddTransaction(userID, txType string, amount float64, refID, description string) Transaction {
	s.mu.Lock()
	defer s.mu.Unlock()

	w := s.wallets[userID]
	w.Balance += amount
	w.UpdatedAt = time.Now().UTC()
	s.wallets[userID] = w

	tx := Transaction{
		ID:          generateID(),
		UserID:      userID,
		Type:        txType,
		Amount:      amount,
		Balance:     w.Balance,
		RefID:       refID,
		Description: description,
		CreatedAt:   time.Now().UTC(),
	}
	s.transactions[userID] = append(s.transactions[userID], tx)

	// Track processed tx hashes to avoid double-crediting
	if txType == "deposit" && refID != "" {
		if s.processedTxs == nil {
			s.processedTxs = make(map[string]bool)
		}
		s.processedTxs[refID] = true
	}

	s.saveToDisk()
	return tx
}

func (s *Store) GetTransactions(userID string) []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transactions[userID]
}

// EscrowHold freezes USDC credits from poster's balance for a task
func (s *Store) EscrowHold(userID string, amount float64, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.wallets[userID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}
	if w.Balance < amount {
		return fmt.Errorf("insufficient balance: have %.2f USDC, need %.2f USDC", w.Balance, amount)
	}

	w.Balance -= amount
	w.EscrowHeld += amount
	w.UpdatedAt = time.Now().UTC()
	s.wallets[userID] = w

	tx := Transaction{
		ID:          generateID(),
		UserID:      userID,
		Type:        "escrow_hold",
		Amount:      -amount,
		Balance:     w.Balance,
		RefID:       taskID,
		Description: fmt.Sprintf("Escrow hold for task %s", taskID),
		CreatedAt:   time.Now().UTC(),
	}
	s.transactions[userID] = append(s.transactions[userID], tx)
	s.saveToDisk()
	return nil
}

// EscrowRelease moves escrowed funds to the agent's balance
func (s *Store) EscrowRelease(posterID, agentID string, amount float64, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pw, ok := s.wallets[posterID]
	if !ok {
		return fmt.Errorf("poster wallet not found")
	}
	if pw.EscrowHeld < amount {
		return fmt.Errorf("insufficient escrow: have %.2f, need %.2f", pw.EscrowHeld, amount)
	}
	pw.EscrowHeld -= amount
	pw.UpdatedAt = time.Now().UTC()
	s.wallets[posterID] = pw

	aw := s.wallets[agentID]
	if aw.UserID == "" {
		aw = Wallet{UserID: agentID, CreatedAt: time.Now().UTC()}
	}
	aw.Balance += amount
	aw.UpdatedAt = time.Now().UTC()
	s.wallets[agentID] = aw

	now := time.Now().UTC()
	s.transactions[posterID] = append(s.transactions[posterID], Transaction{
		ID: generateID(), UserID: posterID, Type: "escrow_release",
		Amount: 0, Balance: pw.Balance, RefID: taskID,
		Description: fmt.Sprintf("Escrow released to agent %s for task %s", agentID, taskID),
		CreatedAt:   now,
	})
	s.transactions[agentID] = append(s.transactions[agentID], Transaction{
		ID: generateID(), UserID: agentID, Type: "escrow_release",
		Amount: amount, Balance: aw.Balance, RefID: taskID,
		Description: fmt.Sprintf("Payment received for task %s", taskID),
		CreatedAt:   now,
	})

	s.saveToDisk()
	return nil
}

// EscrowReturn returns escrowed funds to the poster (task cancelled)
func (s *Store) EscrowReturn(posterID string, amount float64, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.wallets[posterID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}
	if w.EscrowHeld < amount {
		return fmt.Errorf("insufficient escrow")
	}

	w.EscrowHeld -= amount
	w.Balance += amount
	w.UpdatedAt = time.Now().UTC()
	s.wallets[posterID] = w

	s.transactions[posterID] = append(s.transactions[posterID], Transaction{
		ID: generateID(), UserID: posterID, Type: "escrow_return",
		Amount: amount, Balance: w.Balance, RefID: taskID,
		Description: fmt.Sprintf("Escrow returned for cancelled task %s", taskID),
		CreatedAt:   time.Now().UTC(),
	})

	s.saveToDisk()
	return nil
}

// CreateWithdrawal creates a USDC withdrawal request
func (s *Store) CreateWithdrawal(userID string, amount float64, walletAddress string) (WithdrawalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.wallets[userID]
	if !ok {
		return WithdrawalRequest{}, fmt.Errorf("wallet not found")
	}
	if w.Balance < amount {
		return WithdrawalRequest{}, fmt.Errorf("insufficient balance: have %.2f USDC, need %.2f USDC", w.Balance, amount)
	}
	if amount < 5.0 {
		return WithdrawalRequest{}, fmt.Errorf("minimum withdrawal is 5.00 USDC")
	}

	w.Balance -= amount
	w.UpdatedAt = time.Now().UTC()
	s.wallets[userID] = w

	now := time.Now().UTC()
	wr := WithdrawalRequest{
		ID:            generateID(),
		UserID:        userID,
		Amount:        amount,
		WalletAddress: walletAddress,
		Status:        "pending",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.withdrawals[wr.ID] = wr

	s.transactions[userID] = append(s.transactions[userID], Transaction{
		ID: generateID(), UserID: userID, Type: "withdrawal",
		Amount: -amount, Balance: w.Balance, RefID: wr.ID,
		Description: fmt.Sprintf("Withdrawal request %.2f USDC to %s", amount, walletAddress),
		CreatedAt:   now,
	})

	s.saveToDisk()
	return wr, nil
}

func (s *Store) GetWithdrawals(userID string) []WithdrawalRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []WithdrawalRequest
	for _, wr := range s.withdrawals {
		if wr.UserID == userID {
			result = append(result, wr)
		}
	}
	return result
}

func (s *Store) UpdateWithdrawalStatus(id, status, txHash string) (WithdrawalRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wr, ok := s.withdrawals[id]
	if !ok {
		return WithdrawalRequest{}, false
	}

	oldStatus := wr.Status
	wr.Status = status
	wr.TxHash = txHash
	wr.UpdatedAt = time.Now().UTC()
	s.withdrawals[id] = wr

	if status == "failed" && oldStatus != "failed" {
		w := s.wallets[wr.UserID]
		w.Balance += wr.Amount
		w.UpdatedAt = time.Now().UTC()
		s.wallets[wr.UserID] = w

		s.transactions[wr.UserID] = append(s.transactions[wr.UserID], Transaction{
			ID: generateID(), UserID: wr.UserID, Type: "withdrawal_failed",
			Amount: wr.Amount, Balance: w.Balance, RefID: id,
			Description: "Withdrawal failed, funds returned",
			CreatedAt:   time.Now().UTC(),
		})
	}

	s.saveToDisk()
	return wr, true
}

// HasSufficientBalance checks if user can afford a budget
func (s *Store) HasSufficientBalance(userID string, amount float64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.wallets[userID]
	if !ok {
		return false
	}
	return w.Balance >= amount
}

// --- HTTP Handlers ---

func (s *Server) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, 400, map[string]string{"error": "user_id is required"})
		return
	}
	wallet := s.store.GetOrCreateWallet(userID)

	// Include deposit info - prefer user's unique HD address, fallback to platform address
	depositAddr := ""
	if wallet.UniqueDepositAddr != "" {
		depositAddr = wallet.UniqueDepositAddr
	} else if s.crypto != nil {
		depositAddr = s.crypto.GetDepositAddress()
	}

	platformAddr := ""
	if s.crypto != nil {
		platformAddr = s.crypto.GetDepositAddress()
	}

	writeJSON(w, 200, map[string]interface{}{
		"user_id":              wallet.UserID,
		"balance":              wallet.Balance,
		"escrow_held":          wallet.EscrowHeld,
		"wallet_address":       wallet.WalletAddress,
		"deposit_address":      depositAddr,
		"platform_address":     platformAddr,
		"deposit_source":       wallet.DepositAddress,
		"unique_deposit_addr":  wallet.UniqueDepositAddr,
		"hd_index":             wallet.HDIndex,
		"currency":             "USDC",
		"chain":                s.chainName(),
		"created_at":           wallet.CreatedAt,
		"updated_at":           wallet.UpdatedAt,
	})
}

func (s *Server) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, 400, map[string]string{"error": "user_id is required"})
		return
	}
	txs := s.store.GetTransactions(userID)
	if txs == nil {
		txs = []Transaction{}
	}
	writeJSON(w, 200, txs)
}

func (s *Server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID        string  `json:"user_id"`
		Amount        float64 `json:"amount"`
		WalletAddress string  `json:"wallet_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.UserID == "" || req.WalletAddress == "" {
		writeJSON(w, 400, map[string]string{"error": "user_id and wallet_address are required"})
		return
	}

	// Validate wallet address
	if !IsValidAddress(req.WalletAddress) {
		writeJSON(w, 400, map[string]string{"error": "invalid wallet address, must be a valid 0x Ethereum address"})
		return
	}

	// Save wallet address to profile
	s.store.SetWalletAddress(req.UserID, req.WalletAddress)

	wr, err := s.store.CreateWithdrawal(req.UserID, req.Amount, req.WalletAddress)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("withdrawal request: %.2f USDC to %s for user %s (ID: %s)",
		wr.Amount, wr.WalletAddress, wr.UserID, wr.ID)

	writeJSON(w, 200, map[string]interface{}{
		"withdrawal":  wr,
		"message":     "Withdrawal queued for admin approval. USDC will be sent to your wallet on Base.",
		"chain":       s.chainName(),
	})
}

func (s *Server) handleOnramp(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	amountStr := r.URL.Query().Get("amount")
	if userID == "" {
		writeJSON(w, 400, map[string]string{"error": "user_id is required"})
		return
	}

	amount := 50.0 // default
	if amountStr != "" {
		if a, err := strconv.ParseFloat(amountStr, 64); err == nil && a > 0 {
			amount = a
		}
	}

	if s.crypto == nil {
		writeJSON(w, 503, map[string]string{"error": "crypto not configured"})
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"coinbase_onramp_url": s.crypto.CoinbaseOnrampURL(userID, amount),
		"transak_url":         s.crypto.TransakOnrampURL(userID, amount),
		"amount":              amount,
		"currency":            "USDC",
		"chain":               s.crypto.ChainName(),
		"note":                "Use either on-ramp to purchase USDC with a card. Funds will be deposited to the platform automatically.",
	})
}

func (s *Server) handleSetDepositAddress(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"user_id"`
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.UserID == "" || req.Address == "" {
		writeJSON(w, 400, map[string]string{"error": "user_id and address are required"})
		return
	}
	if !IsValidAddress(req.Address) {
		writeJSON(w, 400, map[string]string{"error": "invalid address format"})
		return
	}

	s.store.GetOrCreateWallet(req.UserID)
	wallet := s.store.SetDepositAddress(req.UserID, req.Address)

	depositAddr := ""
	if s.crypto != nil {
		depositAddr = s.crypto.GetDepositAddress()
	}

	writeJSON(w, 200, map[string]interface{}{
		"user_id":           wallet.UserID,
		"deposit_source":    wallet.DepositAddress,
		"platform_address":  depositAddr,
		"message":           "Send USDC on Base from this address to the platform address. Deposits are auto-detected.",
	})
}

func (s *Server) chainName() string {
	if s.crypto != nil {
		return s.crypto.ChainName()
	}
	return "Base"
}

// --- Helpers ---

func parseBudget(budget string) (float64, error) {
	str := budget
	for _, prefix := range []string{"$", "USD", "usd", "USDC", "usdc"} {
		if len(str) > len(prefix) && strings.EqualFold(str[:len(prefix)], prefix) {
			str = str[len(prefix):]
			break
		}
	}
	str = strings.TrimSpace(str)
	return strconv.ParseFloat(str, 64)
}
