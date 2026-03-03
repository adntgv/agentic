package server

import "time"

// Agent represents a registered agent
type Agent struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Skills         []string  `json:"skills,omitempty"`
	WebhookURL     string    `json:"webhook_url,omitempty"`
	WebhookSecret  string    `json:"webhook_secret,omitempty"`
	APIKey         string    `json:"api_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AgentRegistration is the request body for registering an agent
type AgentRegistration struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	WebhookURL    string   `json:"webhook_url,omitempty"`
	WebhookSecret string   `json:"webhook_secret,omitempty"`
}

// AgentUpdate is the request body for updating an agent
type AgentUpdate struct {
	Name          string   `json:"name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Skills        []string `json:"skills,omitempty"`
	WebhookURL    string   `json:"webhook_url,omitempty"`
	WebhookSecret string   `json:"webhook_secret,omitempty"`
}

// Task represents a task in the system
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"` // open, assigned, in_progress, completed, cancelled
	Skills      []string  `json:"skills,omitempty"`
	Budget      string    `json:"budget,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TaskCreate is the request body for creating a task
type TaskCreate struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Skills      []string `json:"skills,omitempty"`
	Budget      string   `json:"budget,omitempty"`
	CreatedBy   string   `json:"created_by,omitempty"`
}

// TaskUpdate is the request body for updating a task
type TaskUpdate struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Budget      string `json:"budget,omitempty"`
}

// Message represents a message on a task
type Message struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	AgentID   string    `json:"agent_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
