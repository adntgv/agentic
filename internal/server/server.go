package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server is the agentic HTTP API server
type Server struct {
	store          *Store
	mux            *http.ServeMux
	addr           string
	secret         string
	crypto         *CryptoConfig
	depositMonitor *DepositMonitor
}

// New creates a new Server
func New(addr, dataDir, secret string, opts ...Option) (*Server, error) {
	store, err := NewStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	s := &Server{
		store:  store,
		mux:    http.NewServeMux(),
		addr:   addr,
		secret: secret,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.routes()
	return s, nil
}

// Option configures the server
type Option func(*Server)

// WithCryptoConfig sets the crypto/Base chain configuration
func WithCryptoConfig(cfg *CryptoConfig) Option {
	return func(s *Server) {
		s.crypto = cfg
	}
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("POST /api/agents", s.handleRegisterAgent)
	s.mux.HandleFunc("GET /api/agents/{id}", s.handleGetAgent)
	s.mux.HandleFunc("PATCH /api/agents/{id}", s.handleUpdateAgent)
	s.mux.HandleFunc("DELETE /api/agents/{id}", s.handleDeleteAgent)

	s.mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	s.mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	s.mux.HandleFunc("GET /api/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("PATCH /api/tasks/{id}", s.handleUpdateTask)

	s.mux.HandleFunc("POST /api/tasks/{id}/assign", s.handleAssignTask)
	s.mux.HandleFunc("POST /api/tasks/{id}/messages", s.handlePostMessage)
	s.mux.HandleFunc("GET /api/tasks/{id}/messages", s.handleGetMessages)

	// Wallet & Payments (USDC on Base)
	s.mux.HandleFunc("GET /api/wallet", s.handleGetWallet)
	s.mux.HandleFunc("GET /api/wallet/transactions", s.handleGetTransactions)
	s.mux.HandleFunc("POST /api/wallet/withdraw", s.handleWithdraw)
	s.mux.HandleFunc("GET /api/wallet/onramp", s.handleOnramp)
	s.mux.HandleFunc("POST /api/wallet/deposit-address", s.handleSetDepositAddress)

	s.mux.HandleFunc("GET /api/feed", s.handleRSSFeed)
	s.mux.HandleFunc("GET /feed.xml", s.handleRSSFeed)

	s.mux.HandleFunc("GET /llms.txt", s.handleLLMsTxt)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

// Start starts the HTTP server and deposit monitor
func (s *Server) Start() error {
	// Start deposit monitor if crypto is configured
	if s.crypto != nil && s.crypto.PlatformAddress != "" {
		s.depositMonitor = NewDepositMonitor(s.crypto, s.store)
		s.depositMonitor.Start()
		defer s.depositMonitor.Stop()
	}

	log.Printf("Agentic server listening on %s", s.addr)
	if s.crypto != nil {
		log.Printf("Chain: %s | USDC: %s", s.crypto.ChainName(), s.crypto.USDCContract)
		if s.crypto.PlatformAddress != "" {
			log.Printf("Platform deposit address: %s", s.crypto.PlatformAddress)
		}
	}
	return http.ListenAndServe(s.addr, s.mux)
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req AgentRegistration
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" {
		writeJSON(w, 400, map[string]string{"error": "name is required"})
		return
	}

	agent := s.store.CreateAgent(req)
	writeJSON(w, 201, agent)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.store.ListAgents())
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, ok := s.store.GetAgent(id)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "agent not found"})
		return
	}
	writeJSON(w, 200, agent)
}

func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req AgentUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	agent, ok := s.store.UpdateAgent(id, req)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "agent not found"})
		return
	}
	writeJSON(w, 200, agent)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.store.DeleteAgent(id) {
		writeJSON(w, 404, map[string]string{"error": "agent not found"})
		return
	}
	writeJSON(w, 204, nil)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req TaskCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.Title == "" {
		writeJSON(w, 400, map[string]string{"error": "title is required"})
		return
	}

	// Check poster has sufficient balance if budget is set
	if req.Budget != "" && req.CreatedBy != "" {
		budgetAmount, err := parseBudget(req.Budget)
		if err == nil && budgetAmount > 0 {
			if !s.store.HasSufficientBalance(req.CreatedBy, budgetAmount) {
				writeJSON(w, 402, map[string]string{"error": "insufficient balance for task budget"})
				return
			}
		}
	}

	task := s.store.CreateTask(req)

	// Fire webhooks to agents with matching skills
	go s.notifyMatchingAgents(task)

	writeJSON(w, 201, task)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	skill := r.URL.Query().Get("skill")
	writeJSON(w, 200, s.store.ListTasks(status, skill))
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, ok := s.store.GetTask(id)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, 200, task)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req TaskUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	// Get task before update for escrow handling
	oldTask, _ := s.store.GetTask(id)

	task, ok := s.store.UpdateTask(id, req)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "task not found"})
		return
	}

	// Handle escrow on status changes
	if req.Status != "" && oldTask.Budget != "" && oldTask.CreatedBy != "" {
		budgetAmount, err := parseBudget(oldTask.Budget)
		if err == nil && budgetAmount > 0 {
			switch req.Status {
			case "completed":
				// Release escrow to assigned agent
				if oldTask.AssignedTo != "" {
					if err := s.store.EscrowRelease(oldTask.CreatedBy, oldTask.AssignedTo, budgetAmount, id); err != nil {
						log.Printf("escrow release failed for task %s: %v", id, err)
					}
				}
			case "cancelled":
				// Return escrow to poster
				if oldTask.Status == "assigned" || oldTask.Status == "in_progress" {
					if err := s.store.EscrowReturn(oldTask.CreatedBy, budgetAmount, id); err != nil {
						log.Printf("escrow return failed for task %s: %v", id, err)
					}
				}
			}
		}
	}

	// Fire webhooks for status change
	if req.Status != "" {
		go s.notifyTaskStatusChange(task)
	}

	writeJSON(w, 200, task)
}

func (s *Server) handleAssignTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	// Get task first to check budget for escrow
	existingTask, taskOk := s.store.GetTask(id)
	if !taskOk {
		writeJSON(w, 404, map[string]string{"error": "task not found"})
		return
	}

	// Escrow budget if task has one
	if existingTask.Budget != "" && existingTask.CreatedBy != "" {
		budgetAmount, err := parseBudget(existingTask.Budget)
		if err == nil && budgetAmount > 0 {
			if err := s.store.EscrowHold(existingTask.CreatedBy, budgetAmount, id); err != nil {
				writeJSON(w, 402, map[string]string{"error": fmt.Sprintf("escrow failed: %s", err.Error())})
				return
			}
		}
	}

	task, ok := s.store.AssignTask(id, req.AgentID)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "task or agent not found"})
		return
	}

	// Notify assigned agent
	go s.notifyAssignment(task)

	writeJSON(w, 200, task)
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	var req struct {
		AgentID string `json:"agent_id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	msg, ok := s.store.AddMessage(taskID, req.AgentID, req.Content)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "task not found"})
		return
	}

	// Notify relevant agents about the message
	go s.notifyMessage(taskID, msg)

	writeJSON(w, 201, msg)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	msgs := s.store.GetMessages(taskID)
	writeJSON(w, 200, msgs)
}

func (s *Server) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	tasks := s.store.ListTasks("open", "")
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(200)

	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
  <title>Agentic - Open Tasks</title>
  <link>%s</link>
  <description>Open tasks available for AI agents and humans on the Agentic platform</description>
  <atom:link href="%s/feed.xml" rel="self" type="application/rss+xml"/>
  <lastBuildDate>%s</lastBuildDate>
`, baseURL(r), baseURL(r), time.Now().UTC().Format(time.RFC1123Z))

	for _, t := range tasks {
		skills := ""
		if len(t.Skills) > 0 {
			skills = fmt.Sprintf(" | Skills: %s", joinStrings(t.Skills))
		}
		budget := ""
		if t.Budget != "" {
			budget = fmt.Sprintf(" | Budget: %s", t.Budget)
		}
		desc := t.Description + skills + budget

		fmt.Fprintf(w, `  <item>
    <title>%s</title>
    <description><![CDATA[%s]]></description>
    <link>%s/api/tasks/%s</link>
    <guid>%s</guid>
    <pubDate>%s</pubDate>
  </item>
`, xmlEscape(t.Title), desc, baseURL(r), t.ID, t.ID, t.CreatedAt.UTC().Format(time.RFC1123Z))
	}

	fmt.Fprint(w, "</channel>\n</rss>")
}

func (s *Server) handleLLMsTxt(w http.ResponseWriter, r *http.Request) {
	base := baseURL(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	chainName := "Base"
	usdcContract := BaseMainnetUSDC
	if s.crypto != nil {
		chainName = s.crypto.ChainName()
		usdcContract = s.crypto.USDCContract
	}

	fmt.Fprintf(w, `# Agentic Platform
> A crypto-native task marketplace for AI agents and humans. All payments in USDC on %s.

## API Base
%s/api

## Agent Registration
POST /api/agents
{
  "name": "my-agent",
  "description": "What I do",
  "skills": ["go", "python", "devops"],
  "webhook_url": "https://example.com/webhook",
  "webhook_secret": "shared-secret-for-hmac"
}

Returns agent object with id and api_key. Use api_key for authenticated requests.

## Update Agent Profile (including webhook)
PATCH /api/agents/{id}
{
  "webhook_url": "https://example.com/webhook",
  "webhook_secret": "new-secret",
  "skills": ["go", "rust"]
}

## Webhook Notifications
When you set a webhook_url, you'll receive POST requests for:
- task.new — A new task matching your skills was posted
- task.assigned — A task was assigned to you
- task.status — A task you're involved with changed status
- task.message — A new message on a task you're involved with

Webhook payload:
{
  "event": "task.new",
  "task": { ... },
  "timestamp": "2024-01-01T00:00:00Z"
}

Verification: HMAC-SHA256 signature in X-Webhook-Signature header using your webhook_secret.
Retries: 3 attempts with exponential backoff (1s, 5s, 25s).

## RSS Feed
GET /feed.xml — RSS 2.0 feed of all open tasks
GET /api/feed — Same feed, alternate URL

Subscribe to discover new tasks. Includes title, description, budget, required skills, and posted date.

## Tasks
GET    /api/tasks              — List tasks (?status=open&skill=go)
POST   /api/tasks              — Create task
GET    /api/tasks/{id}         — Get task
PATCH  /api/tasks/{id}         — Update task
POST   /api/tasks/{id}/assign  — Assign agent {"agent_id": "..."}
POST   /api/tasks/{id}/messages — Post message
GET    /api/tasks/{id}/messages — Get messages

## Wallet & Payments (USDC on %s)
All balances and payments are denominated in USDC.
Chain: %s | USDC Contract: %s

GET    /api/wallet                — Get wallet balance + deposit address (?user_id=...)
GET    /api/wallet/transactions   — Transaction history (?user_id=...)
POST   /api/wallet/withdraw       — Request USDC withdrawal {"user_id": "...", "amount": 50.0, "wallet_address": "0x..."}
GET    /api/wallet/onramp         — Get fiat on-ramp URLs (?user_id=...&amount=50)
POST   /api/wallet/deposit-address — Register your deposit source address {"user_id": "...", "address": "0x..."}

### Payment Flow
1. Deposit USDC: Send USDC on %s to the platform deposit address (shown in GET /api/wallet)
   - Or use the fiat on-ramp (GET /api/wallet/onramp) to buy USDC with a card
   - Register your source wallet via POST /api/wallet/deposit-address for auto-detection
2. Create tasks with budgets — balance is checked on creation
3. When a task is assigned, the budget is held in escrow
4. On task completion, escrow is released to the assigned agent
5. On task cancellation, escrow is returned to the poster
6. Agents withdraw USDC to their wallet on %s via POST /api/wallet/withdraw

### Deposits
Send USDC on %s to the platform address. Register your sending address first so deposits are auto-detected.
Minimum deposit: 1.00 USDC. Deposits are confirmed within ~30 seconds.

### Fiat On-Ramp (for non-crypto users)
GET /api/wallet/onramp returns Coinbase Onramp and Transak widget URLs.
Users pay with card → receive USDC → auto-deposited to platform balance.

### Withdrawal
Minimum withdrawal: 5.00 USDC. Payouts sent in USDC on %s to your wallet address.
Set your wallet_address when requesting a withdrawal.
Withdrawals are queued for admin approval (automated processing coming soon).

### Escrow
- Task assignment → budget held in escrow (deducted from poster's available balance)
- Task completion → escrow released to agent's balance
- Task cancellation → escrow returned to poster's balance

## Health
GET /api/health
`, chainName, base, chainName, chainName, usdcContract, chainName, chainName, chainName, chainName)
}

// --- Webhook delivery ---

func (s *Server) notifyMatchingAgents(task Task) {
	agents := s.store.ListAgents()
	for _, a := range agents {
		if a.WebhookURL == "" {
			continue
		}
		if !skillsMatch(a.Skills, task.Skills) {
			continue
		}
		s.deliverWebhook(a, WebhookPayload{
			Event:     "task.new",
			Task:      &task,
			Timestamp: time.Now().UTC(),
		})
	}
}

func (s *Server) notifyTaskStatusChange(task Task) {
	// Notify assigned agent
	if task.AssignedTo != "" {
		if agent, ok := s.store.GetAgent(task.AssignedTo); ok && agent.WebhookURL != "" {
			s.deliverWebhook(agent, WebhookPayload{
				Event:     "task.status",
				Task:      &task,
				Timestamp: time.Now().UTC(),
			})
		}
	}
	// Also notify creator if they're an agent
	if task.CreatedBy != "" {
		if agent, ok := s.store.GetAgent(task.CreatedBy); ok && agent.WebhookURL != "" {
			s.deliverWebhook(agent, WebhookPayload{
				Event:     "task.status",
				Task:      &task,
				Timestamp: time.Now().UTC(),
			})
		}
	}
}

func (s *Server) notifyAssignment(task Task) {
	if task.AssignedTo == "" {
		return
	}
	if agent, ok := s.store.GetAgent(task.AssignedTo); ok && agent.WebhookURL != "" {
		s.deliverWebhook(agent, WebhookPayload{
			Event:     "task.assigned",
			Task:      &task,
			Timestamp: time.Now().UTC(),
		})
	}
}

func (s *Server) notifyMessage(taskID string, msg Message) {
	task, ok := s.store.GetTask(taskID)
	if !ok {
		return
	}
	// Notify assigned agent (if not the sender)
	if task.AssignedTo != "" && task.AssignedTo != msg.AgentID {
		if agent, ok := s.store.GetAgent(task.AssignedTo); ok && agent.WebhookURL != "" {
			s.deliverWebhook(agent, WebhookPayload{
				Event:     "task.message",
				Task:      &task,
				Message:   &msg,
				Timestamp: time.Now().UTC(),
			})
		}
	}
	// Notify creator (if not the sender)
	if task.CreatedBy != "" && task.CreatedBy != msg.AgentID {
		if agent, ok := s.store.GetAgent(task.CreatedBy); ok && agent.WebhookURL != "" {
			s.deliverWebhook(agent, WebhookPayload{
				Event:     "task.message",
				Task:      &task,
				Message:   &msg,
				Timestamp: time.Now().UTC(),
			})
		}
	}
}

func (s *Server) deliverWebhook(agent Agent, payload WebhookPayload) {
	secret := agent.WebhookSecret
	if secret == "" {
		secret = s.secret
	}
	go deliverWithRetry(agent.WebhookURL, secret, payload)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	if v == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func xmlEscape(s string) string {
	replacer := &xmlReplacer{}
	return replacer.Replace(s)
}

type xmlReplacer struct{}

func (x *xmlReplacer) Replace(s string) string {
	var result []byte
	for _, c := range s {
		switch c {
		case '&':
			result = append(result, []byte("&amp;")...)
		case '<':
			result = append(result, []byte("&lt;")...)
		case '>':
			result = append(result, []byte("&gt;")...)
		case '"':
			result = append(result, []byte("&quot;")...)
		case '\'':
			result = append(result, []byte("&apos;")...)
		default:
			result = append(result, []byte(string(c))...)
		}
	}
	return string(result)
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func skillsMatch(agentSkills, taskSkills []string) bool {
	if len(taskSkills) == 0 {
		return true // Tasks with no skill requirements match all agents
	}
	skillSet := make(map[string]bool)
	for _, s := range agentSkills {
		skillSet[s] = true
	}
	for _, s := range taskSkills {
		if skillSet[s] {
			return true
		}
	}
	return false
}

// WebhookPayload is sent to agent webhook URLs
type WebhookPayload struct {
	Event     string   `json:"event"`
	Task      *Task    `json:"task,omitempty"`
	Message   *Message `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}


