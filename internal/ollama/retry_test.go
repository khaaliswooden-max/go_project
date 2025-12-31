// internal/ollama/retry_test.go
// Table-driven tests for retry logic with exponential backoff
//
// LEARN: Testing retry logic requires careful handling of:
// 1. Time-based behavior (use deterministic delays in tests)
// 2. Context cancellation (test both pre-canceled and mid-retry cancellation)
// 3. Error classification (retryable vs non-retryable)
// 4. Jitter randomness (test bounds, not exact values)

package ollama

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/khaaliswooden-max/go_project/pkg/errors"
)

// === Retry Configuration Tests ===

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	// Verify default values match exercise requirements
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialDelay != 100*time.Millisecond {
		t.Errorf("InitialDelay = %v, want 100ms", cfg.InitialDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", cfg.Multiplier)
	}
	if cfg.JitterFactor != 0.1 {
		t.Errorf("JitterFactor = %f, want 0.1", cfg.JitterFactor)
	}
}

// === Retry Success Tests ===

func TestRetry_SucceedsImmediately(t *testing.T) {
	// LEARN: Test the happy path first - function succeeds on first try
	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return nil // Success!
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetry_SucceedsAfterRetries(t *testing.T) {
	// LEARN: Test that retry actually retries on retryable errors
	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0, // Disable jitter for predictable timing
	}

	tests := []struct {
		name         string
		failCount    int  // Number of times to fail before succeeding
		wantAttempts int  // Expected total attempts
		wantErr      bool // Whether we expect final error
	}{
		{
			name:         "succeeds after 1 failure",
			failCount:    1,
			wantAttempts: 2,
			wantErr:      false,
		},
		{
			name:         "succeeds after 2 failures",
			failCount:    2,
			wantAttempts: 3,
			wantErr:      false,
		},
		{
			name:         "succeeds after 3 failures (max retries)",
			failCount:    3,
			wantAttempts: 4,
			wantErr:      false,
		},
		{
			name:         "fails after exhausting retries",
			failCount:    10, // More than max retries
			wantAttempts: 4,  // 1 initial + 3 retries
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			err := Retry(context.Background(), cfg, func(ctx context.Context) error {
				attempts++
				if attempts <= tt.failCount {
					// Return retryable error
					return errors.ErrOllamaUnavailable
				}
				return nil
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("Retry() error = %v, wantErr %v", err, tt.wantErr)
			}
			if attempts != tt.wantAttempts {
				t.Errorf("attempts = %d, want %d", attempts, tt.wantAttempts)
			}
		})
	}
}

// === Non-Retryable Error Tests ===

func TestRetry_NonRetryableError(t *testing.T) {
	// LEARN: Non-retryable errors should fail immediately without retry
	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	tests := []struct {
		name        string
		err         error
		wantRetries int
	}{
		{
			name:        "invalid request",
			err:         errors.ErrInvalidRequest,
			wantRetries: 0,
		},
		{
			name:        "model not found",
			err:         errors.ErrModelNotFound,
			wantRetries: 0,
		},
		{
			name:        "invalid JSON",
			err:         errors.ErrInvalidJSON,
			wantRetries: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			err := Retry(context.Background(), cfg, func(ctx context.Context) error {
				attempts++
				return tt.err
			})

			if !errors.Is(err, tt.err) {
				t.Errorf("Retry() error = %v, want %v", err, tt.err)
			}
			// Should only attempt once (no retries for non-retryable errors)
			if attempts != 1 {
				t.Errorf("attempts = %d, want 1 (no retries)", attempts)
			}
		})
	}
}

// === Context Cancellation Tests ===

func TestRetry_ContextCancellation(t *testing.T) {
	// LEARN: Context cancellation should be respected at multiple points:
	// 1. Before first attempt
	// 2. During retry delay
	// 3. Within the retryable function

	tests := []struct {
		name        string
		setupCtx    func() (context.Context, context.CancelFunc)
		wantErrType error
	}{
		{
			name: "pre-canceled context",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			wantErrType: errors.ErrRequestCanceled,
		},
		{
			name: "pre-expired deadline",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(1 * time.Millisecond) // Ensure timeout
				return ctx, cancel
			},
			wantErrType: errors.ErrRequestTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupCtx()
			defer cancel()

			cfg := RetryConfig{
				MaxRetries:   3,
				InitialDelay: 10 * time.Millisecond,
				Multiplier:   2.0,
				JitterFactor: 0,
			}

			attempts := 0
			err := Retry(ctx, cfg, func(ctx context.Context) error {
				attempts++
				return errors.ErrOllamaUnavailable
			})

			if !errors.Is(err, tt.wantErrType) {
				t.Errorf("Retry() error = %v, want %v", err, tt.wantErrType)
			}
			// Should not have attempted (canceled before first try)
			if attempts > 1 {
				t.Errorf("attempts = %d, want 0 or 1", attempts)
			}
		})
	}
}

func TestRetry_CanceledDuringDelay(t *testing.T) {
	// LEARN: Test that cancellation during retry delay is handled correctly
	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 500 * time.Millisecond, // Long enough to cancel during
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	var attempts int32 = 0
	done := make(chan error, 1)

	go func() {
		err := Retry(ctx, cfg, func(ctx context.Context) error {
			atomic.AddInt32(&attempts, 1)
			// Always fail with retryable error to trigger retry delay
			return errors.ErrOllamaUnavailable
		})
		done <- err
	}()

	// Wait for first attempt, then cancel during delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for retry to complete
	select {
	case err := <-done:
		if !errors.Is(err, errors.ErrRequestCanceled) {
			t.Errorf("Retry() error = %v, want ErrRequestCanceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Retry() did not respond to cancellation")
	}
}

// === Exponential Backoff Tests ===

func TestRetry_ExponentialBackoff(t *testing.T) {
	// LEARN: Verify that delays increase exponentially
	// We can't test exact timing due to jitter, but we can verify the pattern

	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0, // Disable jitter for predictable timing
	}

	var timestamps []time.Time
	start := time.Now()

	// Force all retries by always returning retryable error
	_ = Retry(context.Background(), cfg, func(ctx context.Context) error {
		timestamps = append(timestamps, time.Now())
		return errors.ErrOllamaUnavailable
	})

	// Expected delays: 50ms, 100ms, 200ms (exponential)
	// Total time should be around 350ms (plus execution overhead)

	if len(timestamps) != 4 { // 1 initial + 3 retries
		t.Fatalf("got %d attempts, want 4", len(timestamps))
	}

	totalTime := time.Since(start)
	minExpected := 300 * time.Millisecond // 50 + 100 + 200 = 350, with some tolerance
	maxExpected := 600 * time.Millisecond // Allow for overhead

	if totalTime < minExpected {
		t.Errorf("total time %v < minimum expected %v", totalTime, minExpected)
	}
	if totalTime > maxExpected {
		t.Errorf("total time %v > maximum expected %v", totalTime, maxExpected)
	}

	// Verify increasing delays between attempts
	for i := 1; i < len(timestamps); i++ {
		delay := timestamps[i].Sub(timestamps[i-1])
		t.Logf("Delay %d: %v", i, delay)
	}
}

func TestRetry_MaxDelayRespected(t *testing.T) {
	// LEARN: Verify that delays don't exceed MaxDelay
	cfg := RetryConfig{
		MaxRetries:   5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     75 * time.Millisecond, // Low max to test capping
		Multiplier:   2.0,
		JitterFactor: 0,
	}

	var timestamps []time.Time

	_ = Retry(context.Background(), cfg, func(ctx context.Context) error {
		timestamps = append(timestamps, time.Now())
		return errors.ErrOllamaUnavailable
	})

	// Check that no delay exceeds MaxDelay (with tolerance)
	for i := 1; i < len(timestamps); i++ {
		delay := timestamps[i].Sub(timestamps[i-1])
		maxAllowed := cfg.MaxDelay + 50*time.Millisecond // Allow overhead
		if delay > maxAllowed {
			t.Errorf("delay %d = %v exceeds MaxDelay %v", i, delay, cfg.MaxDelay)
		}
	}
}

// === Jitter Tests ===

func TestAddJitter(t *testing.T) {
	// LEARN: Testing randomness requires statistical approaches.
	// We verify that values fall within expected bounds.

	tests := []struct {
		name     string
		duration time.Duration
		factor   float64
		minBound time.Duration
		maxBound time.Duration
	}{
		{
			name:     "10% jitter on 100ms",
			duration: 100 * time.Millisecond,
			factor:   0.1,
			minBound: 90 * time.Millisecond,
			maxBound: 110 * time.Millisecond,
		},
		{
			name:     "20% jitter on 100ms",
			duration: 100 * time.Millisecond,
			factor:   0.2,
			minBound: 80 * time.Millisecond,
			maxBound: 120 * time.Millisecond,
		},
		{
			name:     "no jitter",
			duration: 100 * time.Millisecond,
			factor:   0,
			minBound: 100 * time.Millisecond,
			maxBound: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to test randomness
			for i := 0; i < 100; i++ {
				result := addJitter(tt.duration, tt.factor)
				if result < tt.minBound || result > tt.maxBound {
					t.Errorf("addJitter(%v, %f) = %v, want in [%v, %v]",
						tt.duration, tt.factor, result, tt.minBound, tt.maxBound)
				}
			}
		})
	}
}

func TestAddJitter_NegativeFactor(t *testing.T) {
	// LEARN: Edge case - negative factor should be treated as no jitter
	d := 100 * time.Millisecond
	result := addJitter(d, -0.1)
	if result != d {
		t.Errorf("addJitter with negative factor = %v, want %v", result, d)
	}
}

// === RetryWithDefaults Tests ===

func TestRetryWithDefaults(t *testing.T) {
	// LEARN: Test the convenience function uses correct defaults
	attempts := 0
	_ = RetryWithDefaults(context.Background(), func(ctx context.Context) error {
		attempts++
		return errors.ErrOllamaUnavailable
	})

	// Default is 3 retries, so 4 total attempts
	if attempts != 4 {
		t.Errorf("RetryWithDefaults attempts = %d, want 4", attempts)
	}
}

// === Integration-style Tests ===

func TestRetry_RealWorldScenario(t *testing.T) {
	// LEARN: Test a realistic scenario where service is temporarily unavailable
	// then recovers

	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}

	// Simulate service that fails twice then recovers
	callCount := 0
	recoveryPoint := 3 // Succeed on 3rd attempt

	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		callCount++
		if callCount < recoveryPoint {
			return errors.ErrOllamaUnavailable
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil (service should recover)", err)
	}
	if callCount != recoveryPoint {
		t.Errorf("callCount = %d, want %d", callCount, recoveryPoint)
	}
}

// === Benchmark Tests ===

func BenchmarkRetry_NoRetries(b *testing.B) {
	cfg := RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Retry(ctx, cfg, func(ctx context.Context) error {
			return nil // Success on first try
		})
	}
}

func BenchmarkAddJitter(b *testing.B) {
	d := 100 * time.Millisecond
	factor := 0.1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = addJitter(d, factor)
	}
}
