package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// OutboxEvent represents an event in the outbox table
type OutboxEvent struct {
	ID              int64           `json:"id"`
	EventType       string          `json:"event_type"`
	AggregateID     string          `json:"aggregate_id"` // taskId, disputeId, etc.
	Payload         json.RawMessage `json:"payload"`
	Status          string          `json:"status"` // pending, dispatched, failed
	RetryCount      int             `json:"retry_count"`
	MaxRetries      int             `json:"max_retries"`
	LastError       *string         `json:"last_error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	DispatchedAt    *time.Time      `json:"dispatched_at,omitempty"`
	NextRetryAfter  *time.Time      `json:"next_retry_after,omitempty"`
}

// Webhook represents a webhook subscription
type Webhook struct {
	ID         int64    `json:"id"`
	WorkerID   string   `json:"worker_id"`
	URL        string   `json:"url"`
	Secret     string   `json:"secret"`
	Events     []string `json:"events"` // List of subscribed event types
	Active     bool     `json:"active"`
	FailCount  int      `json:"fail_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Repository interfaces
type OutboxRepository interface {
	GetPendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkDispatched(ctx context.Context, eventID int64) error
	IncrementRetry(ctx context.Context, eventID int64, errorMsg string, nextRetryAfter time.Time) error
	MarkFailed(ctx context.Context, eventID int64, errorMsg string) error
}

type WebhookRepository interface {
	FindSubscribers(ctx context.Context, eventType string) ([]Webhook, error)
	IncrementFailCount(ctx context.Context, webhookID int64) error
	DisableWebhook(ctx context.Context, webhookID int64, reason string) error
}

// WebhookDelivery interface for delivering webhooks
type WebhookDelivery interface {
	Deliver(ctx context.Context, webhook Webhook, payload WebhookPayload) error
}

// WebhookPayload represents the payload sent to webhook endpoints
type WebhookPayload struct {
	Event            string      `json:"event"`
	IdempotencyKey   string      `json:"idempotency_key"`
	SequenceNumber   int64       `json:"event_sequence_number"`
	Timestamp        time.Time   `json:"timestamp"`
	Data             interface{} `json:"data"`
}

// Dispatcher polls the outbox table and dispatches events to webhooks
type Dispatcher struct {
	outboxRepo   OutboxRepository
	webhookRepo  WebhookRepository
	delivery     WebhookDelivery
	pollInterval time.Duration
	batchSize    int
	
	stopCh    chan struct{}
	stoppedCh chan struct{}
}

// NewDispatcher creates a new outbox dispatcher
func NewDispatcher(
	outboxRepo OutboxRepository,
	webhookRepo WebhookRepository,
	delivery WebhookDelivery,
	pollInterval time.Duration,
	batchSize int,
) *Dispatcher {
	return &Dispatcher{
		outboxRepo:   outboxRepo,
		webhookRepo:  webhookRepo,
		delivery:     delivery,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		stopCh:       make(chan struct{}),
		stoppedCh:    make(chan struct{}),
	}
}

// Run starts the dispatcher loop
func (d *Dispatcher) Run(ctx context.Context) error {
	defer close(d.stoppedCh)
	
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	log.Printf("Starting outbox dispatcher (poll interval: %v, batch size: %d)", d.pollInterval, d.batchSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.stopCh:
			return nil
		case <-ticker.C:
			if err := d.processEvents(ctx); err != nil {
				log.Printf("Error processing outbox events: %v", err)
				// Continue on error, don't stop the dispatcher
			}
		}
	}
}

// Stop stops the dispatcher
func (d *Dispatcher) Stop() {
	close(d.stopCh)
	<-d.stoppedCh
}

// processEvents fetches and processes pending events
func (d *Dispatcher) processEvents(ctx context.Context) error {
	events, err := d.outboxRepo.GetPendingEvents(ctx, d.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending events: %w", err)
	}

	if len(events) == 0 {
		return nil // No events to process
	}

	log.Printf("Processing %d outbox events", len(events))

	for _, event := range events {
		if err := d.dispatchEvent(ctx, event); err != nil {
			log.Printf("Error dispatching event %d: %v", event.ID, err)
			// Continue processing other events
		}
	}

	return nil
}

// dispatchEvent dispatches a single event to all matching webhooks
func (d *Dispatcher) dispatchEvent(ctx context.Context, event OutboxEvent) error {
	// Find all webhooks subscribed to this event type
	webhooks, err := d.webhookRepo.FindSubscribers(ctx, event.EventType)
	if err != nil {
		return fmt.Errorf("failed to find subscribers: %w", err)
	}

	if len(webhooks) == 0 {
		// No subscribers, mark as dispatched
		return d.outboxRepo.MarkDispatched(ctx, event.ID)
	}

	// Parse event payload
	var payloadData interface{}
	if err := json.Unmarshal(event.Payload, &payloadData); err != nil {
		return fmt.Errorf("failed to parse event payload: %w", err)
	}

	// Build webhook payload
	payload := WebhookPayload{
		Event:            event.EventType,
		IdempotencyKey:   fmt.Sprintf("outbox-%d", event.ID),
		SequenceNumber:   event.ID,
		Timestamp:        event.CreatedAt,
		Data:             payloadData,
	}

	// Deliver to each webhook
	allSuccess := true
	var lastError error

	for _, webhook := range webhooks {
		if !webhook.Active {
			continue
		}

		deliveryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := d.delivery.Deliver(deliveryCtx, webhook, payload)
		cancel()

		if err != nil {
			log.Printf("Failed to deliver event %d to webhook %d: %v", event.ID, webhook.ID, err)
			allSuccess = false
			lastError = err

			// Increment webhook fail count
			if err := d.webhookRepo.IncrementFailCount(ctx, webhook.ID); err != nil {
				log.Printf("Failed to increment fail count for webhook %d: %v", webhook.ID, err)
			}

			// Check if webhook should be disabled (50 consecutive failures)
			if webhook.FailCount+1 >= 50 {
				log.Printf("Disabling webhook %d after 50 consecutive failures", webhook.ID)
				if err := d.webhookRepo.DisableWebhook(ctx, webhook.ID, "50 consecutive failures"); err != nil {
					log.Printf("Failed to disable webhook %d: %v", webhook.ID, err)
				}
			}
		} else {
			log.Printf("Successfully delivered event %d to webhook %d (%s)", event.ID, webhook.ID, webhook.URL)
		}
	}

	// Update event status
	if allSuccess {
		return d.outboxRepo.MarkDispatched(ctx, event.ID)
	} else {
		// Schedule retry
		if event.RetryCount >= event.MaxRetries {
			// Max retries reached, mark as failed
			errorMsg := "max retries reached"
			if lastError != nil {
				errorMsg = lastError.Error()
			}
			return d.outboxRepo.MarkFailed(ctx, event.ID, errorMsg)
		}

		// Calculate next retry time with exponential backoff
		nextRetry := calculateNextRetry(event.RetryCount)
		errorMsg := ""
		if lastError != nil {
			errorMsg = lastError.Error()
		}
		
		return d.outboxRepo.IncrementRetry(ctx, event.ID, errorMsg, nextRetry)
	}
}

// calculateNextRetry calculates the next retry time with exponential backoff
// Backoff: 10s, 60s, 300s (5 min), then 300s for subsequent retries
func calculateNextRetry(retryCount int) time.Time {
	var delay time.Duration
	switch retryCount {
	case 0:
		delay = 10 * time.Second
	case 1:
		delay = 60 * time.Second
	default:
		delay = 300 * time.Second
	}
	return time.Now().Add(delay)
}

// EventType constants
const (
	EventTaskPublished         = "task.published"
	EventTaskAssigned          = "task.assigned"
	EventTaskSubmitted         = "task.submitted"
	EventTaskApproved          = "task.approved"
	EventTaskRevisionRequested = "task.revision_requested"
	EventTaskDisputed          = "task.disputed"
	EventTaskCancelled         = "task.cancelled"
	EventTaskUnassigned        = "task.unassigned"
	EventTaskExpired           = "task.expired"
	EventTaskAbandoned         = "task.abandoned"
	
	EventDisputeResolved     = "dispute.resolved"
	EventDisputeBondTimeout  = "dispute.bond_timeout"
	
	EventEscrowDeposited   = "escrow.deposited"
	EventEscrowReleased    = "escrow.released"
	EventEscrowRefunded    = "escrow.refunded"
	EventEscrowSplit       = "escrow.split"
	EventEscrowReassigned  = "escrow.reassigned"
	
	EventMessageReceived = "message.received"
)
