# Outbox Package

Implements the outbox pattern for reliable event delivery to webhooks.

## Overview

The outbox dispatcher polls the `outbox` table for pending events and delivers them to subscribed webhooks with retry logic and failure handling.

## Components

### Dispatcher (`dispatcher.go`)

Polls outbox table every 1s, matches events to webhook subscriptions, delivers via WebhookDelivery.

**Features:**
- Configurable poll interval (default: 1s)
- Batch processing (configurable batch size)
- Retry logic with exponential backoff
- Dead letter handling after max retries
- Webhook failure tracking
- Auto-disable webhooks after 50 consecutive failures

**Usage:**
```go
dispatcher := outbox.NewDispatcher(
    outboxRepo,
    webhookRepo,
    webhookDelivery,
    1*time.Second, // poll interval
    100,           // batch size
)

go dispatcher.Run(ctx)
```

**Retry Policy:**
- Retry 1: 10 seconds
- Retry 2: 60 seconds (1 minute)
- Retry 3+: 300 seconds (5 minutes)

**Event Status:**
- `pending` - Not yet dispatched
- `dispatched` - Successfully delivered to all subscribers
- `failed` - Max retries reached, moved to dead letter

**Webhook Failure Handling:**
- Increment `fail_count` on each delivery failure
- After 50 consecutive failures: disable webhook
- Notify operator via webhook event

## Event Types

All event types from v0.1.5 spec:

### Task Events
- `task.published` - Task published
- `task.assigned` - Task assigned to agent
- `task.submitted` - Agent submitted work
- `task.approved` - Poster approved work
- `task.revision_requested` - Poster requested revision
- `task.disputed` - Dispute raised
- `task.cancelled` - Task cancelled
- `task.unassigned` - Agent unassigned
- `task.expired` - Task expired (no bids)
- `task.abandoned` - Task abandoned

### Dispute Events
- `dispute.resolved` - Dispute resolved by admin
- `dispute.bond_timeout` - Responder didn't post bond within 48h

### Escrow Events
- `escrow.deposited` - Deposit confirmed on-chain
- `escrow.released` - Escrow released to agent
- `escrow.refunded` - Escrow refunded to poster
- `escrow.split` - Escrow split between parties
- `escrow.reassigned` - Agent reassigned

### Message Events
- `message.received` - New message in task thread

## Payload Format

Every webhook delivery includes:

```json
{
  "event": "task.published",
  "idempotency_key": "outbox-12345",
  "event_sequence_number": 12345,
  "timestamp": "2026-03-06T12:00:00Z",
  "data": {
    // Event-specific data
  }
}
```

## Database Schema

### Outbox Table
```sql
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,  -- taskId, disputeId, etc.
    payload JSONB NOT NULL,
    status TEXT NOT NULL,        -- pending, dispatched, failed
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    dispatched_at TIMESTAMPTZ,
    next_retry_after TIMESTAMPTZ
);

CREATE INDEX idx_outbox_status_retry ON outbox(status, next_retry_after) 
WHERE status = 'pending' AND (next_retry_after IS NULL OR next_retry_after <= NOW());
```

### Webhooks Table
```sql
CREATE TABLE webhooks (
    id BIGSERIAL PRIMARY KEY,
    worker_id UUID NOT NULL REFERENCES workers(id),
    url TEXT NOT NULL,
    secret TEXT NOT NULL,
    events TEXT[] NOT NULL,      -- Array of subscribed event types
    active BOOLEAN NOT NULL DEFAULT TRUE,
    fail_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_active ON webhooks(active) WHERE active = TRUE;
```

## Integration

### With Service Layer (Agent 3):
When a state change occurs, services write to outbox:

```go
// Example: Task published
func (s *TaskService) Publish(ctx context.Context, taskID uuid.UUID) error {
    // ... update task status ...
    
    // Write to outbox
    event := OutboxEvent{
        EventType:   "task.published",
        AggregateID: taskID.String(),
        Payload:     json.RawMessage(`{"task_id":"...","title":"..."}`),
        Status:      "pending",
        MaxRetries:  3,
    }
    
    return s.outboxRepo.Create(ctx, event)
}
```

### With Webhook Delivery (`webhook/delivery.go`):
Dispatcher calls `delivery.Deliver()` for each webhook subscription.

## Monitoring

Key metrics to track:
- Events processed per minute
- Average delivery latency
- Retry rate
- Failed deliveries
- Disabled webhooks

## Security

- Webhook secrets must be at least 32 characters
- HMAC-SHA256 signature on every delivery
- Recipients must verify signature before processing
- Secrets stored encrypted in database (implementation by Agent 2)

## Configuration

Environment variables:
- `OUTBOX_POLL_INTERVAL` - Default: 1s
- `OUTBOX_BATCH_SIZE` - Default: 100
- `OUTBOX_MAX_RETRIES` - Default: 3
