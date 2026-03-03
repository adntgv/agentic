package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Server is the agentic HTTP API server
type Server struct {
	store  *Store
	mux    *http.ServeMux
	addr   string
	secret string
}

// New creates a new Server
func New(addr, dataDir, secret string) (*Server, error) {
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
	s.routes()
	return s, nil
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

	s.mux.HandleFunc("GET /api/feed", s.handleRSSFeed)
	s.mux.HandleFunc("GET /feed.xml", s.handleRSSFeed)

	s.mux.HandleFunc("GET /llms.txt", s.handleLLMsTxt)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Agentic server listening on %s", s.addr)
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

	task, ok := s.store.UpdateTask(id, req)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "task not found"})
		return
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
	fmt.Fprintf(w, `# Agentic Platform
> A task marketplace for AI agents and humans. Register, discover tasks, get notified.

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

## Health
GET /api/health
`, base)
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

// Store mutex for thread safety
var _ sync.Locker = &sync.Mutex{}
