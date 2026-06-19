package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDKey is the context key for storing the request ID
type contextKey string

const RequestIDKey contextKey = "x-request-id"

// RequestIDMiddleware is a middleware that extracts or generates the X-Request-ID header
// and makes it available through the request context.
// T028.5: Request ID middleware for tracing support (C14)
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get request ID from header, otherwise generate a new UUID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store request ID in context for downstream handlers
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Continue to next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestIDFromContext retrieves the request ID from the context
func GetRequestIDFromContext(ctx context.Context) string {
	if rid := ctx.Value(RequestIDKey); rid != nil {
		if id, ok := rid.(string); ok {
			return id
		}
	}
	return uuid.New().String()
}
