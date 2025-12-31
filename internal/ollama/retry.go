// internal/ollama/retry.go
// Retry logic with exponential backoff for Ollama API calls
//
// LEARN: Exponential backoff is a standard pattern for handling transient failures.
// Key components:
// 1. Initial delay that doubles with each retry
// 2. Maximum retry count to prevent infinite loops
// 3. Jitter to prevent thundering herd problem
// 4. Context cancellation for graceful shutdown
//
// References:
// - AWS Exponential Backoff: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
// - Google Cloud Retry Strategy: https://cloud.google.com/storage/docs/retry-strategy

package ollama

import (
	"context"
	"math/rand"
	"time"

	"github.com/khaaliswooden-max/go_project/pkg/errors"
	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// RetryConfig holds retry configuration.
//
// LEARN: Configuration structs make defaults explicit and allow customization
// without changing function signatures.
type RetryConfig struct {
	MaxRetries   int           // Maximum number of retry attempts (default: 3)
	InitialDelay time.Duration // Initial delay before first retry (default: 100ms)
	MaxDelay     time.Duration // Maximum delay between retries (default: 10s)
	Multiplier   float64       // Backoff multiplier (default: 2.0)
	JitterFactor float64       // Jitter factor ±percentage (default: 0.1 for ±10%)
}

// DefaultRetryConfig returns sensible defaults for retry configuration.
//
// LEARN: Provide sensible defaults that work for most use cases.
// These values are based on industry best practices.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.1, // ±10%
	}
}

// RetryableFunc is a function that can be retried.
// It returns an error that will be checked for retryability.
type RetryableFunc func(ctx context.Context) error

// Retry executes the given function with exponential backoff.
//
// LEARN: This pattern separates the retry logic from the business logic.
// The caller provides a function that performs the actual work, and
// this function handles all retry mechanics.
//
// The function will:
// 1. Execute the provided function
// 2. If it succeeds, return nil
// 3. If it fails with a retryable error, wait with exponential backoff and retry
// 4. If it fails with a non-retryable error, return immediately
// 5. If context is canceled, return context error
// 6. After max retries, return the last error
func Retry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// LEARN: Always check context before expensive operations.
		// This ensures we respect cancellation even before the first attempt.
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return errors.ErrRequestCanceled
			}
			return errors.ErrRequestTimeout
		default:
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			return nil // Success!
		}

		lastErr = err

		// Check if error is retryable
		// LEARN: Not all errors should be retried. Network errors and timeouts
		// are usually transient, but validation errors are permanent.
		if !errors.IsRetryable(err) {
			return err // Non-retryable error, fail fast
		}

		// Check if we've exhausted retries
		if attempt >= cfg.MaxRetries {
			break
		}

		// Calculate delay with jitter
		// LEARN: Jitter prevents the "thundering herd" problem where many
		// clients retry simultaneously and overwhelm the server.
		jitteredDelay := addJitter(delay, cfg.JitterFactor)

		// Cap the delay at MaxDelay
		if jitteredDelay > cfg.MaxDelay {
			jitteredDelay = cfg.MaxDelay
		}

		// LEARN: Use select with time.After and ctx.Done() for cancellable delays.
		// This pattern ensures we can respond to cancellation even while waiting.
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return errors.ErrRequestCanceled
			}
			return errors.ErrRequestTimeout
		case <-time.After(jitteredDelay):
			// Continue to next retry
		}

		// Increase delay for next attempt (exponential backoff)
		delay = time.Duration(float64(delay) * cfg.Multiplier)
	}

	return lastErr
}

// RetryWithDefaults executes the given function with default retry configuration.
//
// LEARN: Convenience functions that use defaults make the common case easy
// while still allowing customization through the full Retry function.
func RetryWithDefaults(ctx context.Context, fn RetryableFunc) error {
	return Retry(ctx, DefaultRetryConfig(), fn)
}

// addJitter adds random jitter to a duration.
//
// LEARN: Jitter is typically ±percentage of the base delay.
// This spreads out retry attempts across time.
func addJitter(d time.Duration, factor float64) time.Duration {
	if factor <= 0 {
		return d
	}

	// Calculate jitter range: ±factor * d
	// For factor=0.1 (10%), jitter ranges from -10% to +10%
	jitterRange := float64(d) * factor

	// Random value in range [-jitterRange, +jitterRange]
	// LEARN: Using rand.Float64() gives [0, 1), multiply by 2 and subtract 1
	// gives [-1, 1), then multiply by range.
	jitter := (rand.Float64()*2 - 1) * jitterRange

	result := time.Duration(float64(d) + jitter)

	// Ensure we never return a negative duration
	if result < 0 {
		return 0
	}

	return result
}

// === Retry-enabled Client Methods ===
// LEARN: These methods wrap the base client methods with retry logic.
// They demonstrate how to compose retry with existing functionality.

// GenerateWithRetry performs a text generation with automatic retry.
func (c *client) GenerateWithRetry(ctx context.Context, req types.GenerateRequest, cfg RetryConfig) (*types.GenerateResponse, error) {
	var resp *types.GenerateResponse
	var genErr error

	err := Retry(ctx, cfg, func(ctx context.Context) error {
		resp, genErr = c.Generate(ctx, req)
		return genErr
	})

	return resp, err
}

// ChatWithRetry performs a chat completion with automatic retry.
func (c *client) ChatWithRetry(ctx context.Context, req types.ChatRequest, cfg RetryConfig) (*types.ChatResponse, error) {
	var resp *types.ChatResponse
	var chatErr error

	err := Retry(ctx, cfg, func(ctx context.Context) error {
		resp, chatErr = c.Chat(ctx, req)
		return chatErr
	})

	return resp, err
}
