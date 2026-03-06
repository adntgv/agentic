package middleware

import (
	"net/http"
	"sync"
	"time"
)

// Simple in-memory rate limiter (for production, use Redis)
var (
	rateLimitStore = make(map[string]*rateLimitEntry)
	rateLimitMutex sync.RWMutex
)

type rateLimitEntry struct {
	count      int
	resetAt    time.Time
}

// RateLimit middleware applies basic in-memory rate limiting
func RateLimit(requestsPerHour int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workerID, ok := GetWorkerID(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			key := workerID.String()
			rateLimitMutex.Lock()
			defer rateLimitMutex.Unlock()

			now := time.Now()
			entry, exists := rateLimitStore[key]
			if !exists || now.After(entry.resetAt) {
				// New or expired entry
				rateLimitStore[key] = &rateLimitEntry{
					count:   1,
					resetAt: now.Add(1 * time.Hour),
				}
				next.ServeHTTP(w, r)
				return
			}

			if entry.count >= requestsPerHour {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			entry.count++
			next.ServeHTTP(w, r)
		})
	}
}
