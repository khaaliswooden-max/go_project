// pkg/perf/alloc.go
// Allocation reduction techniques and utilities
//
// LEARN: This file demonstrates techniques for reducing allocations
// in hot paths. Every allocation has cost:
// - Memory bandwidth for allocation
// - GC pressure for cleanup
// - Cache pressure from fragmentation
//
// Key techniques:
// 1. Pre-allocation with known sizes
// 2. Reusing buffers via sync.Pool
// 3. Avoiding interface{} conversions
// 4. Using stack allocation when possible

package perf

import (
	"strings"
	"sync"
	"unicode/utf8"
)

// StringBuilder provides efficient string building with minimal allocations.
//
// LEARN: strings.Builder is already optimized, but this wrapper adds:
// - Pooling for the Builder itself
// - Pre-sizing based on expected length
// - Reset for reuse
type StringBuilder struct {
	pool        sync.Pool
	initialSize int
}

// NewStringBuilder creates a new pooled string builder.
func NewStringBuilder(initialSize int) *StringBuilder {
	if initialSize <= 0 {
		initialSize = 256
	}
	return &StringBuilder{
		pool: sync.Pool{
			New: func() any {
				b := &strings.Builder{}
				b.Grow(initialSize)
				return b
			},
		},
		initialSize: initialSize,
	}
}

// Get retrieves a Builder from the pool.
func (sb *StringBuilder) Get() *strings.Builder {
	return sb.pool.Get().(*strings.Builder)
}

// Put returns a Builder to the pool after resetting.
func (sb *StringBuilder) Put(b *strings.Builder) {
	// Only pool if capacity is reasonable
	if b.Cap() <= sb.initialSize*4 {
		b.Reset()
		sb.pool.Put(b)
	}
}

// Build executes a function with a pooled Builder and returns the result.
//
// LEARN: This pattern ensures proper pool usage:
//
//	result := builder.Build(func(b *strings.Builder) {
//	    b.WriteString("hello ")
//	    b.WriteString("world")
//	})
func (sb *StringBuilder) Build(fn func(*strings.Builder)) string {
	b := sb.Get()
	fn(b)
	result := b.String()
	sb.Put(b)
	return result
}

// === Slice utilities ===

// SlicePool pools slices of a specific element type.
//
// LEARN: Pooling slices is tricky because:
// 1. Slices are headers (ptr, len, cap) that point to backing arrays
// 2. The backing array is what we want to reuse
// 3. Resizing may cause reallocation
//
// This pool stores pointers to slice headers to maintain identity.
type SlicePool[T any] struct {
	pool     sync.Pool
	capacity int
}

// NewSlicePool creates a pool for slices with given capacity.
func NewSlicePool[T any](capacity int) *SlicePool[T] {
	return &SlicePool[T]{
		pool: sync.Pool{
			New: func() any {
				s := make([]T, 0, capacity)
				return &s
			},
		},
		capacity: capacity,
	}
}

// Get retrieves a slice from the pool (length=0, capacity=initial).
func (p *SlicePool[T]) Get() []T {
	s := p.pool.Get().(*[]T)
	return (*s)[:0]
}

// Put returns a slice to the pool.
func (p *SlicePool[T]) Put(s []T) {
	// Only pool if capacity is within expected range
	if cap(s) <= p.capacity*4 && cap(s) >= p.capacity/2 {
		s = s[:0]
		p.pool.Put(&s)
	}
}

// === Pre-allocation helpers ===

// GrowSlice ensures a slice has at least the specified capacity.
//
// LEARN: This avoids multiple reallocations when appending:
//
//	// Bad: Multiple reallocations
//	var s []int
//	for i := 0; i < 1000; i++ {
//	    s = append(s, i)  // Reallocates at 1, 2, 4, 8, 16...
//	}
//
//	// Good: Single allocation
//	s := GrowSlice([]int{}, 1000)
//	for i := 0; i < 1000; i++ {
//	    s = append(s, i)  // No reallocation needed
//	}
func GrowSlice[T any](s []T, capacity int) []T {
	if cap(s) >= capacity {
		return s
	}
	newSlice := make([]T, len(s), capacity)
	copy(newSlice, s)
	return newSlice
}

// PreallocMap creates a map with pre-allocated space.
//
// LEARN: Maps grow dynamically, causing allocations.
// Pre-sizing avoids growth during population.
func PreallocMap[K comparable, V any](size int) map[K]V {
	return make(map[K]V, size)
}

// === Zero-copy string/byte conversions ===
// LEARN: These are UNSAFE and should only be used when:
// 1. Performance is critical
// 2. You guarantee the source won't be modified
// 3. You understand the memory safety implications
//
// For now, we provide safe versions. Phase 5 (unsafe) will cover
// the zero-copy variants.

// StringToBytes converts string to []byte.
// This is a safe implementation that allocates.
func StringToBytes(s string) []byte {
	return []byte(s)
}

// BytesToString converts []byte to string.
// This is a safe implementation that allocates.
func BytesToString(b []byte) string {
	return string(b)
}

// === Allocation-aware string operations ===

// ConcatStrings joins strings efficiently.
//
// LEARN: Naive concatenation allocates for each +:
//
//	result := a + b + c + d  // 3 allocations!
//
// strings.Builder allocates once:
func ConcatStrings(parts ...string) string {
	// Calculate total length
	n := 0
	for _, s := range parts {
		n += len(s)
	}

	// Pre-size builder
	var b strings.Builder
	b.Grow(n)

	// Build without reallocations
	for _, s := range parts {
		b.WriteString(s)
	}

	return b.String()
}

// ConcatBytes joins byte slices efficiently into a new slice.
func ConcatBytes(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}

	result := make([]byte, 0, n)
	for _, p := range parts {
		result = append(result, p...)
	}
	return result
}

// === Escape analysis helpers ===

// LEARN: These functions demonstrate escape analysis patterns.
// Use `go build -gcflags='-m'` to see escape decisions.

// noEscape is a helper to mark a value as not escaping.
// This is a hint to the compiler; actual escape depends on usage.
//
//go:noinline
func noEscape[T any](v T) T {
	return v
}

// Point demonstrates stack vs heap allocation.
type Point struct {
	X, Y float64
}

// NewPointValue returns a Point by value (stays on stack).
//
//go:noinline
func NewPointValue(x, y float64) Point {
	return Point{X: x, Y: y}
}

// NewPointPointer returns a *Point (escapes to heap).
//
//go:noinline
func NewPointPointer(x, y float64) *Point {
	return &Point{X: x, Y: y}
}

// === Rune/string utilities without allocation ===

// CountRunes counts runes without allocating (unlike []rune conversion).
//
// LEARN: Converting string to []rune allocates:
//
//	len([]rune(s))  // Allocates rune slice!
//
// Use utf8.RuneCountInString instead:
func CountRunes(s string) int {
	return utf8.RuneCountInString(s)
}

// FirstRune returns the first rune without allocating.
func FirstRune(s string) (rune, int) {
	return utf8.DecodeRuneInString(s)
}

// ForEachRune iterates over runes without allocating a slice.
//
// LEARN: This avoids []rune allocation:
//
//	// Allocates
//	for _, r := range []rune(s) { ... }
//
//	// No allocation
//	for _, r := range s { ... }  // Go iterates runes directly
//
// This function provides the same iteration with callback:
func ForEachRune(s string, fn func(i int, r rune)) {
	for i, r := range s {
		fn(i, r)
	}
}

