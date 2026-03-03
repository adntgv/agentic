package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is a simple file-backed data store
type Store struct {
	mu       sync.RWMutex
	dataDir  string
	agents   map[string]Agent
	tasks    map[string]Task
	messages map[string][]Message // taskID -> messages
}

// NewStore creates a new Store, loading existing data from disk
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		dataDir:  dataDir,
		agents:   make(map[string]Agent),
		tasks:    make(map[string]Task),
		messages: make(map[string][]Message),
	}

	// Load existing data
	s.loadFromDisk()
	return s, nil
}

func (s *Store) CreateAgent(req AgentRegistration) Agent {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	agent := Agent{
		ID:            generateID(),
		Name:          req.Name,
		Description:   req.Description,
		Skills:        req.Skills,
		WebhookURL:    req.WebhookURL,
		WebhookSecret: req.WebhookSecret,
		APIKey:        generateAPIKey(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.agents[agent.ID] = agent
	s.saveToDisk()
	return agent
}

func (s *Store) GetAgent(id string) (Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[id]
	return a, ok
}

func (s *Store) ListAgents() []Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agents := make([]Agent, 0, len(s.agents))
	for _, a := range s.agents {
		// Don't expose secrets in list
		a.WebhookSecret = ""
		a.APIKey = ""
		agents = append(agents, a)
	}
	return agents
}

func (s *Store) UpdateAgent(id string, req AgentUpdate) (Agent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[id]
	if !ok {
		return Agent{}, false
	}

	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.Skills != nil {
		agent.Skills = req.Skills
	}
	if req.WebhookURL != "" {
		agent.WebhookURL = req.WebhookURL
	}
	if req.WebhookSecret != "" {
		agent.WebhookSecret = req.WebhookSecret
	}
	agent.UpdatedAt = time.Now().UTC()

	s.agents[id] = agent
	s.saveToDisk()
	return agent, true
}

func (s *Store) DeleteAgent(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.agents[id]; !ok {
		return false
	}
	delete(s.agents, id)
	s.saveToDisk()
	return true
}

func (s *Store) CreateTask(req TaskCreate) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	task := Task{
		ID:          generateID(),
		Title:       req.Title,
		Description: req.Description,
		Status:      "open",
		Skills:      req.Skills,
		Budget:      req.Budget,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tasks[task.ID] = task
	s.saveToDisk()
	return task
}

func (s *Store) GetTask(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *Store) ListTasks(status, skill string) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, 0)
	for _, t := range s.tasks {
		if status != "" && t.Status != status {
			continue
		}
		if skill != "" && !containsSkill(t.Skills, skill) {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks
}

func (s *Store) UpdateTask(id string, req TaskUpdate) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return Task{}, false
	}

	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Status != "" {
		task.Status = req.Status
	}
	if req.Budget != "" {
		task.Budget = req.Budget
	}
	task.UpdatedAt = time.Now().UTC()

	s.tasks[id] = task
	s.saveToDisk()
	return task, true
}

func (s *Store) AssignTask(taskID, agentID string) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return Task{}, false
	}
	if _, ok := s.agents[agentID]; !ok {
		return Task{}, false
	}

	task.AssignedTo = agentID
	task.Status = "assigned"
	task.UpdatedAt = time.Now().UTC()

	s.tasks[taskID] = task
	s.saveToDisk()
	return task, true
}

func (s *Store) AddMessage(taskID, agentID, content string) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[taskID]; !ok {
		return Message{}, false
	}

	msg := Message{
		ID:        generateID(),
		TaskID:    taskID,
		AgentID:   agentID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	s.messages[taskID] = append(s.messages[taskID], msg)
	s.saveToDisk()
	return msg, true
}

func (s *Store) GetMessages(taskID string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messages[taskID]
}

// --- Persistence ---

type storeData struct {
	Agents   map[string]Agent      `json:"agents"`
	Tasks    map[string]Task       `json:"tasks"`
	Messages map[string][]Message  `json:"messages"`
}

func (s *Store) saveToDisk() {
	data := storeData{
		Agents:   s.agents,
		Tasks:    s.tasks,
		Messages: s.messages,
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(s.dataDir, "store.json"), b, 0644)
}

func (s *Store) loadFromDisk() {
	b, err := os.ReadFile(filepath.Join(s.dataDir, "store.json"))
	if err != nil {
		return
	}
	var data storeData
	if err := json.Unmarshal(b, &data); err != nil {
		return
	}
	if data.Agents != nil {
		s.agents = data.Agents
	}
	if data.Tasks != nil {
		s.tasks = data.Tasks
	}
	if data.Messages != nil {
		s.messages = data.Messages
	}
}

// --- Utilities ---

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return fmt.Sprintf("ag_%s", hex.EncodeToString(b))
}

func containsSkill(skills []string, skill string) bool {
	for _, s := range skills {
		if s == skill {
			return true
		}
	}
	return false
}
