# GoLlama Runtime (GLR)

A production-grade LLM tool orchestration runtime built in Go. This project teaches systems programming from undergraduate to postgraduate level through hands-on implementation.

## Purpose

GLR is both a learning project and a real runtime that:
- Proxies requests to local Ollama instances
- Provides structured audit logging for compliance
- Implements graceful shutdown and proper error handling
- Serves as the foundation for tool-use orchestration (Phases 2-7)

## Quick Start

```bash
# Prerequisites: Go 1.22+, Ollama running locally

# Clone and build
git clone https://github.com/khaaliswooden-max/go_project.git
cd go_project
make build

# Run (Ollama must be running: ollama serve)
make run

# Test
curl http://localhost:8080/health

# Generate text
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model": "llama3.2", "prompt": "Hello!"}'
```

## Project Structure

```
.
├── cmd/glr/              # Application entry point
├── internal/
│   ├── ollama/           # Ollama API client
│   ├── server/           # HTTP server and handlers
│   ├── audit/            # Audit logging
│   └── middleware/       # HTTP middleware (Exercise 1.2)
├── pkg/
│   ├── types/            # Shared domain types
│   └── errors/           # Error definitions
├── docs/                 # Documentation
├── scripts/              # Build utilities
└── .github/workflows/    # CI pipeline
```

## Learning Path

### Phase 1: Foundations (Current)
- Interface design and dependency injection
- HTTP server patterns with graceful shutdown
- Structured logging with correlation IDs
- Table-driven testing methodology
- Error handling and wrapping

**Exercises:**
- 1.1: Implement retry with exponential backoff
- 1.2: Implement logging middleware with correlation IDs

Run `make learn` for detailed objectives.

### Phase 2: Concurrency
- Worker pools and bounded parallelism
- Channel patterns and select statements
- Race condition detection

### Phase 3: Generics
- Type parameters and constraints
- Code generation

### Phase 4: Performance
- Profiling with pprof
- Benchmarking
- Memory pooling

### Phase 5: Systems Programming
- unsafe package
- Memory-mapped files
- CGO integration

### Phase 6: Static Analysis
- go/ast and go/types
- Custom linters

### Phase 7: Distributed Systems
- Consensus algorithms
- gRPC streaming

## Development Commands

```bash
make build          # Build binary
make run            # Start server
make test           # Run tests with race detector
make lint           # Run linters
make bench          # Run benchmarks
make coverage       # View coverage in browser

# Learning
make learn          # Show current phase objectives
make exercise-1-1   # Start Exercise 1.1
make verify-1-1     # Verify Exercise 1.1 solution
```

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-addr` | `GLR_ADDR` | `:8080` | Server listen address |
| `-ollama-url` | `OLLAMA_URL` | `http://localhost:11434` | Ollama API URL |
| `-timeout` | - | `30s` | Request timeout |
| `-log-level` | `LOG_LEVEL` | `info` | Log level |
| `-audit-file` | `AUDIT_FILE` | stdout | Audit log file |

## API Endpoints

### Health Check
```http
GET /health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "dev",
  "ollama": {
    "connected": true,
    "models": ["llama3.2", "mistral"]
  }
}
```

### Generate
```http
POST /api/generate
Content-Type: application/json

{
  "model": "llama3.2",
  "prompt": "Explain Go interfaces",
  "stream": false
}
```

### Chat
```http
POST /api/chat
Content-Type: application/json

{
  "model": "llama3.2",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
```

## Integration with Zuup Ecosystem

GLR is designed to become a core component of the Zuup platform:

- **Veyra (Autonomy OS):** GLR provides the tool execution layer with safety boundaries
- **Aureon (Procurement):** Audit logs satisfy FAR/DFARS traceability requirements
- **Civium (Halal Compliance):** Tool attestation hooks for compliance verification
- **zuup-core:** `pkg/types` becomes shared primitives across platforms

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `make lint test`
5. Submit a pull request

## License

MIT
