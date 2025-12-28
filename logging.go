// internal/middleware/logging.go
// Exercise 1.2: Logging Middleware
//
// TASK: Implement HTTP middleware that logs request/response details
// with correlation IDs for request tracing.
//
// Requirements:
// 1. Log: method, path, status code, duration, correlation ID
// 2. Generate correlation ID if not in X-Correlation-ID header
// 3. Add correlation ID to response headers
// 4. Use structured JSON logging
//
// Run tests with: make verify-1-2

package middleware

import (
	"net/http"
)

// TODO: Implement LoggingMiddleware
//
// Hints:
// - Use a uuid library or crypto/rand for correlation IDs
// - Wrap http.ResponseWriter to capture status code
// - Use time.Since() for duration
// - Consider using slog for structured logging

// LoggingMiddleware logs HTTP request details.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		// 1. Check for existing correlation ID in header
		// 2. Generate new one if missing
		// 3. Add to response header
		// 4. Record start time
		// 5. Wrap ResponseWriter to capture status
		// 6. Call next handler
		// 7. Log details after response

		next.ServeHTTP(w, r)
	})
}

// correlationIDHeader is the standard header for request correlation.
const correlationIDHeader = "X-Correlation-ID"

// generateCorrelationID creates a unique request identifier.
// TODO: Implement using crypto/rand or uuid
func generateCorrelationID() string {
	return "TODO"
}
