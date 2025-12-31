# Phase 1: Foundations - What You've Learned

This document tracks the techniques and concepts covered in Phase 1 of the GLR project. Each section maps to specific files and connects to real-world applications.

---

## 1. Interface-Based Design

**Files:** `internal/ollama/client.go`

### Concept
Interfaces in Go define behavior, not data. They enable:
- Dependency injection for testability
- Loose coupling between components
- Easy mocking in tests

### What You Learned
```go
// Define interfaces at point of USE, not implementation
type Generator interface {
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}

// Return interface from constructor, not concrete type
func New(cfg Config) (Client, error) { ... }
```

### Key Principles
- **Interface Segregation Principle (ISP):** Small, focused interfaces are better than large ones
- **Implicit Implementation:** Types implement interfaces automatically if they have the right methods
- **Testability:** Interfaces enable mock implementations without dependency injection frameworks

### Zuup Integration
- Veyra tool interfaces follow this pattern
- Enables hot-swapping of inference backends (Ollama → vLLM → TensorRT)

---

## 2. Table-Driven Tests

**Files:** `internal/ollama/client_test.go`

### Concept
A single test function with a slice of test cases that covers all code paths.

### What You Learned
```go
tests := []struct {
    name    string           // Descriptive name
    input   SomeInput        // Test inputs
    want    ExpectedOutput   // Expected result
    wantErr bool             // Whether error expected
}{
    {name: "valid input", input: ..., want: ..., wantErr: false},
    {name: "missing field", input: ..., wantErr: true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := FunctionUnderTest(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
        // compare got vs tt.want
    })
}
```

### Key Principles
- **Exhaustive Coverage:** Each row tests one scenario
- **Self-Documenting:** Test names describe expected behavior
- **Easy Extension:** Adding cases is just adding rows
- **Subtest Isolation:** `t.Run()` creates independent subtests

### Industry Standard
- This pattern is used in the Go standard library
- Required at Google, Microsoft, and most tech companies
- Enables running specific tests: `go test -run TestX/specific_case`

---

## 3. Error Handling Patterns

**Files:** `pkg/errors/errors.go`, throughout codebase

### Concept
Go uses explicit error returns instead of exceptions. Proper error handling includes:
- Sentinel errors for programmatic checking
- Error wrapping for context
- Error codes for API responses

### What You Learned
```go
// Sentinel errors
var ErrNotFound = errors.New("not found")

// Error wrapping preserves the chain
return fmt.Errorf("operation failed: %w", err)

// Checking wrapped errors
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

### Key Principles
- **Never Ignore Errors:** If intentionally ignoring, use `_ = err` with a comment
- **Add Context:** Each layer should add relevant context
- **Preserve Original:** Use `%w` to allow `errors.Is()` matching
- **Fail Fast:** Validate inputs early, return errors immediately

### Compliance Application
- Audit logs include error codes for traceability
- Error categorization enables retry logic
- Consistent error responses aid debugging

---

## 4. HTTP Server Patterns

**Files:** `internal/server/http.go`

### Concept
Production HTTP servers require:
- Graceful shutdown (drain in-flight requests)
- Appropriate timeouts (prevent resource exhaustion)
- Middleware chains (cross-cutting concerns)
- Proper error responses

### What You Learned
```go
// Graceful shutdown pattern
srv := &http.Server{
    Addr:         ":8080",
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 60 * time.Second,
}

go srv.ListenAndServe()

// Wait for shutdown signal
<-quit
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
srv.Shutdown(ctx)  // Drains in-flight requests
```

### Key Principles
- **Timeouts Are Required:** Prevent slow clients from exhausting resources
- **Graceful Shutdown:** Never kill connections mid-request
- **Middleware Pattern:** Wrap handlers to add logging, auth, recovery
- **ResponseWriter Wrapping:** Capture status codes for logging

### Real-World Application
- Load balancers expect graceful shutdown for rolling deploys
- Timeout tuning prevents cascading failures
- Middleware enables observability without code changes

---

## 5. Structured Logging

**Files:** `internal/audit/logger.go`, `cmd/glr/main.go`

### Concept
Structured logging outputs machine-parseable formats (JSON) with consistent fields, enabling:
- Log aggregation and search (ELK, Datadog, CloudWatch)
- Automated alerting on patterns
- Compliance audit trails

### What You Learned
```go
// slog (Go 1.21+) for structured logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

logger.Info("request processed",
    "method", "POST",
    "path", "/api/generate",
    "duration_ms", 150,
    "status", 200,
)
// Output: {"time":"...","level":"INFO","msg":"request processed","method":"POST",...}
```

### Key Principles
- **Consistent Fields:** Same field names across all logs
- **Structured, Not Strings:** Use key-value pairs, not formatted strings
- **Appropriate Levels:** DEBUG for development, INFO for operations, ERROR for failures
- **Correlation IDs:** Track requests across services

### Compliance (FAR/DFARS, HIPAA)
- Audit logs must be tamper-evident (append-only)
- Must include: timestamp, action, actor, outcome
- Retention policies vary by regulation

---

## 6. Context Propagation

**Files:** Throughout codebase

### Concept
`context.Context` carries:
- Cancellation signals (user cancelled, timeout)
- Deadlines
- Request-scoped values (use sparingly)

### What You Learned
```go
// Always pass context as first parameter
func Generate(ctx context.Context, req Request) (*Response, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Use context-aware HTTP client
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, body)
}
```

### Key Principles
- **First Parameter:** Convention is `ctx context.Context` as first arg
- **Don't Store in Structs:** Pass explicitly through call chain
- **Check Cancellation:** Especially before expensive operations
- **Create Child Contexts:** Add timeouts for sub-operations

### Why This Matters
- Prevents goroutine leaks (cancelled contexts clean up)
- Enables request-level timeouts
- Critical for distributed tracing

---

## 7. Configuration Management

**Files:** `cmd/glr/main.go`

### Concept
Configuration comes from multiple sources with different priorities:
1. Flags (highest priority)
2. Environment variables
3. Config files
4. Defaults (lowest priority)

### What You Learned
```go
// Flags with env var defaults
addr := flag.String("addr", envOrDefault("GLR_ADDR", ":8080"), "Listen address")

func envOrDefault(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}
```

### Key Principles
- **Explicit Over Implicit:** Make defaults visible
- **Validate Early:** Fail at startup, not runtime
- **Document All Options:** Usage text should be complete
- **Secrets Via Env:** Never put secrets in flags (visible in ps)

---

## Summary: Phase 1 Competencies

After completing Phase 1, you can:

| Skill | Assessment |
|-------|------------|
| Design Go interfaces | Define at point of use, small & focused |
| Write table-driven tests | Exhaustive coverage, easy extension |
| Handle errors properly | Wrap with context, check with errors.Is |
| Build HTTP servers | Graceful shutdown, proper timeouts |
| Implement structured logging | JSON output, consistent fields |
| Use context correctly | Propagate cancellation, add timeouts |
| Manage configuration | Flags + env vars + defaults |

---

## Exercises Checklist

- [x] **Exercise 1.1:** Retry with exponential backoff ✓
  - Implements: time.After, select, context cancellation
  - Files: `internal/ollama/retry.go`, `internal/ollama/retry_test.go`
  - Key Learnings:
    - Exponential backoff with configurable multiplier (2x)
    - Jitter (±10%) to prevent thundering herd problem
    - Context cancellation during delays using select
    - Error classification (retryable vs non-retryable)
  
- [ ] **Exercise 1.2:** Logging middleware
  - Implements: middleware pattern, correlation IDs, ResponseWriter wrapping
  - File: `internal/middleware/logging.go`

---

## Next Phase Preview

**Phase 2: Concurrency** will cover:
- Goroutines and channels
- Worker pools with bounded parallelism
- Race condition detection
- sync primitives (Mutex, WaitGroup, Once)

The foundation you've built in Phase 1 (interfaces, testing, error handling) will be essential for safely implementing concurrent code.

