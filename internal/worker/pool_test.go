// internal/worker/pool_test.go
// Tests for worker pool implementation
//
// LEARN: Testing concurrent code requires:
// 1. Race detector: go test -race
// 2. Timeouts: Prevent tests from hanging forever
// 3. Synchronization: Properly wait for goroutines
// 4. Deterministic assertions: Avoid timing-dependent tests

package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// === Configuration Tests ===

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Workers <= 0 {
		t.Errorf("Workers = %d, want > 0", cfg.Workers)
	}
	if cfg.JobBuffer <= 0 {
		t.Errorf("JobBuffer = %d, want > 0", cfg.JobBuffer)
	}
	if cfg.ResultBuffer <= 0 {
		t.Errorf("ResultBuffer = %d, want > 0", cfg.ResultBuffer)
	}
}

func TestNew_WithConfig(t *testing.T) {
	cfg := Config{
		Workers:      2,
		JobBuffer:    4,
		ResultBuffer: 4,
	}

	pool := New(cfg)
	defer pool.Shutdown()

	if pool.WorkerCount() != 2 {
		t.Errorf("WorkerCount() = %d, want 2", pool.WorkerCount())
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	// LEARN: Test edge cases - invalid config should use defaults
	cfg := Config{
		Workers:      -1,
		JobBuffer:    0,
		ResultBuffer: 0,
	}

	pool := New(cfg)
	defer pool.Shutdown()

	// Should use defaults
	if pool.WorkerCount() <= 0 {
		t.Errorf("WorkerCount() = %d, want > 0", pool.WorkerCount())
	}
}

// === Basic Functionality Tests ===

func TestPool_SingleJob(t *testing.T) {
	pool := NewDefault()
	defer pool.Shutdown()

	// Submit a simple job
	expected := 42
	pool.Submit(func(ctx context.Context) (any, error) {
		return expected, nil
	})

	// Wait for result with timeout
	select {
	case result := <-pool.Results():
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Value != expected {
			t.Errorf("Value = %v, want %v", result.Value, expected)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestPool_MultipleJobs(t *testing.T) {
	pool := NewDefault()

	numJobs := 10
	for i := 0; i < numJobs; i++ {
		i := i // Capture loop variable
		pool.SubmitWithID(func(ctx context.Context) (any, error) {
			return i * 2, nil
		}, i)
	}

	// Collect results
	results := make(map[int]int)
	for i := 0; i < numJobs; i++ {
		select {
		case result := <-pool.Results():
			if result.Err != nil {
				t.Errorf("job %d error: %v", result.JobID, result.Err)
			}
			results[result.JobID] = result.Value.(int)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for result %d", i)
		}
	}

	pool.Shutdown()

	// Verify all results (order may vary due to concurrency)
	for i := 0; i < numJobs; i++ {
		expected := i * 2
		if got := results[i]; got != expected {
			t.Errorf("job %d: got %d, want %d", i, got, expected)
		}
	}
}

func TestPool_JobError(t *testing.T) {
	pool := NewDefault()
	defer pool.Shutdown()

	expectedErr := errors.New("job failed")
	pool.Submit(func(ctx context.Context) (any, error) {
		return nil, expectedErr
	})

	select {
	case result := <-pool.Results():
		if result.Err == nil {
			t.Error("expected error, got nil")
		}
		if result.Err.Error() != expectedErr.Error() {
			t.Errorf("error = %v, want %v", result.Err, expectedErr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

// === Concurrency Tests ===

func TestPool_ConcurrentExecution(t *testing.T) {
	// LEARN: Verify that jobs actually run in parallel
	pool := New(Config{Workers: 4, JobBuffer: 8, ResultBuffer: 8})

	var running int32
	var maxRunning int32

	numJobs := 10
	for i := 0; i < numJobs; i++ {
		pool.Submit(func(ctx context.Context) (any, error) {
			// Atomically track concurrent jobs
			current := atomic.AddInt32(&running, 1)
			defer atomic.AddInt32(&running, -1)

			// Track maximum concurrency
			for {
				max := atomic.LoadInt32(&maxRunning)
				if current <= max || atomic.CompareAndSwapInt32(&maxRunning, max, current) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond) // Simulate work
			return current, nil
		})
	}

	// Drain results
	for i := 0; i < numJobs; i++ {
		select {
		case <-pool.Results():
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	pool.Shutdown()

	// With 4 workers and 50ms sleep, we should see some parallelism
	if maxRunning < 2 {
		t.Errorf("maxRunning = %d, expected at least 2 for parallel execution", maxRunning)
	}
	t.Logf("Maximum concurrent jobs: %d", maxRunning)
}

// === Context Cancellation Tests ===

func TestPool_ContextCancellation(t *testing.T) {
	pool := NewDefault()

	// Submit a slow job
	started := make(chan struct{})
	pool.Submit(func(ctx context.Context) (any, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			return "completed", nil
		}
	})

	// Wait for job to start
	<-started

	// Shutdown should cancel the context
	pool.ShutdownNow()

	// Results channel should be closed after shutdown
	select {
	case result, ok := <-pool.Results():
		if ok && result.Err == nil {
			t.Error("expected error or closed channel after ShutdownNow")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for shutdown")
	}
}

// === Shutdown Tests ===

func TestPool_GracefulShutdown(t *testing.T) {
	pool := NewDefault()

	// Submit jobs
	numJobs := 5
	var completed int32

	for i := 0; i < numJobs; i++ {
		pool.Submit(func(ctx context.Context) (any, error) {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&completed, 1)
			return nil, nil
		})
	}

	// Collect all results before shutdown
	go func() {
		for range pool.Results() {
			// Drain results
		}
	}()

	// Graceful shutdown waits for jobs to complete
	pool.Shutdown()

	if got := atomic.LoadInt32(&completed); got != int32(numJobs) {
		t.Errorf("completed = %d, want %d", got, numJobs)
	}
}

func TestPool_ShutdownNow(t *testing.T) {
	pool := NewDefault()

	// Submit a long-running job
	jobStarted := make(chan struct{})
	pool.Submit(func(ctx context.Context) (any, error) {
		close(jobStarted)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Minute):
			return "done", nil
		}
	})

	// Wait for job to start
	<-jobStarted

	// ShutdownNow should return quickly
	done := make(chan struct{})
	go func() {
		pool.ShutdownNow()
		close(done)
	}()

	select {
	case <-done:
		// Good - shutdown completed quickly
	case <-time.After(1 * time.Second):
		t.Fatal("ShutdownNow took too long")
	}
}

func TestPool_SubmitAfterShutdown(t *testing.T) {
	pool := NewDefault()
	pool.Shutdown()

	// Submit after shutdown should return -1
	id := pool.Submit(func(ctx context.Context) (any, error) {
		return nil, nil
	})

	if id != -1 {
		t.Errorf("Submit after shutdown returned %d, want -1", id)
	}
}

// === Benchmark Tests ===

func BenchmarkPool_Submit(b *testing.B) {
	pool := New(Config{Workers: 4, JobBuffer: 100, ResultBuffer: 100})
	defer pool.Shutdown()

	// Drain results in background
	go func() {
		for range pool.Results() {
		}
	}()

	job := func(ctx context.Context) (any, error) {
		return 42, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(job)
	}
}

func BenchmarkPool_RoundTrip(b *testing.B) {
	pool := New(Config{Workers: 4, JobBuffer: 100, ResultBuffer: 100})
	defer pool.Shutdown()

	job := func(ctx context.Context) (any, error) {
		return 42, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(job)
		<-pool.Results()
	}
}

