package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebhookPayload represents the payload sent to webhook endpoints
type WebhookPayload struct {
	Event            string      `json:"event"`
	IdempotencyKey   string      `json:"idempotency_key"`
	SequenceNumber   int64       `json:"event_sequence_number"`
	Timestamp        time.Time   `json:"timestamp"`
	Data             interface{} `json:"data"`
}

// Webhook represents a webhook subscription
type Webhook struct {
	ID         int64     `json:"id"`
	WorkerID   string    `json:"worker_id"`
	URL        string    `json:"url"`
	Secret     string    `json:"secret"`
	Events     []string  `json:"events"`
	Active     bool      `json:"active"`
	FailCount  int       `json:"fail_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Delivery handles webhook delivery with HMAC signing and retry logic
type Delivery struct {
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

// NewDelivery creates a new webhook delivery handler
func NewDelivery(timeout time.Duration, maxRetries int) *Delivery {
	return &Delivery{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

// Deliver sends a webhook with HMAC-SHA256 signing
func (d *Delivery) Deliver(ctx context.Context, webhook Webhook, payload WebhookPayload) error {
	// Marshal payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Compute HMAC signature
	signature := d.computeSignature(body, webhook.Secret)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Marketplace-Signature", signature)
	req.Header.Set("X-Marketplace-Event", payload.Event)
	req.Header.Set("X-Marketplace-Idempotency-Key", payload.IdempotencyKey)
	req.Header.Set("X-Marketplace-Timestamp", payload.Timestamp.Format(time.RFC3339))
	req.Header.Set("User-Agent", "Agentic-Marketplace-Webhook/v0.1.5")

	// Send request with retries
	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff for retries
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := d.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request failed (attempt %d): %w", attempt+1, err)
			continue
		}

		// Read response body (for debugging/logging)
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // Success
		}

		lastErr = fmt.Errorf("http status %d (attempt %d): %s", resp.StatusCode, attempt+1, string(respBody))

		// Don't retry on 4xx errors (except 429 Too Many Requests)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			break
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("webhook delivery failed after %d attempts", d.maxRetries+1)
}

// computeSignature computes HMAC-SHA256 signature for the payload
func (d *Delivery) computeSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies the HMAC signature of a webhook payload
func VerifySignature(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// DeliveryResult contains the result of a webhook delivery attempt
type DeliveryResult struct {
	Success    bool
	StatusCode int
	Error      error
	Attempts   int
	Duration   time.Duration
}

// DeliverWithResult sends a webhook and returns detailed result information
func (d *Delivery) DeliverWithResult(ctx context.Context, webhook Webhook, payload WebhookPayload) DeliveryResult {
	startTime := time.Now()
	result := DeliveryResult{}

	err := d.Deliver(ctx, webhook, payload)
	
	result.Duration = time.Since(startTime)
	result.Success = (err == nil)
	result.Error = err

	return result
}

// BatchDeliver sends a webhook to multiple endpoints in parallel
func (d *Delivery) BatchDeliver(ctx context.Context, webhooks []Webhook, payload WebhookPayload) map[int64]DeliveryResult {
	results := make(map[int64]DeliveryResult)
	resultCh := make(chan struct {
		webhookID int64
		result    DeliveryResult
	}, len(webhooks))

	// Launch goroutines for parallel delivery
	for _, webhook := range webhooks {
		go func(wh Webhook) {
			result := d.DeliverWithResult(ctx, wh, payload)
			resultCh <- struct {
				webhookID int64
				result    DeliveryResult
			}{wh.ID, result}
		}(webhook)
	}

	// Collect results
	for i := 0; i < len(webhooks); i++ {
		res := <-resultCh
		results[res.webhookID] = res.result
	}

	return results
}

// RetryPolicy defines the retry behavior
type RetryPolicy struct {
	MaxRetries int
	Delays     []time.Duration
}

// DefaultRetryPolicy returns the default retry policy
// Retry: 3 attempts with backoff (10s, 60s, 300s)
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		Delays: []time.Duration{
			10 * time.Second,
			60 * time.Second,
			300 * time.Second,
		},
	}
}
