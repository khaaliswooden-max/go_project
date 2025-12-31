// pkg/sync/semaphore_test.go
// Tests for semaphore implementation
//
// LEARN: Testing concurrent code requires:
// 1. Race detector: go test -race
// 2. Timeouts: Prevent tests from hanging forever
// 3. Synchronization: Properly wait for goroutines
// 4. Deterministic assertions: Avoid timing-dependent tests

package sync

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// === Construction Tests ===

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		want     int
	}{
		{"positive capacity", 5, 5},
		{"capacity one", 1, 1},
		{"zero capacity defaults to 1", 0, 1},
		{"negative capacity defaults to 1", -1, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sem := New(tc.capacity)
			if sem.Capacity() != tc.want {
				t.Errorf("Capacity() = %d, want %d", sem.Capacity(), tc.want)
			}
			if sem.Available() != tc.want {
				t.Errorf("Available() = %d, want %d", sem.Available(), tc.want)
			}
		})
	}
}

// === Basic Acquire/Release Tests ===

func TestSemaphore_AcquireRelease(t *testing.T) {
	sem := New(3)

	// Acquire all tokens
	for i := 0; i < 3; i++ {
		if err := sem.Acquire(context.Background()); err != nil {
			t.Fatalf("Acquire %d failed: %v", i, err)
		}
	}

	// Should have 0 available
	if avail := sem.Available(); avail != 0 {
		t.Errorf("Available() = %d, want 0 after acquiring all", avail)
	}

	// Release all tokens
	for i := 0; i < 3; i++ {
		sem.Release()
	}

	// Should have all available again
	if avail := sem.Available(); avail != 3 {
		t.Errorf("Available() = %d, want 3 after releasing all", avail)
	}
}

func TestSemaphore_TryAcquire(t *testing.T) {
	sem := New(2)

	// Should succeed twice
	if !sem.TryAcquire() {
		t.Error("first TryAcquire() = false, want true")
	}
	if !sem.TryAcquire() {
		t.Error("second TryAcquire() = false, want true")
	}

	// Third should fail (non-blocking)
	if sem.TryAcquire() {
		t.Error("third TryAcquire() = true, want false (full)")
	}

	// Release one and try again
	sem.Release()
	if !sem.TryAcquire() {
		t.Error("TryAcquire() after release = false, want true")
	}
}

// === Context Cancellation Tests ===

func TestSemaphore_AcquireWithCancelledContext(t *testing.T) {
	sem := New(1)

	// Fill the semaphore
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("initial Acquire failed: %v", err)
	}

	// Try to acquire with already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := sem.Acquire(ctx)
	if err != context.Canceled {
		t.Errorf("Acquire with cancelled context = %v, want context.Canceled", err)
	}
}

func TestSemaphore_AcquireWithTimeout(t *testing.T) {
	sem := New(1)

	// Fill the semaphore
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("initial Acquire failed: %v", err)
	}

	// Try to acquire with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := sem.Acquire(ctx)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("Acquire with timeout = %v, want context.DeadlineExceeded", err)
	}

	// Should have waited approximately the timeout duration
	if elapsed < 40*time.Millisecond {
		t.Errorf("Acquire returned too quickly: %v", elapsed)
	}
}

func TestSemaphore_AcquireContextCancellation(t *testing.T) {
	sem := New(1)

	// Fill the semaphore
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("initial Acquire failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start acquire in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- sem.Acquire(ctx)
	}()

	// Cancel after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Should receive cancellation error
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Acquire after cancel = %v, want context.Canceled", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Acquire to return after cancel")
	}
}

// === AcquireMany Tests ===

func TestSemaphore_AcquireMany(t *testing.T) {
	sem := New(5)

	// Acquire 3 tokens
	if err := sem.AcquireMany(context.Background(), 3); err != nil {
		t.Fatalf("AcquireMany(3) failed: %v", err)
	}

	if avail := sem.Available(); avail != 2 {
		t.Errorf("Available() = %d, want 2", avail)
	}

	// Acquire remaining 2
	if err := sem.AcquireMany(context.Background(), 2); err != nil {
		t.Fatalf("AcquireMany(2) failed: %v", err)
	}

	if avail := sem.Available(); avail != 0 {
		t.Errorf("Available() = %d, want 0", avail)
	}
}

func TestSemaphore_AcquireManyZero(t *testing.T) {
	sem := New(5)

	// Acquiring 0 should succeed immediately
	if err := sem.AcquireMany(context.Background(), 0); err != nil {
		t.Errorf("AcquireMany(0) = %v, want nil", err)
	}

	// Negative should also succeed
	if err := sem.AcquireMany(context.Background(), -1); err != nil {
		t.Errorf("AcquireMany(-1) = %v, want nil", err)
	}

	// Available shouldn't change
	if avail := sem.Available(); avail != 5 {
		t.Errorf("Available() = %d, want 5", avail)
	}
}

func TestSemaphore_AcquireManyRollback(t *testing.T) {
	sem := New(5)

	// Acquire 3 to leave only 2 available
	if err := sem.AcquireMany(context.Background(), 3); err != nil {
		t.Fatalf("initial AcquireMany failed: %v", err)
	}

	// Try to acquire 3 more with timeout (only 2 available)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := sem.AcquireMany(ctx, 3)
	if err != context.DeadlineExceeded {
		t.Errorf("AcquireMany = %v, want context.DeadlineExceeded", err)
	}

	// The partial acquisitions should have been rolled back
	// Original 3 + 0 from failed attempt = 2 available
	if avail := sem.Available(); avail != 2 {
		t.Errorf("Available() after rollback = %d, want 2", avail)
	}
}

// === ReleaseMany Tests ===

func TestSemaphore_ReleaseMany(t *testing.T) {
	sem := New(5)

	// Acquire all
	if err := sem.AcquireMany(context.Background(), 5); err != nil {
		t.Fatalf("AcquireMany failed: %v", err)
	}

	// Release 3
	sem.ReleaseMany(3)
	if avail := sem.Available(); avail != 3 {
		t.Errorf("Available() = %d, want 3", avail)
	}

	// Release remaining 2
	sem.ReleaseMany(2)
	if avail := sem.Available(); avail != 5 {
		t.Errorf("Available() = %d, want 5", avail)
	}
}

// === Concurrency Tests ===

func TestSemaphore_ConcurrentBoundedAccess(t *testing.T) {
	// LEARN: Verify semaphore actually limits concurrency
	const capacity = 3
	const workers = 10
	const iterations = 5

	sem := New(capacity)
	var concurrent int32
	var maxConcurrent int32
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if err := sem.Acquire(context.Background()); err != nil {
					t.Errorf("Acquire failed: %v", err)
					return
				}

				// Track concurrency
				current := atomic.AddInt32(&concurrent, 1)
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
						break
					}
				}

				time.Sleep(10 * time.Millisecond) // Simulate work
				atomic.AddInt32(&concurrent, -1)

				sem.Release()
			}
		}()
	}

	wg.Wait()

	// Max concurrent should never exceed capacity
	if max := atomic.LoadInt32(&maxConcurrent); max > capacity {
		t.Errorf("maxConcurrent = %d, want <= %d", max, capacity)
	}
	t.Logf("Max concurrent accessors: %d (capacity: %d)", maxConcurrent, capacity)
}

func TestSemaphore_ConcurrentAcquireRelease(t *testing.T) {
	// LEARN: Race detector will catch any data races
	const capacity = 5
	const operations = 100

	sem := New(capacity)
	var wg sync.WaitGroup

	// Many goroutines acquiring and releasing
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sem.TryAcquire() {
				time.Sleep(time.Millisecond)
				sem.Release()
			}
		}()
	}

	wg.Wait()

	// All tokens should be available after all operations complete
	if avail := sem.Available(); avail != capacity {
		t.Errorf("Available() = %d, want %d", avail, capacity)
	}
}

// === Close/Release Tests ===

func TestSemaphore_Close(t *testing.T) {
	sem := New(2)

	// Start waiting goroutine
	errCh := make(chan error, 1)
	started := make(chan struct{})

	go func() {
		// Fill semaphore first
		sem.Acquire(context.Background())
		sem.Acquire(context.Background())
		close(started)

		// This should block until Close is called
		errCh <- sem.Acquire(context.Background())
	}()

	// Wait for semaphore to be full
	<-started
	time.Sleep(50 * time.Millisecond)

	// Close the semaphore
	sem.Close()

	// Waiting goroutine should get ErrSemaphoreReleased
	select {
	case err := <-errCh:
		if err != ErrSemaphoreReleased {
			t.Errorf("Acquire after Close = %v, want ErrSemaphoreReleased", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Acquire to return after Close")
	}
}

func TestSemaphore_TryAcquireAfterClose(t *testing.T) {
	sem := New(5)
	sem.Close()

	if sem.TryAcquire() {
		t.Error("TryAcquire after Close = true, want false")
	}
}

func TestSemaphore_DoubleClose(t *testing.T) {
	sem := New(1)

	// Double close should not panic
	sem.Close()
	sem.Close() // Should be safe
}

// === Error Handling Tests ===

func TestSemaphore_ReleasePanic(t *testing.T) {
	sem := New(1)

	// Releasing without acquiring should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Release without Acquire did not panic")
		}
	}()

	sem.Release()
}

func TestSemaphore_ReleaseAfterClose(t *testing.T) {
	sem := New(1)

	// Acquire first
	if err := sem.Acquire(context.Background()); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	sem.Close()

	// Release after close should not panic (graceful)
	sem.Release() // Should be safe
}

// === Benchmark Tests ===

func BenchmarkSemaphore_AcquireRelease(b *testing.B) {
	sem := New(1)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sem.Acquire(ctx)
		sem.Release()
	}
}

func BenchmarkSemaphore_TryAcquire(b *testing.B) {
	sem := New(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if sem.TryAcquire() {
			sem.Release()
		}
	}
}

func BenchmarkSemaphore_Contended(b *testing.B) {
	sem := New(4)
	ctx := context.Background()
	var wg sync.WaitGroup

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem.Acquire(ctx)
			sem.Release()
		}()
	}
	wg.Wait()
}

