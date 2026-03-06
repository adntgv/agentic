# Webhook Package

Handles webhook delivery with HMAC-SHA256 signing, retry logic, and failure handling.

## Overview

The webhook delivery component sends HTTP POST requests to webhook endpoints with proper authentication and retry logic per v0.1.5 spec.

## Components

### Delivery (`delivery.go`)

Sends webhook payloads with HMAC signing and automatic retries.

**Features:**
- HMAC-SHA256 signature generation
- `X-Marketplace-Signature` header
- 10s timeout per delivery attempt
- 3 retry attempts with exponential backoff
- Support for 4xx vs 5xx error handling (no retry on 4xx except 429)
- Batch delivery to multiple endpoints
- Detailed delivery result tracking

**Usage:**
```go
delivery := webhook.NewDelivery(
    10*time.Second, // timeout
    3,              // max retries
)

webhook := webhook.Webhook{
    ID:       1,
    URL:      "https://example.com/webhook",
    Secret:   "webhook_secret_key",
    Active:   true,
}

payload := webhook.WebhookPayload{
    Event:            "task.published",
    IdempotencyKey:   "outbox-12345",
    SequenceNumber:   12345,
    Timestamp:        time.Now(),
    Data:             map[string]interface{}{"task_id": "..."},
}

err := delivery.Deliver(ctx, webhook, payload)
if err != nil {
    log.Printf("Delivery failed: %v", err)
}
```

## HMAC Signature

### Generation (Server Side)
```go
func computeSignature(body []byte, secret string) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}
```

### Verification (Client Side)
Webhook recipients must verify the signature:

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func verifyWebhook(body []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expectedSignature := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
```

Example webhook handler:
```go
func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Marketplace-Signature")
    
    if !verifyWebhook(body, signature, secret) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    var payload webhook.WebhookPayload
    json.Unmarshal(body, &payload)
    
    // Process event...
    
    w.WriteHeader(http.StatusOK)
}
```

## HTTP Headers

Every webhook delivery includes:
- `Content-Type: application/json`
- `X-Marketplace-Signature: <hmac-sha256-hex>` - HMAC signature for verification
- `X-Marketplace-Event: <event-type>` - Event type (e.g., "task.published")
- `X-Marketplace-Idempotency-Key: <key>` - Unique key for deduplication
- `X-Marketplace-Timestamp: <rfc3339>` - Event timestamp
- `User-Agent: Agentic-Marketplace-Webhook/v0.1.5`

## Retry Policy

### Default Policy
- **Attempt 1**: Immediate
- **Attempt 2**: 1 second delay (2^0 = 1s)
- **Attempt 3**: 2 second delay (2^1 = 2s)
- **Attempt 4**: 4 second delay (2^2 = 4s)

Total: 3 retries with exponential backoff

### Error Handling
- **2xx**: Success, no retry
- **4xx** (except 429): Client error, no retry (bad request, auth failure, etc.)
- **429 Too Many Requests**: Retry with backoff
- **5xx**: Server error, retry with backoff
- **Network errors**: Retry with backoff

## Payload Format

```json
{
  "event": "task.published",
  "idempotency_key": "outbox-12345",
  "event_sequence_number": 12345,
  "timestamp": "2026-03-06T12:00:00Z",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Build a marketplace",
    "budget": "1000.00",
    "poster_worker_id": "..."
  }
}
```

## Failure Handling

### Consecutive Failures
- Track `fail_count` in webhooks table
- After **50 consecutive failures**: auto-disable webhook
- Notify operator via notification system

### Dead Letter Queue
- After max retries: mark event as `failed` in outbox
- Move to dead letter queue for manual review
- Alert admin for investigation

## Batch Delivery

Send to multiple webhooks in parallel:

```go
webhooks := []webhook.Webhook{...}
results := delivery.BatchDeliver(ctx, webhooks, payload)

for webhookID, result := range results {
    if !result.Success {
        log.Printf("Webhook %d failed: %v", webhookID, result.Error)
    }
}
```

## Performance

### HTTP Client Tuning
- **Timeout**: 10s per request
- **Max Idle Connections**: 100
- **Max Idle Connections Per Host**: 10
- **Idle Connection Timeout**: 90s

### Concurrency
- Batch delivery uses goroutines for parallel delivery
- Each webhook delivery is independent
- No global rate limiting (per-webhook limits handled by retry backoff)

## Monitoring

Track these metrics:
- Delivery success rate
- Average latency per webhook
- Retry rate
- Failed webhooks
- Disabled webhooks (50+ failures)

## Security Best Practices

### Server Side (Sending)
1. Generate strong webhook secrets (32+ chars, random)
2. Always sign payloads with HMAC-SHA256
3. Use HTTPS only for webhook URLs
4. Include timestamp to prevent replay attacks (check age < 5 minutes)
5. Store secrets encrypted in database

### Client Side (Receiving)
1. **Always verify signature** before processing
2. Check timestamp freshness (reject old events)
3. Use idempotency key to prevent duplicate processing
4. Respond 200 OK quickly (process async if needed)
5. Log failed signature attempts (potential attack)

### Example Secure Handler
```go
func secureWebhookHandler(w http.ResponseWriter, r *http.Request) {
    // Read body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Invalid body", http.StatusBadRequest)
        return
    }
    
    // Verify signature
    signature := r.Header.Get("X-Marketplace-Signature")
    if !verifyWebhook(body, signature, secret) {
        logSecurityEvent("invalid_signature", r.RemoteAddr)
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Parse payload
    var payload webhook.WebhookPayload
    if err := json.Unmarshal(body, &payload); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Check timestamp (prevent replay)
    age := time.Since(payload.Timestamp)
    if age > 5*time.Minute {
        http.Error(w, "Event too old", http.StatusBadRequest)
        return
    }
    
    // Check idempotency (prevent duplicates)
    if isProcessed(payload.IdempotencyKey) {
        w.WriteHeader(http.StatusOK) // Already processed
        return
    }
    
    // Queue for async processing
    queue.Push(payload)
    w.WriteHeader(http.StatusAccepted)
}
```

## Configuration

Environment variables:
- `WEBHOOK_TIMEOUT` - Default: 10s
- `WEBHOOK_MAX_RETRIES` - Default: 3
- `WEBHOOK_FAILURE_THRESHOLD` - Default: 50

## Integration with Outbox

The dispatcher uses this delivery component:

```go
// In outbox/dispatcher.go
deliveryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
err := d.delivery.Deliver(deliveryCtx, webhook, payload)
cancel()

if err != nil {
    // Handle failure...
}
```
