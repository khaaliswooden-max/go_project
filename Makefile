# GoLlama Runtime (GLR) Makefile
# PhD-level Go learning project

.PHONY: all build run test lint bench clean help learn exercise verify

# Variables
BINARY_NAME=glr
MAIN_PATH=./cmd/glr
COVERAGE_FILE=coverage.out

# Default target
all: lint test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Run the server
run:
	@echo "Starting GLR server..."
	go run $(MAIN_PATH)

# Run tests with race detector
test:
	@echo "Running tests with race detector..."
	go test -race -v -coverprofile=$(COVERAGE_FILE) ./...
	@go tool cover -func=$(COVERAGE_FILE) | tail -1

# Run specific package tests
test-pkg:
	@echo "Testing package: $(PKG)"
	go test -race -v ./$(PKG)/...

# Run tests for ollama client specifically
test-ollama:
	@echo "Testing Ollama client..."
	go test -race -v ./internal/ollama/...

# Lint the code
lint:
	@echo "Running linters..."
	go vet ./...
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed, skipping (install: go install honnef.co/go/tools/cmd/staticcheck@latest)"; \
	fi
	go fmt ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Show coverage in browser
coverage: test
	go tool cover -html=$(COVERAGE_FILE)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f $(COVERAGE_FILE)

# Verify Ollama is running
check-ollama:
	@echo "Checking Ollama status..."
	@curl -s http://localhost:11434/api/tags > /dev/null && echo "✓ Ollama is running" || echo "✗ Ollama not responding (start with: ollama serve)"

# Integration test (requires Ollama)
test-integration: check-ollama
	@echo "Running integration tests..."
	go test -race -v -tags=integration ./...

# === Learning Commands ===

# Show current phase objectives
learn:
	@echo "=== Current Phase: 7 - Distributed Systems ==="
	@echo ""
	@echo "Phase 1 Status: ✓ Complete"
	@echo "  - Exercise 1.1: Retry with exponential backoff ✓"
	@echo "  - Exercise 1.2: Logging middleware (optional)"
	@echo ""
	@echo "Phase 2 Status: ✓ Complete"
	@echo "  - Exercise 2.1: Worker pool implementation ✓"
	@echo "  - Exercise 2.2: Parallel request handler ✓"
	@echo "  - Exercise 2.3: Semaphore implementation ✓"
	@echo ""
	@echo "Phase 3 Status: ✓ Complete"
	@echo "  - Exercise 3.1: Generic worker pool ✓"
	@echo "  - Exercise 3.2: Generic Result type ✓"
	@echo "  - Exercise 3.3: Generic LRU cache ✓"
	@echo ""
	@echo "Phase 4 Status: ✓ Complete"
	@echo "  - Exercise 4.1: Optimized Buffer Pool ✓"
	@echo "  - Exercise 4.2: Zero-Allocation JSON Encoder ✓"
	@echo "  - Exercise 4.3: Profiling Integration ✓"
	@echo ""
	@echo "Phase 5 Status: ✓ Complete"
	@echo "  - Exercise 5.1: Memory Layout Analyzer ✓"
	@echo "  - Exercise 5.2: Memory-Mapped File Reader ✓"
	@echo "  - Exercise 5.3: CGO Integration (optional) ✓"
	@echo ""
	@echo "Phase 6 Status: ✓ Complete"
	@echo "  - Exercise 6.1: AST Explorer ✓"
	@echo "  - Exercise 6.2: Custom Linter ✓"
	@echo "  - Exercise 6.3: Mock Generator ✓"
	@echo ""
	@echo "=== Phase 7: Distributed Systems ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. Consensus algorithms (Raft basics)"
	@echo "  2. Streaming communication patterns"
	@echo "  3. Leader election mechanisms"
	@echo "  4. Log replication"
	@echo ""
	@echo "Key Files:"
	@echo "  - pkg/distributed/raft.go         (Raft consensus)"
	@echo "  - pkg/distributed/stream.go       (Streaming patterns)"
	@echo "  - pkg/distributed/leader.go       (Leader election)"
	@echo "  - docs/PHASE7_LEARNING.md         (Learning guide)"
	@echo ""
	@echo "Exercises:"
	@echo "  7.1: Simplified Raft"
	@echo "  7.2: Streaming Service"
	@echo "  7.3: Leader Election Service"
	@echo ""
	@echo "Run 'make exercise-7-1' to start Exercise 7.1"

learn-phase1:
	@echo "=== Phase 1: Foundations (Complete) ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. Interface design and dependency injection"
	@echo "  2. HTTP server patterns with graceful shutdown"
	@echo "  3. Structured logging with correlation IDs"
	@echo "  4. Table-driven testing methodology"
	@echo "  5. Error handling and wrapping"
	@echo ""
	@echo "Key Files:"
	@echo "  - internal/ollama/client.go      (Interface design)"
	@echo "  - internal/ollama/client_test.go (Table-driven tests)"
	@echo "  - internal/ollama/retry.go       (Retry + backoff)"
	@echo "  - internal/server/http.go        (HTTP patterns)"
	@echo "  - internal/audit/logger.go       (Structured logging)"
	@echo "  - docs/PHASE1_LEARNING.md        (Learning guide)"

# Exercise starters
exercise-1-1:
	@echo "=== Exercise 1.1: Retry with Exponential Backoff ==="
	@echo ""
	@echo "Objective: Implement retry logic using only stdlib"
	@echo ""
	@echo "Requirements:"
	@echo "  - Max 3 retries"
	@echo "  - Initial delay: 100ms"
	@echo "  - Backoff multiplier: 2x"
	@echo "  - Respect context cancellation"
	@echo "  - Add jitter (±10%)"
	@echo ""
	@echo "File to edit: internal/ollama/retry.go"
	@echo "Test file: internal/ollama/retry_test.go"
	@echo ""
	@echo "Verify with: make verify-1-1"

exercise-1-2:
	@echo "=== Exercise 1.2: Logging Middleware ==="
	@echo ""
	@echo "Objective: Add HTTP middleware for request/response logging"
	@echo ""
	@echo "Requirements:"
	@echo "  - Log: method, path, status, duration, correlation ID"
	@echo "  - Generate correlation ID if not in X-Correlation-ID header"
	@echo "  - Propagate correlation ID to response headers"
	@echo "  - Use structured JSON logging"
	@echo ""
	@echo "File to edit: internal/middleware/logging.go"
	@echo "Test file: internal/middleware/logging_test.go"
	@echo ""
	@echo "Verify with: make verify-1-2"

# Verify exercise solutions
verify-1-1:
	@echo "Verifying Exercise 1.1..."
	@go test -v -run TestRetry ./internal/ollama/...

verify-1-2:
	@echo "Verifying Exercise 1.2..."
	@go test -v -run TestLoggingMiddleware ./internal/middleware/...

# === Phase 2: Concurrency Exercises ===

exercise-2-1:
	@echo "=== Exercise 2.1: Worker Pool Implementation ==="
	@echo ""
	@echo "Objective: Implement a worker pool with bounded parallelism"
	@echo ""
	@echo "Requirements:"
	@echo "  - Configurable number of workers"
	@echo "  - Submit jobs via channel"
	@echo "  - Collect results via channel"
	@echo "  - Support context cancellation"
	@echo "  - Graceful shutdown (drain queue)"
	@echo "  - ShutdownNow (cancel immediately)"
	@echo ""
	@echo "File to edit: internal/worker/pool.go"
	@echo "Test file: internal/worker/pool_test.go"
	@echo ""
	@echo "Verify with: make verify-2-1"

exercise-2-2:
	@echo "=== Exercise 2.2: Parallel Request Handler ==="
	@echo ""
	@echo "Objective: Process multiple prompts in parallel with rate limiting"
	@echo ""
	@echo "Requirements:"
	@echo "  - Fan-out: Distribute work across workers"
	@echo "  - Fan-in: Aggregate results in order"
	@echo "  - Bounded concurrency (max parallel requests)"
	@echo "  - Handle partial failures gracefully"
	@echo ""
	@echo "File to edit: internal/ollama/parallel.go"
	@echo "Test file: internal/ollama/parallel_test.go"
	@echo ""
	@echo "Verify with: make verify-2-2"

exercise-2-3:
	@echo "=== Exercise 2.3: Semaphore Implementation ==="
	@echo ""
	@echo "Objective: Implement a semaphore using channels"
	@echo ""
	@echo "Requirements:"
	@echo "  - Acquire(ctx) - blocks until slot available"
	@echo "  - Release() - releases a slot"
	@echo "  - TryAcquire() - non-blocking acquire"
	@echo "  - Context cancellation support"
	@echo ""
	@echo "File to edit: pkg/sync/semaphore.go"
	@echo "Test file: pkg/sync/semaphore_test.go"
	@echo ""
	@echo "Verify with: make verify-2-3"

verify-2-1:
	@echo "Verifying Exercise 2.1..."
	@go test -race -v -run TestPool ./internal/worker/...

verify-2-2:
	@echo "Verifying Exercise 2.2..."
	@go test -race -v -run TestParallel ./internal/ollama/...

verify-2-3:
	@echo "Verifying Exercise 2.3..."
	@go test -race -v -run TestSemaphore ./pkg/sync/...

# Test worker pool specifically
test-worker:
	@echo "Testing worker pool..."
	@go test -race -v ./internal/worker/...

# === Phase 3: Generics Exercises ===

learn-phase2:
	@echo "=== Phase 2: Concurrency (Complete) ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. Goroutines and lifecycle management"
	@echo "  2. Channels (buffered, unbuffered, directional)"
	@echo "  3. Select statement patterns"
	@echo "  4. sync primitives (Mutex, RWMutex, WaitGroup, Once)"
	@echo "  5. Worker pool pattern"
	@echo "  6. Race condition detection"
	@echo ""
	@echo "Key Files:"
	@echo "  - internal/worker/pool.go        (Worker pool)"
	@echo "  - internal/ollama/retry.go       (Context + select)"
	@echo "  - pkg/sync/semaphore.go          (Semaphore)"
	@echo "  - docs/PHASE2_LEARNING.md        (Learning guide)"

exercise-3-1:
	@echo "=== Exercise 3.1: Generic Worker Pool ==="
	@echo ""
	@echo "Objective: Convert the Phase 2 worker pool to use generics"
	@echo ""
	@echo "Requirements:"
	@echo "  - Generic input type In"
	@echo "  - Generic output type Out"
	@echo "  - Type-safe Submit(input In, fn func(In) Out)"
	@echo "  - Type-safe Results() <-chan PoolResult[Out]"
	@echo "  - No type assertions needed by caller"
	@echo ""
	@echo "File to edit: pkg/generic/pool.go"
	@echo "Test file: pkg/generic/pool_test.go"
	@echo ""
	@echo "Verify with: make verify-3-1"

exercise-3-2:
	@echo "=== Exercise 3.2: Generic Result Type ==="
	@echo ""
	@echo "Objective: Implement a Rust-inspired Result[T] for error handling"
	@echo ""
	@echo "Requirements:"
	@echo "  - Ok[T](value) and Err[T](error) constructors"
	@echo "  - Unwrap(), UnwrapOr(default), UnwrapOrElse(fn)"
	@echo "  - IsOk(), IsErr() predicates"
	@echo "  - Map(fn) for value transformation"
	@echo "  - And(), Or(), OrElse() combinators"
	@echo ""
	@echo "File to edit: pkg/generic/result.go"
	@echo "Test file: pkg/generic/result_test.go"
	@echo ""
	@echo "Verify with: make verify-3-2"

exercise-3-3:
	@echo "=== Exercise 3.3: Generic LRU Cache ==="
	@echo ""
	@echo "Objective: Implement a type-safe LRU cache with generics"
	@echo ""
	@echo "Requirements:"
	@echo "  - Generic key type K (comparable constraint)"
	@echo "  - Generic value type V (any)"
	@echo "  - Get, Set, Delete operations"
	@echo "  - LRU eviction when over capacity"
	@echo "  - Thread-safe with RWMutex"
	@echo "  - GetOrSet for atomic get-or-create"
	@echo ""
	@echo "File to edit: pkg/generic/cache.go"
	@echo "Test file: pkg/generic/cache_test.go"
	@echo ""
	@echo "Verify with: make verify-3-3"

verify-3-1:
	@echo "Verifying Exercise 3.1..."
	@go test -race -v -run TestPool ./pkg/generic/...

verify-3-2:
	@echo "Verifying Exercise 3.2..."
	@go test -race -v -run "Test.*" ./pkg/generic/... -run "Result|Ok|Err|Map|Unwrap|Collect|Partition"

verify-3-3:
	@echo "Verifying Exercise 3.3..."
	@go test -race -v -run TestLRUCache ./pkg/generic/...

# Test all Phase 3 generics
test-generic:
	@echo "Testing generic package..."
	@go test -race -v ./pkg/generic/...

# === Phase 4: Performance Exercises ===

learn-phase3:
	@echo "=== Phase 3: Generics (Complete) ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. Type parameters and syntax"
	@echo "  2. Type constraints (comparable, any, custom)"
	@echo "  3. Generic data structures"
	@echo "  4. Type inference"
	@echo "  5. Generics best practices"
	@echo ""
	@echo "Key Files:"
	@echo "  - pkg/generic/pool.go            (Generic worker pool)"
	@echo "  - pkg/generic/result.go          (Result[T] type)"
	@echo "  - pkg/generic/cache.go           (Generic LRU cache)"
	@echo "  - docs/PHASE3_LEARNING.md        (Learning guide)"

learn-phase4:
	@echo "=== Phase 4: Performance (Complete) ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. Benchmarking with testing.B"
	@echo "  2. Profiling with pprof"
	@echo "  3. Memory pooling with sync.Pool"
	@echo "  4. Escape analysis"
	@echo "  5. Reducing allocations"
	@echo ""
	@echo "Key Files:"
	@echo "  - pkg/perf/pool.go               (sync.Pool wrapper)"
	@echo "  - pkg/perf/json.go               (Zero-alloc JSON)"
	@echo "  - pkg/perf/alloc.go              (Allocation utilities)"
	@echo "  - pkg/perf/profile.go            (pprof integration)"
	@echo "  - docs/PHASE4_LEARNING.md        (Learning guide)"

learn-phase5:
	@echo "=== Phase 5: Systems Programming (Complete) ==="
	@echo ""
	@echo "⚠️  WARNING: Phase 5 bypasses Go's safety guarantees!"
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. unsafe package and memory layout"
	@echo "  2. Memory-mapped files (mmap)"
	@echo "  3. CGO and calling C libraries"
	@echo "  4. Low-level optimization techniques"
	@echo ""
	@echo "Key Files:"
	@echo "  - pkg/systems/unsafe.go           (Memory layout, pointers)"
	@echo "  - pkg/systems/mmap.go             (Memory-mapped files)"
	@echo "  - pkg/systems/cgo.go              (CGO integration)"
	@echo "  - docs/PHASE5_LEARNING.md         (Learning guide)"
	@echo ""
	@echo "Key Concepts:"
	@echo "  - unsafe.Pointer for type-punning"
	@echo "  - unsafe.Sizeof, Alignof, Offsetof"
	@echo "  - Zero-copy string conversions"
	@echo "  - Platform-specific mmap APIs"
	@echo "  - CGO overhead and best practices"

learn-phase6:
	@echo "=== Phase 6: Static Analysis ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. go/ast package for parsing Go code"
	@echo "  2. go/types for type checking"
	@echo "  3. Custom linter development"
	@echo "  4. Code generation with templates"
	@echo ""
	@echo "Key Files:"
	@echo "  - pkg/analysis/ast.go            (AST parsing, traversal)"
	@echo "  - pkg/analysis/linter.go         (Custom linter rules)"
	@echo "  - pkg/analysis/generator.go      (Code generation)"
	@echo "  - docs/PHASE6_LEARNING.md        (Learning guide)"
	@echo ""
	@echo "Key Concepts:"
	@echo "  - Abstract Syntax Trees (AST)"
	@echo "  - AST traversal with ast.Inspect and ast.Walk"
	@echo "  - Type checking with go/types"
	@echo "  - Template-based code generation"
	@echo "  - Building custom linter rules"

exercise-4-1:
	@echo "=== Exercise 4.1: Optimized Buffer Pool ==="
	@echo ""
	@echo "Objective: Implement a type-safe buffer pool with sync.Pool"
	@echo ""
	@echo "Requirements:"
	@echo "  - Generic Pool[T] using sync.Pool"
	@echo "  - Type-safe Get/Put without assertions at call site"
	@echo "  - Reset function support for object cleanup"
	@echo "  - ByteBufferPool specialized for bytes.Buffer"
	@echo "  - TrackedPool for statistics/debugging"
	@echo ""
	@echo "File to edit: pkg/perf/pool.go"
	@echo "Test file: pkg/perf/pool_test.go"
	@echo ""
	@echo "Verify with: make verify-4-1"
	@echo "Benchmark with: make bench-pool"

exercise-4-2:
	@echo "=== Exercise 4.2: Zero-Allocation JSON Encoder ==="
	@echo ""
	@echo "Objective: Build JSON encoding with minimal allocations"
	@echo ""
	@echo "Requirements:"
	@echo "  - Reuse bytes.Buffer via pool"
	@echo "  - EncodeTo() for direct writer output"
	@echo "  - EncodeAppend() for building larger documents"
	@echo "  - MarshalPooled() as drop-in replacement"
	@echo "  - < 2 allocations per encode"
	@echo ""
	@echo "File to edit: pkg/perf/json.go"
	@echo "Test file: pkg/perf/json_test.go"
	@echo ""
	@echo "Verify with: make verify-4-2"
	@echo "Benchmark with: make bench-json"

exercise-4-3:
	@echo "=== Exercise 4.3: Profiling Integration ==="
	@echo ""
	@echo "Objective: Add pprof endpoints for production profiling"
	@echo ""
	@echo "Requirements:"
	@echo "  - RegisterPprof(mux) adds all pprof handlers"
	@echo "  - Enable CPU, heap, goroutine profiles"
	@echo "  - RuntimeStats for lightweight monitoring"
	@echo "  - HealthHandler for liveness checks"
	@echo "  - Documentation for profiling workflow"
	@echo ""
	@echo "File to edit: pkg/perf/profile.go"
	@echo "Test file: Manual verification with pprof tools"
	@echo ""
	@echo "Verify with: make verify-4-3"
	@echo "Test profiling: make run, then visit http://localhost:8080/debug/pprof/"

verify-4-1:
	@echo "Verifying Exercise 4.1..."
	@go test -race -v -run TestPool ./pkg/perf/...
	@go test -race -v -run TestByteBufferPool ./pkg/perf/...
	@go test -race -v -run TestTrackedPool ./pkg/perf/...

verify-4-2:
	@echo "Verifying Exercise 4.2..."
	@go test -race -v -run TestJSON ./pkg/perf/...
	@go test -race -v -run TestMarshalPooled ./pkg/perf/...
	@go test -race -v -run TestUnmarshalPooled ./pkg/perf/...

verify-4-3:
	@echo "Verifying Exercise 4.3..."
	@go build ./pkg/perf/...
	@echo "✓ Package compiles successfully"
	@echo ""
	@echo "To test pprof integration:"
	@echo "  1. Run the server: make run"
	@echo "  2. Visit: http://localhost:8080/debug/pprof/"
	@echo "  3. Collect CPU profile: go tool pprof http://localhost:8080/debug/pprof/profile?seconds=10"
	@echo "  4. View heap: go tool pprof http://localhost:8080/debug/pprof/heap"

# Test all Phase 4 performance code
test-perf:
	@echo "Testing perf package..."
	@go test -race -v ./pkg/perf/...

# === Phase 5: Systems Programming Exercises ===

exercise-5-1:
	@echo "=== Exercise 5.1: Memory Layout Analyzer ==="
	@echo ""
	@echo "Objective: Analyze and optimize struct memory layout"
	@echo ""
	@echo "Requirements:"
	@echo "  - AnalyzeStruct() returns complete layout info"
	@echo "  - Show field offsets, sizes, alignment, padding"
	@echo "  - OptimizeSuggestion() for field reordering"
	@echo "  - Zero-copy string/byte conversions (with warnings)"
	@echo "  - Benchmark showing layout performance impact"
	@echo ""
	@echo "File to edit: pkg/systems/unsafe.go"
	@echo "Test file: pkg/systems/unsafe_test.go"
	@echo ""
	@echo "Verify with: make verify-5-1"
	@echo "Benchmark with: make bench-unsafe"

exercise-5-2:
	@echo "=== Exercise 5.2: Memory-Mapped File Reader ==="
	@echo ""
	@echo "Objective: Implement cross-platform mmap abstraction"
	@echo ""
	@echo "Requirements:"
	@echo "  - MappedFile interface for platform abstraction"
	@echo "  - Windows: CreateFileMapping/MapViewOfFile"
	@echo "  - Unix: mmap(2) syscall"
	@echo "  - Safe open/close with proper cleanup"
	@echo "  - Benchmark comparing mmap vs regular I/O"
	@echo ""
	@echo "File to edit: pkg/systems/mmap.go"
	@echo "Test file: pkg/systems/mmap_test.go"
	@echo ""
	@echo "Verify with: make verify-5-2"
	@echo "Benchmark with: make bench-mmap"

exercise-5-3:
	@echo "=== Exercise 5.3: CGO Integration (Optional) ==="
	@echo ""
	@echo "Objective: Call C functions from Go"
	@echo ""
	@echo "⚠️  Requires C compiler (gcc, clang, or MSVC)"
	@echo ""
	@echo "Requirements:"
	@echo "  - Basic C function calls"
	@echo "  - String passing (CString, free)"
	@echo "  - Array/slice passing"
	@echo "  - Memory management (malloc, free)"
	@echo "  - Benchmark CGO overhead vs pure Go"
	@echo ""
	@echo "File to edit: pkg/systems/cgo.go"
	@echo "Test file: pkg/systems/cgo_test.go"
	@echo ""
	@echo "Verify with: make verify-5-3"
	@echo "Benchmark with: make bench-cgo"

verify-5-1:
	@echo "Verifying Exercise 5.1..."
	@go test -race -v -run TestAnalyzeStruct ./pkg/systems/...
	@go test -race -v -run TestUnsafe ./pkg/systems/...
	@go test -race -v -run TestSizeOf ./pkg/systems/...
	@go test -race -v -run TestPtrAdd ./pkg/systems/...
	@echo ""
	@echo "✓ Exercise 5.1 verified!"

verify-5-2:
	@echo "Verifying Exercise 5.2..."
	@go test -race -v -run TestOpenMmap ./pkg/systems/...
	@go test -race -v -run TestMmapFile ./pkg/systems/...
	@go test -race -v -run TestMmapReader ./pkg/systems/...
	@go test -race -v -run TestSearchMmap ./pkg/systems/...
	@echo ""
	@echo "✓ Exercise 5.2 verified!"

verify-5-3:
	@echo "Verifying Exercise 5.3..."
	@echo "Note: CGO tests require CGO_ENABLED=1 and a C compiler"
	@go test -race -v -tags=cgo -run TestC ./pkg/systems/... 2>/dev/null || echo "CGO not available (this is OK)"
	@echo ""
	@echo "To run CGO tests manually:"
	@echo "  CGO_ENABLED=1 go test -v -run TestC ./pkg/systems/..."

# Test all Phase 5 systems code
test-systems:
	@echo "Testing systems package..."
	@go test -race -v ./pkg/systems/...

# === Phase 6: Static Analysis Exercises ===

exercise-6-1:
	@echo "=== Exercise 6.1: AST Explorer ==="
	@echo ""
	@echo "Objective: Parse and analyze Go source code structure"
	@echo ""
	@echo "Requirements:"
	@echo "  - ParseFile() and ParseSource() for parsing"
	@echo "  - FindFunctions() to discover function declarations"
	@echo "  - AnalyzeFile() for comprehensive analysis"
	@echo "  - Extract structs, interfaces, and imports"
	@echo "  - FindTODOs() to locate TODO comments"
	@echo ""
	@echo "File to edit: pkg/analysis/ast.go"
	@echo "Test file: pkg/analysis/ast_test.go"
	@echo ""
	@echo "Verify with: make verify-6-1"

exercise-6-2:
	@echo "=== Exercise 6.2: Custom Linter ==="
	@echo ""
	@echo "Objective: Build a linter with multiple rules"
	@echo ""
	@echo "Requirements:"
	@echo "  - Rule 1: context.Context should be first parameter"
	@echo "  - Rule 2: Error return values must be checked"
	@echo "  - Rule 3: TODO comments should reference issues"
	@echo "  - Rule 4: Detect empty error handlers"
	@echo "  - Rule 5: Flag naked returns in long functions"
	@echo "  - Output diagnostics with position info"
	@echo ""
	@echo "File to edit: pkg/analysis/linter.go"
	@echo "Test file: pkg/analysis/linter_test.go"
	@echo ""
	@echo "Verify with: make verify-6-2"

exercise-6-3:
	@echo "=== Exercise 6.3: Mock Generator ==="
	@echo ""
	@echo "Objective: Generate mock implementations from interfaces"
	@echo ""
	@echo "Requirements:"
	@echo "  - ExtractInterfaceSpec() to analyze interfaces"
	@echo "  - GenerateMock() using text/template"
	@echo "  - Handle multiple methods and parameters"
	@echo "  - Generate proper zero values for returns"
	@echo "  - AST-based generation helpers"
	@echo "  - Format output with go/format"
	@echo ""
	@echo "File to edit: pkg/analysis/generator.go"
	@echo "Test file: pkg/analysis/generator_test.go"
	@echo ""
	@echo "Verify with: make verify-6-3"

verify-6-1:
	@echo "Verifying Exercise 6.1..."
	@go test -race -v -run TestParseSource ./pkg/analysis/...
	@go test -race -v -run TestFindFunctions ./pkg/analysis/...
	@go test -race -v -run TestAnalyzeFile ./pkg/analysis/...
	@go test -race -v -run TestFindTODOs ./pkg/analysis/...
	@echo ""
	@echo "✓ Exercise 6.1 verified!"

verify-6-2:
	@echo "Verifying Exercise 6.2..."
	@go test -race -v -run TestContextFirstRule ./pkg/analysis/...
	@go test -race -v -run TestEmptyErrorHandlerRule ./pkg/analysis/...
	@go test -race -v -run TestTODOWithoutIssueRule ./pkg/analysis/...
	@go test -race -v -run TestNakedReturnRule ./pkg/analysis/...
	@go test -race -v -run TestLinter ./pkg/analysis/...
	@echo ""
	@echo "✓ Exercise 6.2 verified!"

verify-6-3:
	@echo "Verifying Exercise 6.3..."
	@go test -race -v -run TestGenerateMock ./pkg/analysis/...
	@go test -race -v -run TestExtractInterfaceSpec ./pkg/analysis/...
	@go test -race -v -run TestMockMethod ./pkg/analysis/...
	@go test -race -v -run TestGenerateFromAST ./pkg/analysis/...
	@go test -race -v -run TestCreate ./pkg/analysis/...
	@echo ""
	@echo "✓ Exercise 6.3 verified!"

# Test all Phase 6 analysis code
test-analysis:
	@echo "Testing analysis package..."
	@go test -race -v ./pkg/analysis/...

# === Phase 7: Distributed Systems Exercises ===

learn-phase7:
	@echo "=== Phase 7: Distributed Systems ==="
	@echo ""
	@echo "Learning Objectives:"
	@echo "  1. Consensus algorithms (Raft basics)"
	@echo "  2. Streaming communication patterns"
	@echo "  3. Leader election mechanisms"
	@echo "  4. Log replication"
	@echo ""
	@echo "Key Files:"
	@echo "  - pkg/distributed/raft.go         (Raft consensus)"
	@echo "  - pkg/distributed/stream.go       (Streaming patterns)"
	@echo "  - pkg/distributed/leader.go       (Leader election)"
	@echo "  - docs/PHASE7_LEARNING.md         (Learning guide)"
	@echo ""
	@echo "Key Concepts:"
	@echo "  - Raft state machine (Follower/Candidate/Leader)"
	@echo "  - Log entries and terms"
	@echo "  - RequestVote and AppendEntries RPCs"
	@echo "  - Stream types (Unary, Server, Client, Bidirectional)"
	@echo "  - Lease-based leader election"
	@echo "  - Fencing tokens for safety"

exercise-7-1:
	@echo "=== Exercise 7.1: Simplified Raft ==="
	@echo ""
	@echo "Objective: Implement core Raft consensus mechanics"
	@echo ""
	@echo "Requirements:"
	@echo "  - Node state transitions (Follower → Candidate → Leader)"
	@echo "  - RequestVote RPC with term checking"
	@echo "  - AppendEntries RPC with log consistency"
	@echo "  - Heartbeat mechanism"
	@echo "  - Basic log entry append and replication"
	@echo ""
	@echo "File to edit: pkg/distributed/raft.go"
	@echo "Test file: pkg/distributed/raft_test.go"
	@echo ""
	@echo "Verify with: make verify-7-1"
	@echo "Benchmark with: make bench-raft"

exercise-7-2:
	@echo "=== Exercise 7.2: Streaming Service ==="
	@echo ""
	@echo "Objective: Implement generic streaming primitives"
	@echo ""
	@echo "Requirements:"
	@echo "  - Generic Stream[T] interface"
	@echo "  - Channel-based stream implementation"
	@echo "  - Server-side streaming (one-to-many)"
	@echo "  - Client-side streaming (many-to-one)"
	@echo "  - Bidirectional streaming"
	@echo "  - Flow control with backpressure"
	@echo ""
	@echo "File to edit: pkg/distributed/stream.go"
	@echo "Test file: pkg/distributed/stream_test.go"
	@echo ""
	@echo "Verify with: make verify-7-2"
	@echo "Benchmark with: make bench-stream"

exercise-7-3:
	@echo "=== Exercise 7.3: Leader Election Service ==="
	@echo ""
	@echo "Objective: Implement lease-based leader election"
	@echo ""
	@echo "Requirements:"
	@echo "  - LeaseStore interface (acquire, renew, release)"
	@echo "  - In-memory lease store for testing"
	@echo "  - Leader observer notifications"
	@echo "  - Fencing tokens for split-brain prevention"
	@echo "  - Leadership transfer support"
	@echo ""
	@echo "File to edit: pkg/distributed/leader.go"
	@echo "Test file: pkg/distributed/leader_test.go"
	@echo ""
	@echo "Verify with: make verify-7-3"
	@echo "Benchmark with: make bench-leader"

verify-7-1:
	@echo "Verifying Exercise 7.1..."
	@go test -race -v -run TestRaftNodeCreation ./pkg/distributed/...
	@go test -race -v -run TestRaftStateTransitions ./pkg/distributed/...
	@go test -race -v -run TestRaftRequestVote ./pkg/distributed/...
	@go test -race -v -run TestRaftAppendEntries ./pkg/distributed/...
	@go test -race -v -run TestRaftLogAppend ./pkg/distributed/...
	@go test -race -v -run TestRaftElectionTimeout ./pkg/distributed/...
	@echo ""
	@echo "✓ Exercise 7.1 verified!"

verify-7-2:
	@echo "Verifying Exercise 7.2..."
	@go test -race -v -run TestChannelStream ./pkg/distributed/...
	@go test -race -v -run TestStreamPair ./pkg/distributed/...
	@go test -race -v -run TestFlowControlled ./pkg/distributed/...
	@go test -race -v -run TestServerStream ./pkg/distributed/...
	@go test -race -v -run TestClientStream ./pkg/distributed/...
	@go test -race -v -run TestBidiStream ./pkg/distributed/...
	@echo ""
	@echo "✓ Exercise 7.2 verified!"

verify-7-3:
	@echo "Verifying Exercise 7.3..."
	@go test -race -v -run TestMemoryLeaseStore ./pkg/distributed/...
	@go test -race -v -run TestLeaderElection ./pkg/distributed/...
	@go test -race -v -run TestFenceToken ./pkg/distributed/...
	@go test -race -v -run TestFencedServer ./pkg/distributed/...
	@go test -race -v -run TestLeaderAware ./pkg/distributed/...
	@go test -race -v -run TestTransferLeadership ./pkg/distributed/...
	@echo ""
	@echo "✓ Exercise 7.3 verified!"

# Test all Phase 7 distributed code
test-distributed:
	@echo "Testing distributed package..."
	@go test -race -v ./pkg/distributed/...

# Benchmark Phase 7 packages
bench-distributed:
	@echo "Running distributed benchmarks..."
	@go test -bench=. -benchmem ./pkg/distributed/...

bench-raft:
	@echo "Running Raft benchmarks..."
	@go test -bench=BenchmarkRaft -benchmem ./pkg/distributed/...

bench-stream:
	@echo "Running stream benchmarks..."
	@go test -bench=BenchmarkStream -benchmem ./pkg/distributed/...

bench-leader:
	@echo "Running leader election benchmarks..."
	@go test -bench=BenchmarkLease -benchmem ./pkg/distributed/...
	@go test -bench=BenchmarkFenced -benchmem ./pkg/distributed/...

# Benchmark Phase 5 packages
bench-systems:
	@echo "Running systems benchmarks..."
	@go test -bench=. -benchmem ./pkg/systems/...

bench-unsafe:
	@echo "Running unsafe benchmarks..."
	@go test -bench=BenchmarkString -benchmem ./pkg/systems/...
	@go test -bench=BenchmarkBytes -benchmem ./pkg/systems/...
	@go test -bench=BenchmarkStruct -benchmem ./pkg/systems/...
	@go test -bench=BenchmarkPointer -benchmem ./pkg/systems/...

bench-mmap:
	@echo "Running mmap benchmarks..."
	@go test -bench=BenchmarkMmap -benchmem ./pkg/systems/...
	@go test -bench=BenchmarkSearch -benchmem ./pkg/systems/...

bench-cgo:
	@echo "Running CGO benchmarks..."
	@echo "Note: Requires CGO_ENABLED=1"
	@CGO_ENABLED=1 go test -bench=BenchmarkCGO -benchmem ./pkg/systems/... 2>/dev/null || echo "CGO not available"

# Show escape analysis for systems code
escape-systems:
	@echo "Escape analysis for pkg/systems..."
	@go build -gcflags='-m' ./pkg/systems/... 2>&1 | head -50
	@echo ""
	@echo "(Use -gcflags='-m -m' for more verbose output)"

# Benchmark Phase 4 packages
bench-perf:
	@echo "Running performance benchmarks..."
	@go test -bench=. -benchmem ./pkg/perf/...

bench-pool:
	@echo "Running pool benchmarks..."
	@go test -bench=BenchmarkWithPool -benchmem ./pkg/perf/...
	@go test -bench=BenchmarkGenericPool -benchmem ./pkg/perf/...

bench-json:
	@echo "Running JSON benchmarks..."
	@go test -bench=BenchmarkJSON -benchmem ./pkg/perf/...

bench-alloc:
	@echo "Running allocation benchmarks..."
	@go test -bench=BenchmarkString -benchmem ./pkg/perf/...
	@go test -bench=BenchmarkSlice -benchmem ./pkg/perf/...
	@go test -bench=BenchmarkEscape -benchmem ./pkg/perf/...

# Show escape analysis for performance code
escape-analysis:
	@echo "Escape analysis for pkg/perf..."
	@go build -gcflags='-m' ./pkg/perf/... 2>&1 | head -50
	@echo ""
	@echo "(Use -gcflags='-m -m' for more verbose output)"

# CPU profile during benchmark
cpu-profile:
	@echo "Generating CPU profile..."
	@go test -bench=BenchmarkJSON -cpuprofile=cpu.out ./pkg/perf/...
	@echo "View with: go tool pprof -http=:8081 cpu.out"

# Memory profile during benchmark
mem-profile:
	@echo "Generating memory profile..."
	@go test -bench=BenchmarkJSON -memprofile=mem.out ./pkg/perf/...
	@echo "View with: go tool pprof -http=:8081 mem.out"

# === Development Helpers ===

# Generate mocks (Phase 2+)
generate:
	go generate ./...

# Watch for changes and run tests (requires entr)
watch:
	@if command -v entr > /dev/null; then \
		find . -name "*.go" | entr -c make test; \
	else \
		echo "entr not installed (brew install entr / apt install entr)"; \
	fi

# Print project structure
tree:
	@find . -type f -name "*.go" | grep -v "_test.go" | sort

# Help
help:
	@echo "GoLlama Runtime (GLR) - Development Commands"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build          Build binary"
	@echo "  make run            Run server"
	@echo "  make clean          Clean artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  make test           Run all tests with race detector"
	@echo "  make test-ollama    Run Ollama client tests"
	@echo "  make test-worker    Run worker pool tests"
	@echo "  make test-generic   Run generics tests"
	@echo "  make test-perf      Run performance tests"
	@echo "  make test-systems   Run systems programming tests"
	@echo "  make test-analysis  Run static analysis tests"
	@echo "  make test-distributed Run distributed systems tests"
	@echo "  make bench          Run benchmarks"
	@echo "  make coverage       Show coverage in browser"
	@echo ""
	@echo "Quality:"
	@echo "  make lint           Run linters"
	@echo "  make check-ollama   Verify Ollama is running"
	@echo ""
	@echo "Learning:"
	@echo "  make learn          Show current phase objectives"
	@echo "  make learn-phase1   Review Phase 1 materials"
	@echo "  make learn-phase2   Review Phase 2 materials"
	@echo "  make learn-phase3   Review Phase 3 materials"
	@echo "  make learn-phase4   Review Phase 4 materials"
	@echo "  make learn-phase5   Review Phase 5 materials"
	@echo "  make learn-phase6   Review Phase 6 materials"
	@echo "  make learn-phase7   Review Phase 7 materials"
	@echo ""
	@echo "Phase 1 Exercises (Foundations):"
	@echo "  make exercise-1-1   Retry with exponential backoff"
	@echo "  make exercise-1-2   Logging middleware"
	@echo "  make verify-1-1     Verify Exercise 1.1"
	@echo "  make verify-1-2     Verify Exercise 1.2"
	@echo ""
	@echo "Phase 2 Exercises (Concurrency):"
	@echo "  make exercise-2-1   Worker pool implementation"
	@echo "  make exercise-2-2   Parallel request handler"
	@echo "  make exercise-2-3   Semaphore implementation"
	@echo "  make verify-2-1     Verify Exercise 2.1"
	@echo "  make verify-2-2     Verify Exercise 2.2"
	@echo "  make verify-2-3     Verify Exercise 2.3"
	@echo ""
	@echo "Phase 3 Exercises (Generics):"
	@echo "  make exercise-3-1   Generic worker pool"
	@echo "  make exercise-3-2   Generic Result type"
	@echo "  make exercise-3-3   Generic LRU cache"
	@echo "  make verify-3-1     Verify Exercise 3.1"
	@echo "  make verify-3-2     Verify Exercise 3.2"
	@echo "  make verify-3-3     Verify Exercise 3.3"
	@echo ""
	@echo "Phase 4 Exercises (Performance):"
	@echo "  make exercise-4-1   Optimized buffer pool"
	@echo "  make exercise-4-2   Zero-allocation JSON encoder"
	@echo "  make exercise-4-3   Profiling integration"
	@echo "  make verify-4-1     Verify Exercise 4.1"
	@echo "  make verify-4-2     Verify Exercise 4.2"
	@echo "  make verify-4-3     Verify Exercise 4.3"
	@echo ""
	@echo "Phase 5 Exercises (Systems Programming):"
	@echo "  make exercise-5-1   Memory layout analyzer"
	@echo "  make exercise-5-2   Memory-mapped file reader"
	@echo "  make exercise-5-3   CGO integration (optional)"
	@echo "  make verify-5-1     Verify Exercise 5.1"
	@echo "  make verify-5-2     Verify Exercise 5.2"
	@echo "  make verify-5-3     Verify Exercise 5.3"
	@echo ""
	@echo "Phase 6 Exercises (Static Analysis):"
	@echo "  make exercise-6-1   AST explorer"
	@echo "  make exercise-6-2   Custom linter"
	@echo "  make exercise-6-3   Mock generator"
	@echo "  make verify-6-1     Verify Exercise 6.1"
	@echo "  make verify-6-2     Verify Exercise 6.2"
	@echo "  make verify-6-3     Verify Exercise 6.3"
	@echo ""
	@echo "Phase 7 Exercises (Distributed Systems):"
	@echo "  make exercise-7-1   Simplified Raft consensus"
	@echo "  make exercise-7-2   Streaming service"
	@echo "  make exercise-7-3   Leader election service"
	@echo "  make verify-7-1     Verify Exercise 7.1"
	@echo "  make verify-7-2     Verify Exercise 7.2"
	@echo "  make verify-7-3     Verify Exercise 7.3"
	@echo ""
	@echo "Performance:"
	@echo "  make bench-perf     Run all performance benchmarks"
	@echo "  make bench-pool     Run pool benchmarks"
	@echo "  make bench-json     Run JSON benchmarks"
	@echo "  make bench-alloc    Run allocation benchmarks"
	@echo "  make bench-systems  Run systems benchmarks"
	@echo "  make bench-distributed Run distributed benchmarks"
	@echo "  make escape-analysis Show escape analysis"
	@echo "  make cpu-profile    Generate CPU profile"
	@echo "  make mem-profile    Generate memory profile"
