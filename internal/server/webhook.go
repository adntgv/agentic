package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	maxRetries    = 3
	webhookTimeout = 10 * time.Second
)

// deliverWithRetry sends a webhook payload with exponential backoff retries
func deliverWithRetry(url, secret string, payload WebhookPayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook: failed to marshal payload: %v", err)
		return
	}

	backoff := 1 * time.Second
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 5 // 1s, 5s, 25s
		}

		if err := sendWebhook(url, secret, body); err != nil {
			log.Printf("webhook: attempt %d/%d to %s failed: %v", attempt+1, maxRetries+1, url, err)
			continue
		}

		log.Printf("webhook: delivered %s to %s", payload.Event, url)
		return
	}

	log.Printf("webhook: exhausted retries for %s to %s", payload.Event, url)
}

func sendWebhook(url, secret string, body []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Agentic-Webhook/1.0")

	// HMAC-SHA256 signature
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", fmt.Sprintf("sha256=%s", sig))
	}

	client := &http.Client{Timeout: webhookTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
