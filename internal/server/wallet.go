package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// --- Wallet Models ---

// Wallet holds a user's credit balance
type Wallet struct {
	UserID        string  `json:"user_id"`
	Balance       float64 `json:"balance"`        // available credits (USD)
	EscrowHeld    float64 `json:"escrow_held"`    // frozen in escrow
	WalletAddress string  `json:"wallet_address"` // Base chain USDC address for withdrawals
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Transaction is a double-entry ledger record
type Transaction struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"` // deposit, escrow_hold, escrow_release, escrow_return, withdrawal, withdrawal_failed
	Amount      float64   `json:"amount"`      // positive = credit, negative = debit
	Balance     float64   `json:"balance"`     // balance after this transaction
	RefID       string    `json:"ref_id"`      // reference: order ID, task ID, withdrawal ID
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

// LemonSqueezy credit packages
var creditPackages = map[string]float64{
	"credits_10":  10.0,
	"credits_25":  25.0,
	"credits_50":  50.0,
	"credits_100": 100.0,
}

// --- Wallet Store Methods ---

func (s *Store) GetOrCreateWallet(userID string) Wallet {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := s.wallets[userID]; ok {
		return w
	}

	now := time.Now().UTC()
	w := Wallet{
		UserID:    userID,
		Balance:   0,
		CreatedAt: now,
		UpdatedAt: now,
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
	s.saveToDisk()
	return tx
}

func (s *Store) GetTransactions(userID string) []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transactions[userID]
}

// EscrowHold freezes credits from poster's balance for a task
func (s *Store) EscrowHold(userID string, amount float64, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.wallets[userID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}
	if w.Balance < amount {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", w.Balance, amount)
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

	// Deduct from poster's escrow
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

	// Credit agent's balance
	aw := s.wallets[agentID]
	if aw.UserID == "" {
		aw = Wallet{UserID: agentID, CreatedAt: time.Now().UTC()}
	}
	aw.Balance += amount
	aw.UpdatedAt = time.Now().UTC()
	s.wallets[agentID] = aw

	now := time.Now().UTC()
	// Log for poster
	s.transactions[posterID] = append(s.transactions[posterID], Transaction{
		ID: generateID(), UserID: posterID, Type: "escrow_release",
		Amount: 0, Balance: pw.Balance, RefID: taskID,
		Description: fmt.Sprintf("Escrow released to agent %s for task %s", agentID, taskID),
		CreatedAt:   now,
	})
	// Log for agent
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

// CreateWithdrawal creates a withdrawal request
func (s *Store) CreateWithdrawal(userID string, amount float64, walletAddress string) (WithdrawalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.wallets[userID]
	if !ok {
		return WithdrawalRequest{}, fmt.Errorf("wallet not found")
	}
	if w.Balance < amount {
		return WithdrawalRequest{}, fmt.Errorf("insufficient balance: have %.2f, need %.2f", w.Balance, amount)
	}
	if amount < 5.0 {
		return WithdrawalRequest{}, fmt.Errorf("minimum withdrawal is $5.00")
	}

	// Deduct balance
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

	// If withdrawal failed, refund
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

// HasSufficientBalance checks if user can afford a budget (used before task creation)
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
	writeJSON(w, 200, wallet)
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

func (s *Server) handleCreateDeposit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"user_id"`
		Package string `json:"package"` // credits_10, credits_25, credits_50, credits_100
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.UserID == "" {
		writeJSON(w, 400, map[string]string{"error": "user_id is required"})
		return
	}

	amount, ok := creditPackages[req.Package]
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "invalid package, use: credits_10, credits_25, credits_50, credits_100"})
		return
	}

	// Create LemonSqueezy checkout
	checkoutURL, err := s.createLemonSqueezyCheckout(req.UserID, req.Package, amount)
	if err != nil {
		log.Printf("lemonsqueezy: checkout creation failed: %v", err)
		writeJSON(w, 500, map[string]string{"error": "failed to create checkout"})
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"checkout_url": checkoutURL,
		"package":      req.Package,
		"amount":       amount,
	})
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

	// Save wallet address to profile
	s.store.SetWalletAddress(req.UserID, req.WalletAddress)

	wr, err := s.store.CreateWithdrawal(req.UserID, req.Amount, req.WalletAddress)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, wr)
}

func (s *Server) handleLemonSqueezyWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "failed to read body"})
		return
	}

	// Verify signature if webhook secret is configured
	if s.secret != "" {
		sig := r.Header.Get("X-Signature")
		if sig == "" {
			sig = r.Header.Get("X-Webhook-Signature")
		}
		if !verifyLemonSqueezySignature(body, sig, s.secret) {
			writeJSON(w, 401, map[string]string{"error": "invalid signature"})
			return
		}
	}

	// Parse LemonSqueezy webhook payload
	var payload struct {
		Meta struct {
			EventName  string            `json:"event_name"`
			CustomData map[string]string `json:"custom_data"`
		} `json:"meta"`
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				OrderNumber int    `json:"order_number"`
				Status      string `json:"status"`
				Total       int    `json:"total"`       // in cents
				TotalFormatted string `json:"total_formatted"`
				FirstOrderItem struct {
					ProductName string `json:"product_name"`
					VariantName string `json:"variant_name"`
				} `json:"first_order_item"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("lemonsqueezy webhook: failed to parse: %v", err)
		writeJSON(w, 400, map[string]string{"error": "invalid payload"})
		return
	}

	log.Printf("lemonsqueezy webhook: event=%s order=%s status=%s",
		payload.Meta.EventName, payload.Data.ID, payload.Data.Attributes.Status)

	// Only process successful orders
	if payload.Meta.EventName != "order_created" {
		writeJSON(w, 200, map[string]string{"status": "ignored"})
		return
	}

	userID := payload.Meta.CustomData["user_id"]
	packageID := payload.Meta.CustomData["package"]
	if userID == "" {
		log.Printf("lemonsqueezy webhook: no user_id in custom_data")
		writeJSON(w, 200, map[string]string{"status": "no_user"})
		return
	}

	amount, ok := creditPackages[packageID]
	if !ok {
		// Fallback: use the actual payment amount
		amount = float64(payload.Data.Attributes.Total) / 100.0
	}

	// Ensure wallet exists
	s.store.GetOrCreateWallet(userID)

	// Credit the user
	orderID := payload.Data.ID
	s.store.AddTransaction(userID, "deposit", amount, orderID,
		fmt.Sprintf("LemonSqueezy deposit $%.2f (order %s)", amount, orderID))

	log.Printf("lemonsqueezy webhook: credited %.2f to user %s (order %s)", amount, userID, orderID)
	writeJSON(w, 200, map[string]string{"status": "credited"})
}

// --- LemonSqueezy API ---

const lemonSqueezyAPIBase = "https://api.lemonsqueezy.com/v1"

func (s *Server) createLemonSqueezyCheckout(userID, packageID string, amount float64) (string, error) {
	// Build checkout request
	// Note: In production you'd have store_id and variant_id configured
	// For now we create a dynamic checkout
	reqBody := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "checkouts",
			"attributes": map[string]interface{}{
				"custom_price": int(amount * 100), // cents
				"product_options": map[string]interface{}{
					"name":        fmt.Sprintf("Agentic Credits - $%.0f", amount),
					"description": fmt.Sprintf("Purchase $%.0f in Agentic platform credits", amount),
				},
				"checkout_data": map[string]interface{}{
					"custom": map[string]string{
						"user_id": userID,
						"package": packageID,
					},
				},
			},
			"relationships": map[string]interface{}{
				"store": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "stores",
						"id":   "agentic", // placeholder - set actual store ID
					},
				},
				"variant": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "variants",
						"id":   "1", // placeholder - set actual variant ID
					},
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", lemonSqueezyAPIBase+"/checkouts", io.NopCloser(
		io.Reader(json.NewDecoder(io.NopCloser(nil)).Buffered()),
	))
	// Simplified: just build the request properly
	req, err = http.NewRequest("POST", lemonSqueezyAPIBase+"/checkouts",
		io.NopCloser(jsonReader(body)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+s.lemonSqueezyKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data.Attributes.URL, nil
}

func verifyLemonSqueezySignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// parseBudget extracts a float64 from a budget string like "$50" or "50.00"
func parseBudget(budget string) (float64, error) {
	// Strip $ and whitespace
	s := budget
	for _, prefix := range []string{"$", "USD", "usd"} {
		if len(s) > len(prefix) && s[:len(prefix)] == prefix {
			s = s[len(prefix):]
		}
	}
	s = trimSpace(s)
	return strconv.ParseFloat(s, 64)
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func jsonReader(data []byte) io.Reader {
	return &bytesReader{data: data}
}
