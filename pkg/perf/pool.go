// pkg/perf/pool.go
// Memory pooling with sync.Pool for allocation reduction
//
// LEARN: sync.Pool reduces garbage collection pressure by reusing
// allocations. This is Exercise 4.1 - build a type-safe generic pool.
//
// Key concepts:
// 1. sync.Pool stores and reuses temporary objects
// 2. Objects may be garbage collected between GC cycles
// 3. Pool is safe for concurrent use
// 4. Always reset objects before returning to pool

package perf

import (
	"bytes"
	"sync"
)

// Pool is a type-safe wrapper around sync.Pool.
//
// LEARN: This combines Phase 3 generics with Phase 4 performance:
// - Generic type parameter T eliminates type assertions
// - sync.Pool provides the actual pooling mechanism
// - Reset function ensures clean objects on reuse
//
// Compare to raw sync.Pool:
//
//	// Raw sync.Pool (requires type assertions)
//	pool := &sync.Pool{New: func() any { return &Buffer{} }}
//	buf := pool.Get().(*Buffer)  // Type assertion required!
//	pool.Put(buf)
//
//	// Generic Pool (type-safe)
//	pool := NewPool(func() *Buffer { return &Buffer{} }, (*Buffer).Reset)
//	buf := pool.Get()  // Returns *Buffer directly!
//	pool.Release(buf)
type Pool[T any] struct {
	pool  sync.Pool
	reset func(T)
}

// NewPool creates a new type-safe pool.
//
// Parameters:
//   - factory: Function to create new objects when pool is empty
//   - reset: Optional function to reset objects before returning to pool
//
// LEARN: The factory function is called when:
// 1. Pool is empty and Get() is called
// 2. All pooled objects were garbage collected
//
// The reset function is called on Release() to ensure:
// 1. No stale data leaks between uses
// 2. Object is in clean state for next use
func NewPool[T any](factory func() T, reset func(T)) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any { return factory() },
		},
		reset: reset,
	}
}

// Get retrieves an object from the pool.
//
// LEARN: Get may return:
// - A previously pooled object (via Release)
// - A newly created object (via factory function)
//
// The caller should NOT assume object state; reset happens on Release,
// but the object may be newly created.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Release returns an object to the pool.
//
// LEARN: Critical rules for Release:
// 1. Object is reset (if reset func provided) before pooling
// 2. Caller must NOT use the object after Release
// 3. Passing nil is safe (no-op for pointer types)
//
// WARNING: Using an object after Release is a data race!
//
//	buf := pool.Get()
//	pool.Release(buf)
//	buf.Write([]byte("oops"))  // DATA RACE: buf may be in use elsewhere!
func (p *Pool[T]) Release(item T) {
	if p.reset != nil {
		p.reset(item)
	}
	p.pool.Put(item)
}

// Put is an alias for Release (matches sync.Pool API).
func (p *Pool[T]) Put(item T) {
	p.Release(item)
}

// === Specialized Pools ===

// ByteBufferPool is a specialized pool for bytes.Buffer.
//
// LEARN: bytes.Buffer is one of the most commonly pooled types because:
// 1. Frequently used for building strings/byte slices
// 2. Grows dynamically, reusing capacity is valuable
// 3. Reset() efficiently clears without deallocating
type ByteBufferPool struct {
	pool        *Pool[*bytes.Buffer]
	initialSize int
}

// NewByteBufferPool creates a pool of bytes.Buffer with initial capacity.
//
// The initialSize determines the starting capacity of new buffers.
// Larger values reduce reallocations for big payloads but waste
// memory for small payloads.
//
// LEARN: Choosing initial size:
//   - Too small: Frequent reallocations during use
//   - Too large: Wasted memory in pooled buffers
//   - Rule of thumb: Use median expected payload size
func NewByteBufferPool(initialSize int) *ByteBufferPool {
	if initialSize <= 0 {
		initialSize = 4096
	}
	return &ByteBufferPool{
		pool: NewPool(
			func() *bytes.Buffer {
				return bytes.NewBuffer(make([]byte, 0, initialSize))
			},
			(*bytes.Buffer).Reset,
		),
		initialSize: initialSize,
	}
}

// Get retrieves a buffer from the pool.
func (p *ByteBufferPool) Get() *bytes.Buffer {
	return p.pool.Get()
}

// Put returns a buffer to the pool after resetting.
func (p *ByteBufferPool) Put(buf *bytes.Buffer) {
	// LEARN: Check capacity to avoid pooling unexpectedly large buffers.
	// If a buffer grew very large (e.g., 10MB), pooling it wastes memory
	// since we'll likely only need 4KB next time.
	if buf.Cap() > p.initialSize*4 {
		// Let GC reclaim oversized buffer; create fresh one on next Get
		return
	}
	p.pool.Put(buf)
}

// ByteSlicePool is a specialized pool for []byte slices.
//
// LEARN: []byte pools are useful for:
// - Network I/O buffers
// - Temporary data processing
// - JSON/encoding buffers
type ByteSlicePool struct {
	pool *Pool[*[]byte]
	size int
}

// NewByteSlicePool creates a pool of byte slices with fixed size.
func NewByteSlicePool(size int) *ByteSlicePool {
	if size <= 0 {
		size = 4096
	}
	return &ByteSlicePool{
		pool: NewPool(
			func() *[]byte {
				b := make([]byte, size)
				return &b
			},
			func(b *[]byte) {
				// Reset by clearing the slice (keeping capacity)
				*b = (*b)[:0]
			},
		),
		size: size,
	}
}

// Get retrieves a byte slice from the pool.
func (p *ByteSlicePool) Get() []byte {
	b := p.pool.Get()
	return (*b)[:p.size]
}

// GetWithLength retrieves a byte slice with specific length.
func (p *ByteSlicePool) GetWithLength(length int) []byte {
	b := p.pool.Get()
	if length > p.size {
		length = p.size
	}
	return (*b)[:length]
}

// Put returns a byte slice to the pool.
func (p *ByteSlicePool) Put(b []byte) {
	// We need to put back the pointer, not the slice
	// This requires the original pointer, which we don't have.
	// LEARN: This is a limitation of pooling slices directly.
	// The ByteSlicePool stores *[]byte to maintain identity.
	p.pool.Put(&b)
}

// === Benchmarking Helpers ===

// WithPooling runs a function with buffer pooling enabled.
//
// LEARN: This pattern lets you compare pooled vs. non-pooled allocations
// in benchmarks:
//
//	func BenchmarkWithPool(b *testing.B) {
//	    for i := 0; i < b.N; i++ {
//	        perf.WithPooling(pool, func(buf *bytes.Buffer) {
//	            buf.WriteString("hello")
//	        })
//	    }
//	}
func WithPooling(pool *ByteBufferPool, fn func(*bytes.Buffer)) {
	buf := pool.Get()
	fn(buf)
	pool.Put(buf)
}

// Stats tracks pool usage statistics for debugging.
//
// LEARN: sync.Pool doesn't expose statistics, so we wrap it
// with counters for debugging and optimization.
type Stats struct {
	Gets   int64
	Puts   int64
	News   int64
	Reuses int64
}

// TrackedPool wraps a Pool with usage statistics.
type TrackedPool[T any] struct {
	pool    *Pool[T]
	stats   Stats
	statsMu sync.Mutex
}

// NewTrackedPool creates a pool that tracks usage statistics.
func NewTrackedPool[T any](factory func() T, reset func(T)) *TrackedPool[T] {
	// Wrap factory to track new allocations
	var tp *TrackedPool[T]
	wrappedFactory := func() T {
		if tp != nil {
			tp.statsMu.Lock()
			tp.stats.News++
			tp.statsMu.Unlock()
		}
		return factory()
	}

	tp = &TrackedPool[T]{
		pool: NewPool(wrappedFactory, reset),
	}
	return tp
}

// Get retrieves an object and updates statistics.
func (p *TrackedPool[T]) Get() T {
	p.statsMu.Lock()
	p.stats.Gets++
	p.statsMu.Unlock()
	return p.pool.Get()
}

// Release returns an object and updates statistics.
func (p *TrackedPool[T]) Release(item T) {
	p.statsMu.Lock()
	p.stats.Puts++
	p.statsMu.Unlock()
	p.pool.Release(item)
}

// Stats returns a copy of current statistics.
//
// LEARN: Returning a copy prevents race conditions if caller
// reads stats while pool is in use.
func (p *TrackedPool[T]) Stats() Stats {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	return p.stats
}

// ReuseRate calculates the percentage of Gets that reused pooled objects.
//
// LEARN: High reuse rate (>80%) indicates the pool is effective.
// Low reuse rate may mean:
// - Pool is too small
// - Objects are held too long
// - GC is clearing the pool frequently
func (p *TrackedPool[T]) ReuseRate() float64 {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()

	if p.stats.Gets == 0 {
		return 0
	}
	reuses := p.stats.Gets - p.stats.News
	return float64(reuses) / float64(p.stats.Gets) * 100
}
