# Phase 4: Performance - Learning Guide

This document covers Go performance optimization techniques: profiling, benchmarking, memory pooling, and escape analysis. Phase 4 builds on the generic data structures from Phase 3 to create high-performance, allocation-efficient code.

---

## Prerequisites

Before starting Phase 4, ensure you've completed:
- [x] Phase 1: Foundations (interfaces, error handling, testing)
- [x] Phase 2: Concurrency (goroutines, channels, sync primitives)
- [x] Phase 3: Generics (type parameters, generic data structures)

---

## 1. Benchmarking with testing.B

**Files:** `pkg/perf/pool_test.go`, `pkg/perf/alloc_test.go`

### Concept
Go's testing package includes built-in benchmarking support. Benchmarks measure execution time and memory allocations, providing data-driven insights for optimization.

### What You'll Learn
```go
// LEARN: Basic benchmark structure
// The name must start with Benchmark and take *testing.B
func BenchmarkOperation(b *testing.B) {
    // Setup code runs once (outside the loop)
    data := prepareData()
    
    // Reset timer if setup is expensive
    b.ResetTimer()
    
    // Run the operation b.N times
    // b.N is adjusted by the framework to get stable results
    for i := 0; i < b.N; i++ {
        processData(data)
    }
}

// Run with: go test -bench=. -benchmem ./pkg/perf/...
```

### Benchmark Patterns
```go
// Pattern 1: Memory allocation tracking
func BenchmarkWithAllocs(b *testing.B) {
    b.ReportAllocs() // Report allocations per operation
    for i := 0; i < b.N; i++ {
        _ = createObject()
    }
}

// Pattern 2: Sub-benchmarks for comparison
func BenchmarkJSON(b *testing.B) {
    data := getData()
    
    b.Run("Marshal", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            json.Marshal(data)
        }
    })
    
    b.Run("Unmarshal", func(b *testing.B) {
        bytes, _ := json.Marshal(data)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            var result Data
            json.Unmarshal(bytes, &result)
        }
    })
}

// Pattern 3: Parallel benchmarks
func BenchmarkParallel(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        // Each goroutine runs this loop
        for pb.Next() {
            doWork()
        }
    })
}
```

### Benchmark Commands
```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkPool -benchmem ./pkg/perf/...

# Compare benchmarks (requires benchstat)
go test -bench=. -count=10 > old.txt
# ... make changes ...
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt

# CPU profile during benchmark
go test -bench=. -cpuprofile=cpu.out ./pkg/perf/...

# Memory profile during benchmark
go test -bench=. -memprofile=mem.out ./pkg/perf/...
```

### Key Principles
- **Stable Results:** b.N adjusts until timing is stable
- **Avoid Setup Overhead:** Use b.ResetTimer() after expensive setup
- **Report Allocations:** Always use -benchmem or b.ReportAllocs()
- **Compare Carefully:** Use benchstat for statistically valid comparisons

---

## 2. Profiling with pprof

**Files:** `pkg/perf/profile.go`, `internal/server/http.go`

### Concept
pprof is Go's built-in profiler for CPU, memory, goroutine, and mutex analysis. It helps identify hotspots and optimization opportunities in production code.

### What You'll Learn
```go
// LEARN: Enable pprof endpoints in HTTP server
import (
    "net/http"
    _ "net/http/pprof" // Side-effect import registers handlers
)

func main() {
    // If using default mux, pprof handlers auto-register
    go http.ListenAndServe(":6060", nil)
    
    // Your main server
    mainServer.ListenAndServe()
}

// For custom mux, register manually:
import "net/http/pprof"

func registerPprof(mux *http.ServeMux) {
    mux.HandleFunc("/debug/pprof/", pprof.Index)
    mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
    mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
    mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
    mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
    mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
}
```

### Profile Types
```go
// CPU Profile: Where is CPU time spent?
// Collect: curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.out
// Analyze: go tool pprof cpu.out

// Heap Profile: Current memory allocations
// Collect: curl http://localhost:6060/debug/pprof/heap > heap.out
// Analyze: go tool pprof heap.out

// Allocation Profile: All allocations (including freed)
// Collect: curl http://localhost:6060/debug/pprof/allocs > allocs.out

// Goroutine Profile: Stack traces of all goroutines
// Collect: curl http://localhost:6060/debug/pprof/goroutine > goroutine.out

// Block Profile: Where goroutines block on synchronization
// Enable: runtime.SetBlockProfileRate(1)
// Collect: curl http://localhost:6060/debug/pprof/block > block.out

// Mutex Profile: Mutex contention
// Enable: runtime.SetMutexProfileFraction(1)
// Collect: curl http://localhost:6060/debug/pprof/mutex > mutex.out
```

### Using pprof Tool
```bash
# Interactive mode
go tool pprof http://localhost:6060/debug/pprof/heap

# Common commands in interactive mode:
# top        - Show top functions by metric
# top20      - Show top 20
# list func  - Show annotated source for function
# web        - Open graph in browser (requires graphviz)
# png        - Save graph as PNG

# One-liner for web visualization
go tool pprof -http=:8080 cpu.out

# Comparing profiles
go tool pprof -base=old.out new.out
```

### Programmatic Profiling
```go
// LEARN: Profile specific code sections
import "runtime/pprof"

func profileFunction() {
    // CPU profiling
    f, _ := os.Create("cpu.out")
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
    
    // Do work...
    
    // Memory profile snapshot
    mf, _ := os.Create("mem.out")
    runtime.GC() // Get accurate heap snapshot
    pprof.WriteHeapProfile(mf)
}

// Labeling goroutines for better profiles
import "runtime/pprof"

func worker(ctx context.Context, id int) {
    labels := pprof.Labels("worker", fmt.Sprintf("worker-%d", id))
    pprof.Do(ctx, labels, func(ctx context.Context) {
        // Work appears under this label in profiles
        doWork(ctx)
    })
}
```

### Key Principles
- **CPU Profiling:** Sample-based; run for 30+ seconds for accuracy
- **Heap vs Allocs:** Heap shows live; allocs shows all (including GC'd)
- **Production Safety:** pprof has minimal overhead, safe for prod
- **Actionable Data:** Focus on top functions; optimize the biggest wins first

---

## 3. Memory Pooling with sync.Pool

**Files:** `pkg/perf/pool.go`

### Concept
`sync.Pool` reduces garbage collection pressure by reusing allocations. Objects are pooled between GC cycles, reducing allocation frequency for short-lived objects.

### What You'll Learn
```go
// LEARN: Basic sync.Pool usage
var bufferPool = sync.Pool{
    New: func() any {
        // Called when pool is empty
        return make([]byte, 0, 4096)
    },
}

func process(data []byte) {
    // Get buffer from pool
    buf := bufferPool.Get().([]byte)
    buf = buf[:0] // Reset length, keep capacity
    
    // Use buffer...
    buf = append(buf, data...)
    process(buf)
    
    // Return to pool (CRITICAL: don't use buf after this!)
    bufferPool.Put(buf)
}
```

### Pool Patterns
```go
// Pattern 1: Type-safe wrapper (recommended)
// LEARN: Avoid type assertions at every Get/Put site
type ByteBufferPool struct {
    pool sync.Pool
}

func NewByteBufferPool(size int) *ByteBufferPool {
    return &ByteBufferPool{
        pool: sync.Pool{
            New: func() any {
                return bytes.NewBuffer(make([]byte, 0, size))
            },
        },
    }
}

func (p *ByteBufferPool) Get() *bytes.Buffer {
    return p.pool.Get().(*bytes.Buffer)
}

func (p *ByteBufferPool) Put(buf *bytes.Buffer) {
    buf.Reset()
    p.pool.Put(buf)
}

// Pattern 2: Generic pool (Phase 3 + Phase 4!)
type Pool[T any] struct {
    pool sync.Pool
}

func NewPool[T any](factory func() T) *Pool[T] {
    return &Pool[T]{
        pool: sync.Pool{
            New: func() any { return factory() },
        },
    }
}

func (p *Pool[T]) Get() T {
    return p.pool.Get().(T)
}

func (p *Pool[T]) Put(item T) {
    p.pool.Put(item)
}

// Pattern 3: Pool with reset function
type PooledObject[T any] struct {
    pool  sync.Pool
    reset func(T)
}

func NewPooledObject[T any](factory func() T, reset func(T)) *PooledObject[T] {
    return &PooledObject[T]{
        pool: sync.Pool{New: func() any { return factory() }},
        reset: reset,
    }
}

func (p *PooledObject[T]) Acquire() T {
    return p.pool.Get().(T)
}

func (p *PooledObject[T]) Release(item T) {
    p.reset(item)
    p.pool.Put(item)
}
```

### When to Use sync.Pool
```go
// YES: Short-lived allocations in hot paths
// - Request/response buffers
// - Temporary byte slices
// - JSON encoder/decoder buffers
// - Compression buffers

// NO: Long-lived objects
// - Caches (use LRU cache instead)
// - Connection pools (sync.Pool doesn't guarantee lifetime)
// - Objects with expensive initialization

// NO: Small objects
// - Allocations < 16 bytes (not worth pooling overhead)
// - Objects allocated infrequently
```

### Key Principles
- **GC Clears Pool:** Objects may be garbage collected between GC cycles
- **No Size Guarantee:** Pool doesn't limit size or guarantee reuse
- **Reset Objects:** Always reset state before putting back
- **Don't Hold References:** Never use pooled object after Put()

---

## 4. Escape Analysis

**Files:** Examples in all `pkg/perf/*.go` files

### Concept
Escape analysis determines whether an object can stay on the stack (fast, no GC) or must escape to the heap (slower, requires GC). Understanding escape analysis helps write allocation-efficient code.

### What You'll Learn
```go
// LEARN: Stack allocation (no escape)
func stackAllocated() int {
    x := 42     // Stays on stack
    return x    // Value copied, x doesn't escape
}

// LEARN: Heap allocation (escapes)
func heapAllocated() *int {
    x := 42     // Must escape to heap
    return &x   // Pointer returned; x outlives function
}

// View escape analysis:
// go build -gcflags='-m' ./pkg/perf/...
// go build -gcflags='-m -m' ./pkg/perf/... (more verbose)
```

### Escape Patterns
```go
// Pattern 1: Interface conversion often causes escape
func escapeViaInterface(x int) any {
    return x // x escapes: interface{} can box any type
}

// Better: Use generics to avoid interface{}
func noEscape[T any](x T) T {
    return x // No interface conversion, may stay on stack
}

// Pattern 2: Slice/map literals with unknown size escape
func sliceEscapes() []int {
    return []int{1, 2, 3} // Escapes: returned slice
}

func sliceStays() {
    s := []int{1, 2, 3} // May stay on stack (small, local)
    _ = s
}

// Pattern 3: Closures capturing variables
func closureEscape() func() int {
    x := 42
    return func() int { return x } // x escapes: captured by closure
}

// Pattern 4: Large objects escape
func largeObjectEscapes() {
    // Arrays > ~64KB typically escape
    var huge [1000000]int // Likely escapes to heap
    _ = huge
}
```

### Reducing Escapes
```go
// Technique 1: Pass by value for small structs
type Point struct { X, Y float64 }

// Escapes: returns pointer to local
func NewPointBad() *Point {
    return &Point{X: 1, Y: 2}
}

// No escape: returns value
func NewPointGood() Point {
    return Point{X: 1, Y: 2}
}

// Technique 2: Pre-allocate and pass pointer
func processBuffer(buf []byte) {
    // Process buf...
}

func caller() {
    buf := make([]byte, 4096) // Single allocation
    for i := 0; i < 1000; i++ {
        processBuffer(buf) // Reuse buffer
    }
}

// Technique 3: Append to existing slice
func appendEscapes(data []int) []int {
    return append(data, 1, 2, 3) // May escape due to capacity growth
}

func appendNoEscape(data []int, cap int) []int {
    if cap(data) >= len(data)+3 {
        return append(data, 1, 2, 3) // No allocation if capacity exists
    }
    // Handle reallocation case...
    return data
}
```

### Analyzing Escapes
```bash
# Show escape analysis decisions
go build -gcflags='-m' ./pkg/perf/...

# Output example:
# ./pool.go:42: moved to heap: buf
# ./pool.go:47: make([]byte, size) escapes to heap
# ./pool.go:52: (*Pool[T]).Get: leaking param: p

# More verbose (shows inlining decisions too)
go build -gcflags='-m -m' ./pkg/perf/...
```

### Key Principles
- **Stack is Free:** Stack allocations have zero GC overhead
- **Heap Costs:** Each heap allocation adds GC pressure
- **Compiler Decides:** Escape analysis is automatic; you can influence it
- **Measure First:** Only optimize after profiling shows allocations matter

---

## 5. Reducing Allocations

**Files:** `pkg/perf/alloc.go`

### Concept
Every allocation has cost: memory bandwidth, cache pressure, and GC work. High-performance Go code minimizes allocations in hot paths.

### What You'll Learn
```go
// LEARN: Common allocation sources and alternatives

// 1. String concatenation (allocates)
func concatBad(parts []string) string {
    result := ""
    for _, p := range parts {
        result += p // New string allocated each iteration!
    }
    return result
}

// Better: Use strings.Builder
func concatGood(parts []string) string {
    var b strings.Builder
    for _, p := range parts {
        b.WriteString(p)
    }
    return b.String()
}

// Even better: Pre-size the builder
func concatBest(parts []string) string {
    total := 0
    for _, p := range parts {
        total += len(p)
    }
    var b strings.Builder
    b.Grow(total)
    for _, p := range parts {
        b.WriteString(p)
    }
    return b.String()
}

// 2. Slice operations
func sliceBad(data []int) []int {
    return append([]int{}, data...) // Always allocates
}

// Better: Reuse slice when possible
func sliceGood(data []int, result []int) []int {
    return append(result[:0], data...) // Reuses result's backing array
}

// 3. Map operations
func mapBad() {
    for i := 0; i < 1000; i++ {
        m := make(map[string]int) // 1000 allocations
        _ = m
    }
}

// Better: Reuse maps
func mapGood() {
    m := make(map[string]int)
    for i := 0; i < 1000; i++ {
        clear(m) // Go 1.21+: clear map without reallocating
        _ = m
    }
}
```

### Allocation Reduction Techniques
```go
// Technique 1: sync.Pool for temporary buffers
var bufPool = sync.Pool{
    New: func() any { return make([]byte, 0, 4096) },
}

func processWithPool(data []byte) []byte {
    buf := bufPool.Get().([]byte)
    defer bufPool.Put(buf)
    
    buf = buf[:0]
    buf = append(buf, data...)
    return transform(buf)
}

// Technique 2: Array instead of slice for fixed size
type Request struct {
    // Bad: slice always allocates
    Headers []Header
    
    // Good: array for common case, overflow to heap
    HeadersArray [8]Header
    HeadersExtra []Header
    HeaderCount  int
}

func (r *Request) AddHeader(h Header) {
    if r.HeaderCount < 8 {
        r.HeadersArray[r.HeaderCount] = h
    } else {
        r.HeadersExtra = append(r.HeadersExtra, h)
    }
    r.HeaderCount++
}

// Technique 3: Avoid interface{} in hot paths
// Bad: interface conversion allocates
func processBad(v any) {}

// Good: Use generics or concrete types
func processGood[T any](v T) {}
func processInt(v int) {}

// Technique 4: Pre-allocate slices with known size
func transform(input []int) []int {
    // Bad: grows dynamically
    var result []int
    for _, v := range input {
        result = append(result, v*2)
    }
    
    // Good: pre-allocated
    result := make([]int, 0, len(input))
    for _, v := range input {
        result = append(result, v*2)
    }
    return result
}
```

### Measuring Allocations
```go
// In benchmarks
func BenchmarkAllocs(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _ = myFunction()
    }
}
// Output: 1234 ns/op    256 B/op    4 allocs/op

// In tests
func TestAllocations(t *testing.T) {
    allocs := testing.AllocsPerRun(100, func() {
        _ = myFunction()
    })
    if allocs > 0 {
        t.Errorf("unexpected allocations: %f", allocs)
    }
}

// At runtime
var m runtime.MemStats
runtime.ReadMemStats(&m)
fmt.Printf("Allocs: %d, TotalAlloc: %d, Sys: %d\n", 
    m.Mallocs, m.TotalAlloc, m.Sys)
```

### Key Principles
- **Measure First:** Profile to find allocation hotspots
- **Reuse Buffers:** sync.Pool or pre-allocated slices
- **Avoid Interface{}:** Use generics or concrete types
- **Pre-allocate:** Use make() with capacity when size is known

---

## Summary: Phase 4 Competencies

After completing Phase 4, you can:

| Skill | Assessment |
|-------|------------|
| Write benchmarks | testing.B, sub-benchmarks, parallel |
| Use pprof | CPU, heap, goroutine profiles |
| Implement memory pools | sync.Pool, type-safe wrappers |
| Analyze escape behavior | gcflags, understand patterns |
| Reduce allocations | Builder, pre-allocation, pooling |
| Measure performance | benchstat, allocation tracking |

---

## Exercises Checklist

- [ ] **Exercise 4.1:** Optimized Buffer Pool
  - Implements: Type-safe buffer pool with sync.Pool
  - File: `pkg/perf/pool.go`
  - Test: `pkg/perf/pool_test.go`
  - Requirements:
    - Generic `Pool[T]` using sync.Pool
    - Type-safe Get/Put without assertions at call site
    - Reset function support
    - Benchmark showing allocation reduction

- [ ] **Exercise 4.2:** Zero-Allocation JSON Encoder
  - Implements: JSON encoding with minimal allocations
  - File: `pkg/perf/json.go`
  - Test: `pkg/perf/json_test.go`
  - Requirements:
    - Reuse bytes.Buffer via pool
    - Pre-size buffer based on typical payload
    - Benchmark comparing with standard json.Marshal
    - < 2 allocations per encode

- [ ] **Exercise 4.3:** Profiling Integration
  - Implements: pprof endpoints for GLR server
  - File: `pkg/perf/profile.go`
  - Test: Manual verification with pprof tools
  - Requirements:
    - Enable CPU, heap, goroutine profiles
    - Add /debug/pprof/ endpoints
    - Document profiling workflow
    - Optional: Add custom pprof labels

---

## Next Phase Preview

**Phase 5: Systems Programming** will cover:
- unsafe package and memory layout
- Memory-mapped files (mmap)
- CGO and calling C libraries
- Low-level optimization techniques

The performance optimization skills from Phase 4 provide the foundation for understanding when (and why) to reach for systems programming techniques in Phase 5.

