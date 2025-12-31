# Phase 2: Concurrency - Learning Guide

This document covers Go's concurrency primitives and patterns. Phase 2 builds on the foundations from Phase 1 (interfaces, testing, error handling) to implement safe concurrent code.

---

## Prerequisites

Before starting Phase 2, ensure you've completed:
- [x] Exercise 1.1: Retry with exponential backoff
- [ ] Exercise 1.2: Logging middleware (optional for Phase 2)

---

## 1. Goroutines

**Files:** `internal/worker/pool.go`

### Concept
Goroutines are lightweight threads managed by the Go runtime. They're cheap to create (2KB stack) and scheduled cooperatively.

### What You'll Learn
```go
// Starting a goroutine - use 'go' keyword
go func() {
    // This runs concurrently
    processItem(item)
}()

// LEARN: Goroutines don't return values directly.
// Use channels or shared state (with synchronization) to communicate.

// WARNING: Don't forget to synchronize!
// Goroutines can outlive the function that created them.
```

### Key Principles
- **Cheap to Create:** Unlike OS threads, goroutines are very lightweight
- **No Direct Return:** Use channels to get results from goroutines
- **Lifecycle Management:** Always ensure goroutines can exit cleanly
- **Context Propagation:** Pass context for cancellation signals

### Common Mistakes
```go
// WRONG: Loop variable capture
for i := range items {
    go process(items[i]) // Race condition!
}

// CORRECT: Pass as parameter
for i := range items {
    go func(item Item) {
        process(item)
    }(items[i])
}

// ALSO CORRECT (Go 1.22+): Loop variables are per-iteration
for _, item := range items {
    go process(item) // Safe in Go 1.22+
}
```

---

## 2. Channels

**Files:** `internal/worker/pool.go`, `internal/ollama/client.go` (streaming)

### Concept
Channels are Go's primary synchronization primitive. They enable safe communication between goroutines following the principle: "Don't communicate by sharing memory; share memory by communicating."

### What You'll Learn
```go
// Unbuffered channel - synchronous communication
ch := make(chan int)

// Buffered channel - async up to capacity
buffered := make(chan int, 10)

// Send and receive
ch <- value    // Send (blocks until received)
val := <-ch    // Receive (blocks until value available)

// Close channel to signal completion
close(ch)

// Range over channel until closed
for item := range ch {
    process(item)
}
```

### Channel Patterns
```go
// Pattern 1: Fan-out (one producer, multiple consumers)
jobs := make(chan Job, 100)
for w := 1; w <= numWorkers; w++ {
    go worker(w, jobs)
}

// Pattern 2: Fan-in (multiple producers, one consumer)
results := make(chan Result)
go producer1(results)
go producer2(results)
for result := range results {
    aggregate(result)
}

// Pattern 3: Pipeline
func stage1(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * 2
        }
    }()
    return out
}
```

### Key Principles
- **Close by Sender:** Only the sender should close a channel
- **Range Until Closed:** Use `for range` to drain channels
- **Buffering Trade-offs:** Unbuffered = synchronization, Buffered = throughput
- **Nil Channel Blocks:** Reading/writing nil channel blocks forever

---

## 3. Select Statement

**Files:** `internal/ollama/retry.go`, `internal/worker/pool.go`

### Concept
The `select` statement lets a goroutine wait on multiple channel operations. It's like a `switch` for channels.

### What You'll Learn
```go
// Wait for multiple channels
select {
case msg := <-messages:
    fmt.Println("Received:", msg)
case sig := <-signals:
    fmt.Println("Signal:", sig)
    return
case <-time.After(timeout):
    fmt.Println("Timeout!")
    return
}

// Non-blocking check with default
select {
case msg := <-messages:
    process(msg)
default:
    // No message available, don't block
}

// Common pattern: Context cancellation + work
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-workDone:
    return result
}
```

### Key Principles
- **Random Selection:** If multiple cases ready, one is chosen at random
- **Default Case:** Makes select non-blocking
- **Timeout Pattern:** Use `time.After` for timeouts
- **Context Integration:** Always include `ctx.Done()` case

---

## 4. sync Package Primitives

**Files:** `internal/worker/pool.go`, `internal/cache/lru.go` (Phase 4)

### Concept
When you must share memory between goroutines, use sync primitives to prevent races.

### Mutex (Mutual Exclusion)
```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.value
}
```

### RWMutex (Read-Write Mutex)
```go
type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock()         // Multiple readers OK
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()          // Exclusive for write
    defer c.mu.Unlock()
    c.data[key] = value
}
```

### WaitGroup
```go
var wg sync.WaitGroup

for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}

wg.Wait() // Block until all done
```

### Once (Singleton/Lazy Init)
```go
var (
    instance *DB
    once     sync.Once
)

func GetDB() *DB {
    once.Do(func() {
        instance = &DB{} // Only runs once, even with concurrent calls
    })
    return instance
}
```

### Key Principles
- **Defer Unlock:** Always use `defer` to ensure unlock
- **RWMutex for Reads:** Use when reads >> writes
- **WaitGroup for Fan-Out:** Track completion of multiple goroutines
- **Once for Init:** Thread-safe lazy initialization

---

## 5. Worker Pool Pattern

**Files:** `internal/worker/pool.go`

### Concept
A worker pool limits concurrent operations to prevent resource exhaustion while maximizing throughput.

### What You'll Learn
```go
// Worker pool with bounded parallelism
type Pool struct {
    workers int
    jobs    chan Job
    results chan Result
    wg      sync.WaitGroup
}

func NewPool(workers int) *Pool {
    p := &Pool{
        workers: workers,
        jobs:    make(chan Job, workers*2),  // Buffer for efficiency
        results: make(chan Result, workers*2),
    }
    p.start()
    return p
}

func (p *Pool) start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go func(workerID int) {
            defer p.wg.Done()
            for job := range p.jobs {
                result := process(job)
                p.results <- result
            }
        }(i)
    }
}

func (p *Pool) Submit(job Job) {
    p.jobs <- job
}

func (p *Pool) Shutdown() {
    close(p.jobs)    // Signal workers to stop
    p.wg.Wait()      // Wait for completion
    close(p.results) // Safe to close now
}
```

### Key Principles
- **Bounded Concurrency:** Limit workers to prevent resource exhaustion
- **Graceful Shutdown:** Close jobs channel, wait for workers, then close results
- **Buffer Sizing:** `workers * 2` is a reasonable starting point
- **Context Support:** Add context for cancellation

---

## 6. Race Detection

**Files:** All test files

### Concept
Data races occur when two goroutines access the same memory concurrently and at least one is a write. Go's race detector finds these bugs.

### What You'll Learn
```bash
# Run tests with race detector
go test -race ./...

# Run binary with race detection
go run -race main.go

# Build with race detection (for testing only - significant overhead)
go build -race
```

### Common Race Patterns
```go
// Race: shared counter without synchronization
var counter int
go func() { counter++ }()
go func() { counter++ }()

// Race: map access
m := make(map[string]int)
go func() { m["key"] = 1 }()
go func() { _ = m["key"] }()

// Race: slice append
var slice []int
go func() { slice = append(slice, 1) }()
go func() { slice = append(slice, 2) }()
```

### Key Principles
- **Always Test with -race:** Make it part of CI
- **No False Positives:** Race detector reports are real bugs
- **Performance Impact:** ~10x slower, ~5-10x more memory (testing only)
- **Fix Immediately:** Races cause undefined behavior

---

## Summary: Phase 2 Competencies

After completing Phase 2, you can:

| Skill | Assessment |
|-------|------------|
| Launch goroutines safely | Proper variable capture, lifecycle management |
| Use channels effectively | Fan-out, fan-in, pipelines |
| Apply select patterns | Timeouts, cancellation, non-blocking |
| Synchronize with sync pkg | Mutex, RWMutex, WaitGroup, Once |
| Implement worker pools | Bounded parallelism, graceful shutdown |
| Detect race conditions | go test -race in CI |

---

## Exercises Checklist

- [x] **Exercise 2.1:** Worker Pool Implementation
  - Implements: goroutines, channels, WaitGroup, graceful shutdown
  - File: `internal/worker/pool.go`
  - Test: `internal/worker/pool_test.go`
  - Requirements:
    - Configurable number of workers
    - Submit jobs via channel
    - Collect results via channel
    - Support context cancellation
    - Graceful shutdown

- [x] **Exercise 2.2:** Parallel Request Handler
  - Implements: Fan-out/fan-in, bounded concurrency
  - File: `internal/ollama/parallel.go`
  - Test: `internal/ollama/parallel_test.go`
  - Requirements:
    - Process multiple prompts in parallel
    - Limit concurrent requests (rate limiting)
    - Aggregate results in order
    - Handle partial failures

- [x] **Exercise 2.3:** Semaphore Implementation
  - Implements: Buffered channels as semaphores
  - File: `pkg/sync/semaphore.go`
  - Test: `pkg/sync/semaphore_test.go`
  - Requirements:
    - Acquire/Release semantics
    - TryAcquire (non-blocking)
    - Context-aware Acquire

---

## Next Phase Preview

**Phase 3: Generics** will cover:
- Type parameters and constraints
- Generic data structures
- Type inference
- Interface constraints

The concurrency patterns you've learned in Phase 2 will be enhanced with type-safe generic implementations in Phase 3.

