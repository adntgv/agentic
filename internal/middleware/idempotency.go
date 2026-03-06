package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// Idempotency middleware checks Idempotency-Key header and returns cached response on duplicate
func Idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to write methods
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
			next.ServeHTTP(w, r)
			return
		}

		idempotencyKey := r.Header.Get("Idempotency-Key")
		if idempotencyKey == "" {
			// No idempotency key provided, proceed normally
			next.ServeHTTP(w, r)
			return
		}

		// Get worker_id from context
		workerID, ok := GetWorkerID(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Normalize endpoint: "{METHOD} {ROUTE_TEMPLATE}"
		// For now, use full path as placeholder (proper normalization requires router info)
		endpoint := r.Method + " " + r.URL.Path

		// TODO: Check idempotency_keys table in DB
		// If exists: return cached response (status code + body)
		// If not: proceed, capture response, store in DB with 24h TTL

		// Placeholder: just proceed for now
		next.ServeHTTP(w, r)

		// In production:
		// 1. Query: SELECT response_status, response_body FROM idempotency_keys WHERE worker_id = ? AND endpoint = ? AND key = ?
		// 2. If found: w.WriteHeader(status), w.Write(body), return
		// 3. If not: wrap response writer to capture status + body, proceed, then INSERT into idempotency_keys

		_ = workerID
		_ = endpoint
		_ = idempotencyKey
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and body
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}
