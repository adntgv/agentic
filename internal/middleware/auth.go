package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	WorkerIDKey   contextKey = "worker_id"
	WorkerTypeKey contextKey = "worker_type"
	ScopesKey     contextKey = "scopes"
)

// Auth middleware extracts authentication from Bearer token or X-API-Key header
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Bearer token (JWT from SIWE)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			_ = strings.TrimPrefix(authHeader, "Bearer ") // token extracted but not yet verified (TODO)
			// TODO: Verify JWT token, extract worker_id, worker_type, scopes
			// For now, placeholder:
			workerID := uuid.New()
			ctx := context.WithValue(r.Context(), WorkerIDKey, workerID)
			ctx = context.WithValue(ctx, WorkerTypeKey, "user")
			ctx = context.WithValue(ctx, ScopesKey, []string{"all"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Check for X-API-Key header (agent API key)
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			// TODO: Verify API key, get worker_id, worker_type, scopes
			workerID := uuid.New()
			ctx := context.WithValue(r.Context(), WorkerIDKey, workerID)
			ctx = context.WithValue(ctx, WorkerTypeKey, "agent")
			ctx = context.WithValue(ctx, ScopesKey, []string{"bid", "ack", "submit", "message", "dispute"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// No valid auth
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// GetWorkerID extracts worker_id from request context
func GetWorkerID(ctx context.Context) (uuid.UUID, bool) {
	workerID, ok := ctx.Value(WorkerIDKey).(uuid.UUID)
	return workerID, ok
}

// GetWorkerType extracts worker_type from request context
func GetWorkerType(ctx context.Context) (string, bool) {
	workerType, ok := ctx.Value(WorkerTypeKey).(string)
	return workerType, ok
}

// GetScopes extracts scopes from request context
func GetScopes(ctx context.Context) ([]string, bool) {
	scopes, ok := ctx.Value(ScopesKey).([]string)
	return scopes, ok
}
