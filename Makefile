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
	@echo "=== Phase 1: Foundations ==="
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
	@echo "  - internal/server/http.go        (HTTP patterns)"
	@echo "  - internal/audit/logger.go       (Structured logging)"
	@echo ""
	@echo "Exercises:"
	@echo "  1.1: Add retry with exponential backoff"
	@echo "  1.2: Add request/response logging middleware"
	@echo ""
	@echo "Run 'make exercise-1-1' to start Exercise 1.1"

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
	@echo "  make bench          Run benchmarks"
	@echo "  make coverage       Show coverage in browser"
	@echo ""
	@echo "Quality:"
	@echo "  make lint           Run linters"
	@echo "  make check-ollama   Verify Ollama is running"
	@echo ""
	@echo "Learning:"
	@echo "  make learn          Show current phase objectives"
	@echo "  make exercise-1-1   Start Exercise 1.1"
	@echo "  make exercise-1-2   Start Exercise 1.2"
	@echo "  make verify-1-1     Verify Exercise 1.1 solution"
	@echo "  make verify-1-2     Verify Exercise 1.2 solution"
