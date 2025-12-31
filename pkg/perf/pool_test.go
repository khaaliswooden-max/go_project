// pkg/perf/pool_test.go
// Tests and benchmarks for memory pooling
//
// LEARN: This file demonstrates:
// 1. Testing pool correctness
// 2. Benchmarking allocation reduction
// 3. Comparing pooled vs non-pooled performance

package perf

import (
	"bytes"
	"sync"
	"testing"
)

// === Unit Tests ===

func TestPool_Basic(t *testing.T) {
	// LEARN: Test that pool correctly returns objects
	pool := NewPool(
		func() *bytes.Buffer { return new(bytes.Buffer) },
		(*bytes.Buffer).Reset,
	)

	// Get should return a buffer
	buf := pool.Get()
	if buf == nil {
		t.Fatal("Get returned nil")
	}

	// Write some data
	buf.WriteString("test data")

	// Release returns to pool
	pool.Release(buf)

	// Get again - should get a reset buffer
	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Errorf("Buffer was not reset, len = %d", buf2.Len())
	}
}

func TestPool_Concurrent(t *testing.T) {
	// LEARN: sync.Pool is safe for concurrent use
	pool := NewPool(
		func() *bytes.Buffer { return new(bytes.Buffer) },
		(*bytes.Buffer).Reset,
	)

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf := pool.Get()
				buf.WriteString("hello")
				pool.Release(buf)
			}
		}(i)
	}

	wg.Wait()
}

func TestByteBufferPool(t *testing.T) {
	pool := NewByteBufferPool(1024)

	buf := pool.Get()
	if buf == nil {
		t.Fatal("Get returned nil")
	}

	// Check initial capacity
	if buf.Cap() < 1024 {
		t.Errorf("Expected capacity >= 1024, got %d", buf.Cap())
	}

	buf.WriteString("test")
	pool.Put(buf)

	// Get again and verify reset
	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Errorf("Buffer not reset, len = %d", buf2.Len())
	}
}

func TestByteBufferPool_OversizedNotPooled(t *testing.T) {
	// LEARN: Oversized buffers should not be pooled to avoid memory waste
	pool := NewByteBufferPool(1024)

	buf := pool.Get()

	// Grow buffer way beyond initial size
	buf.Grow(100000)
	buf.WriteString("x")

	// This should NOT be pooled (capacity > 4x initial)
	pool.Put(buf)

	// Get a new buffer - should be fresh, not the oversized one
	buf2 := pool.Get()
	if buf2.Cap() > 1024*4 {
		t.Errorf("Oversized buffer was pooled, cap = %d", buf2.Cap())
	}
}

func TestTrackedPool(t *testing.T) {
	// LEARN: TrackedPool provides statistics for debugging
	pool := NewTrackedPool(
		func() *bytes.Buffer { return new(bytes.Buffer) },
		(*bytes.Buffer).Reset,
	)

	// First get creates new object
	buf := pool.Get()
	pool.Release(buf)

	stats := pool.Stats()
	if stats.Gets != 1 {
		t.Errorf("Expected 1 get, got %d", stats.Gets)
	}
	if stats.Puts != 1 {
		t.Errorf("Expected 1 put, got %d", stats.Puts)
	}
	if stats.News != 1 {
		t.Errorf("Expected 1 new, got %d", stats.News)
	}
}

// === Benchmarks ===

// BenchmarkWithPool shows allocation reduction from pooling.
//
// LEARN: Run with: go test -bench=BenchmarkWithPool -benchmem ./pkg/perf/...
//
// Expected output shows reduced allocations:
//
//	BenchmarkWithPool/no_pool-8     5000000   300 ns/op   4096 B/op   1 allocs/op
//	BenchmarkWithPool/with_pool-8  20000000    50 ns/op      0 B/op   0 allocs/op
func BenchmarkWithPool(b *testing.B) {
	b.Run("no_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.Grow(4096)
			buf.WriteString("hello world")
			_ = buf.String()
		}
	})

	b.Run("with_pool", func(b *testing.B) {
		pool := NewByteBufferPool(4096)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			buf.WriteString("hello world")
			_ = buf.String()
			pool.Put(buf)
		}
	})
}

// BenchmarkGenericPool compares generic vs typed pools.
func BenchmarkGenericPool(b *testing.B) {
	type Data struct {
		ID   int
		Name string
		Tags []string
	}

	b.Run("no_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			d := &Data{
				ID:   i,
				Name: "test",
				Tags: make([]string, 0, 10),
			}
			_ = d
		}
	})

	b.Run("generic_pool", func(b *testing.B) {
		pool := NewPool(
			func() *Data {
				return &Data{Tags: make([]string, 0, 10)}
			},
			func(d *Data) {
				d.ID = 0
				d.Name = ""
				d.Tags = d.Tags[:0]
			},
		)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			d := pool.Get()
			d.ID = i
			d.Name = "test"
			pool.Release(d)
		}
	})
}

// BenchmarkPoolContention measures performance under contention.
//
// LEARN: sync.Pool uses per-P (processor) caches to reduce contention.
// This benchmark shows it scales well with goroutines.
func BenchmarkPoolContention(b *testing.B) {
	pool := NewByteBufferPool(4096)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			buf.WriteString("hello world from parallel benchmark")
			pool.Put(buf)
		}
	})
}

// BenchmarkByteSlicePool compares slice pooling strategies.
func BenchmarkByteSlicePool(b *testing.B) {
	const size = 4096

	b.Run("make_each_time", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := make([]byte, size)
			buf[0] = 'x'
			_ = buf
		}
	})

	b.Run("with_pool", func(b *testing.B) {
		pool := NewByteSlicePool(size)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			buf[0] = 'x'
			pool.Put(buf)
		}
	})
}

// === Example Tests ===

func ExamplePool() {
	// Create a pool for bytes.Buffer with reset function
	pool := NewPool(
		func() *bytes.Buffer { return new(bytes.Buffer) },
		(*bytes.Buffer).Reset,
	)

	// Get a buffer from the pool
	buf := pool.Get()
	buf.WriteString("Hello, World!")

	// Use the buffer...
	_ = buf.String()

	// Return to pool for reuse
	pool.Release(buf)
}

func ExampleByteBufferPool() {
	// Create pool with 4KB initial buffer size
	pool := NewByteBufferPool(4096)

	// Get buffer, use it, return it
	WithPooling(pool, func(buf *bytes.Buffer) {
		buf.WriteString("Efficiently pooled buffer")
		// Process buf.Bytes()...
	})
}
