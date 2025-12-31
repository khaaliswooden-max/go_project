// pkg/sync/semaphore.go
// Semaphore implementation using buffered channels
//
// LEARN: A semaphore is a synchronization primitive that limits the number
// of concurrent accesses to a resource. In Go, we implement it using
// buffered channels - the channel capacity is the semaphore count.
//
// This is Exercise 2.3 - demonstrates buffered channels as semaphores.

package sync

import (
	"context"
	"errors"
)

// ErrSemaphoreReleased is returned when trying to acquire a released semaphore.
var ErrSemaphoreReleased = errors.New("semaphore has been released")

// Semaphore provides a counting semaphore using buffered channels.
//
// LEARN: Semaphores are useful for:
// - Rate limiting (limit concurrent API calls)
// - Resource pooling (limit database connections)
// - Worker throttling (limit parallel processing)
//
// Unlike sync.Mutex (binary lock), semaphores allow N concurrent accessors.
type Semaphore struct {
	// tokens is a buffered channel where capacity = max concurrent acquisitions.
	// LEARN: An empty struct{} uses 0 bytes - perfect for signaling.
	tokens chan struct{}

	// released indicates the semaphore has been shut down.
	// Once released, no new acquisitions are allowed.
	released chan struct{}
}

// New creates a new semaphore with the given capacity.
//
// LEARN: The capacity determines how many goroutines can hold the semaphore
// simultaneously. capacity=1 is equivalent to a mutex.
func New(capacity int) *Semaphore {
	if capacity <= 0 {
		capacity = 1
	}
	return &Semaphore{
		tokens:   make(chan struct{}, capacity),
		released: make(chan struct{}),
	}
}

// Acquire acquires a semaphore token, blocking until one is available
// or the context is cancelled.
//
// LEARN: This demonstrates the "context-aware blocking" pattern:
// - We want to wait for a resource
// - But we also want to respect cancellation/timeout
// - select statement handles both cases
//
// Returns nil on successful acquisition, or an error if:
// - Context was cancelled (context.Canceled)
// - Context deadline exceeded (context.DeadlineExceeded)
// - Semaphore was released (ErrSemaphoreReleased)
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.released:
		return ErrSemaphoreReleased
	case s.tokens <- struct{}{}:
		return nil
	}
}

// TryAcquire attempts to acquire a semaphore token without blocking.
//
// LEARN: Non-blocking acquisition is useful when you want to:
// - Fall back to alternative behavior if busy
// - Implement "try-once" semantics
// - Check availability without committing
//
// Returns true if acquired, false if the semaphore is full or released.
func (s *Semaphore) TryAcquire() bool {
	// LEARN: We must check released FIRST with priority. If we put all cases
	// in one select, Go randomly picks when multiple cases are ready.
	// This ensures released always takes precedence.
	select {
	case <-s.released:
		return false
	default:
	}

	// Now try to acquire (non-blocking)
	select {
	case s.tokens <- struct{}{}:
		return true
	default:
		// Semaphore is full, don't block
		return false
	}
}

// AcquireMany acquires n tokens, blocking until all are available
// or the context is cancelled.
//
// LEARN: This is an "all-or-nothing" acquisition pattern.
// If we can't get all tokens, we must release any we've acquired
// to avoid deadlock (partial acquisition could block other waiters).
func (s *Semaphore) AcquireMany(ctx context.Context, n int) error {
	if n <= 0 {
		return nil
	}

	acquired := 0
	for acquired < n {
		select {
		case <-ctx.Done():
			// Release any tokens we've acquired
			s.releaseMany(acquired)
			return ctx.Err()
		case <-s.released:
			s.releaseMany(acquired)
			return ErrSemaphoreReleased
		case s.tokens <- struct{}{}:
			acquired++
		}
	}
	return nil
}

// Release releases a semaphore token, making it available for others.
//
// LEARN: Release is typically called with defer immediately after Acquire:
//
//	if err := sem.Acquire(ctx); err != nil {
//	    return err
//	}
//	defer sem.Release()
//
// WARNING: Releasing more times than acquired causes panic (reading from
// empty channel in production would block forever).
func (s *Semaphore) Release() {
	select {
	case <-s.released:
		// Semaphore is released, no-op to be graceful
		return
	default:
		select {
		case <-s.tokens:
			// Successfully released
		default:
			// No tokens to release - this is a programming error
			panic("semaphore: release without acquire")
		}
	}
}

// ReleaseMany releases n tokens.
func (s *Semaphore) ReleaseMany(n int) {
	s.releaseMany(n)
}

// releaseMany is the internal implementation to avoid the panic check.
func (s *Semaphore) releaseMany(n int) {
	for i := 0; i < n; i++ {
		select {
		case <-s.released:
			return
		case <-s.tokens:
			// Token released
		default:
			// No more tokens to release
			return
		}
	}
}

// Available returns the number of available tokens.
//
// LEARN: This is inherently racy - by the time you act on this value,
// it may have changed. Use only for metrics/debugging, not for logic.
func (s *Semaphore) Available() int {
	return cap(s.tokens) - len(s.tokens)
}

// Capacity returns the maximum number of concurrent acquisitions.
func (s *Semaphore) Capacity() int {
	return cap(s.tokens)
}

// Close releases the semaphore, causing all waiting Acquire calls to return
// ErrSemaphoreReleased. This should be called during shutdown.
//
// LEARN: Closing signals to all waiters that the semaphore is no longer
// accepting acquisitions. This prevents goroutine leaks during shutdown.
func (s *Semaphore) Close() {
	select {
	case <-s.released:
		// Already released
	default:
		close(s.released)
	}
}
