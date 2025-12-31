// internal/server/http.go
// HTTP server with graceful shutdown
//
// LEARN: Production HTTP servers must:
// 1. Handle graceful shutdown (drain in-flight requests)
// 2. Set appropriate timeouts
// 3. Provide health checks
// 4. Use middleware for cross-cutting concerns

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/khaaliswooden-max/go_project/internal/audit"
	"github.com/khaaliswooden-max/go_project/internal/ollama"
	"github.com/khaaliswooden-max/go_project/pkg/errors"
	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// Version is set at build time via ldflags.
var Version = "dev"

// Config holds server configuration.
type Config struct {
	Addr           string        // Listen address (default: ":8080")
	ReadTimeout    time.Duration // Max time to read request (default: 30s)
	WriteTimeout   time.Duration // Max time to write response (default: 60s)
	IdleTimeout    time.Duration // Max time for keep-alive (default: 120s)
	ShutdownTimeout time.Duration // Max time to wait for graceful shutdown (default: 30s)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Addr:           ":8080",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Server is the GLR HTTP server.
type Server struct {
	config       Config
	httpServer   *http.Server
	ollamaClient ollama.Client
	auditLogger  *audit.Logger
	logger       *slog.Logger
}

// New creates a new Server.
//
// LEARN: Dependency injection via constructor parameters.
// All dependencies are explicit, making testing easy.
func New(cfg Config, ollamaClient ollama.Client, auditLogger *audit.Logger, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Server{
		config:       cfg,
		ollamaClient: ollamaClient,
		auditLogger:  auditLogger,
		logger:       logger,
	}

	// Create router
	mux := http.NewServeMux()

	// Register routes
	// LEARN: Go 1.22+ supports method routing: "POST /path"
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/generate", s.handleGenerate)
	mux.HandleFunc("POST /api/chat", s.handleChat)

	// Wrap with middleware
	var handler http.Handler = mux
	handler = s.recoveryMiddleware(handler)
	handler = s.loggingMiddleware(handler)

	s.httpServer = &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// Run starts the server and blocks until shutdown.
//
// LEARN: This pattern handles graceful shutdown:
// 1. Start server in goroutine
// 2. Wait for shutdown signal
// 3. Call Shutdown() with timeout context
// 4. Wait for server goroutine to finish
func (s *Server) Run() error {
	// Channel for server errors
	errCh := make(chan error, 1)

	// Start server
	go func() {
		s.logger.Info("starting server", "addr", s.config.Addr, "version", Version)
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.Info("shutdown signal received", "signal", sig.String())
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	s.logger.Info("shutting down server", "timeout", s.config.ShutdownTimeout)
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	s.logger.Info("server stopped")
	return nil
}

// Shutdown stops the server gracefully.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// === Middleware ===

// loggingMiddleware logs request/response information.
//
// LEARN: Middleware wraps handlers to add cross-cutting concerns.
// The pattern: return a function that takes a handler and returns a handler.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log after request completes
		s.logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// recoveryMiddleware recovers from panics and returns 500.
//
// LEARN: Always recover from panics in HTTP handlers.
// An unrecovered panic crashes the entire server.
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic recovered",
					"error", err,
					"path", r.URL.Path,
				)
				s.writeError(w, fmt.Errorf("%w: %v", errors.ErrInternalError, err), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// === Handlers ===

// handleHealth returns server and Ollama health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check Ollama health
	var ollamaHealth *types.OllamaHealth
	if s.ollamaClient != nil {
		ollamaHealth, _ = s.ollamaClient.Health(ctx)
	}

	status := "ok"
	if ollamaHealth != nil && !ollamaHealth.Connected {
		status = "degraded"
	}

	resp := types.HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Version:   Version,
		Ollama:    ollamaHealth,
	}

	s.writeJSON(w, resp, http.StatusOK)
}

// handleGenerate proxies generate requests to Ollama.
func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Parse request
	var req types.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, fmt.Errorf("%w: %v", errors.ErrInvalidJSON, err), http.StatusBadRequest)
		return
	}

	// Check if streaming requested
	if req.Stream {
		s.handleGenerateStream(w, r, req)
		return
	}

	// Non-streaming request
	resp, err := s.ollamaClient.Generate(ctx, req)
	if err != nil {
		statusCode := s.errorStatusCode(err)
		s.writeError(w, err, statusCode)
		
		// Audit log for failed request
		if s.auditLogger != nil {
			s.auditLogger.Log(types.AuditEntry{
				Timestamp: time.Now().UTC(),
				Action:    "generate",
				Model:     req.Model,
				DurationMs: time.Since(start).Milliseconds(),
				Success:   false,
				ErrorCode: errors.ErrorCode(err),
			})
		}
		return
	}

	// Audit log for successful request
	if s.auditLogger != nil {
		s.auditLogger.Log(types.AuditEntry{
			Timestamp:    time.Now().UTC(),
			Action:       "generate",
			Model:        req.Model,
			DurationMs:   time.Since(start).Milliseconds(),
			InputTokens:  resp.PromptEvalCount,
			OutputTokens: resp.EvalCount,
			Success:      true,
		})
	}

	s.writeJSON(w, resp, http.StatusOK)
}

// handleGenerateStream handles streaming generate requests.
func (s *Server) handleGenerateStream(w http.ResponseWriter, r *http.Request, req types.GenerateRequest) {
	ctx := r.Context()

	// Check if response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, fmt.Errorf("%w: streaming not supported", errors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Stream from Ollama
	chunks, errs := s.ollamaClient.GenerateStream(ctx, req)

	for chunk := range chunks {
		if err := json.NewEncoder(w).Encode(chunk); err != nil {
			s.logger.Error("stream write error", "error", err)
			return
		}
		flusher.Flush()
	}

	// Check for errors
	select {
	case err := <-errs:
		if err != nil {
			s.logger.Error("stream error", "error", err)
		}
	default:
	}
}

// handleChat proxies chat requests to Ollama.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	var req types.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, fmt.Errorf("%w: %v", errors.ErrInvalidJSON, err), http.StatusBadRequest)
		return
	}

	if req.Stream {
		s.handleChatStream(w, r, req)
		return
	}

	resp, err := s.ollamaClient.Chat(ctx, req)
	if err != nil {
		statusCode := s.errorStatusCode(err)
		s.writeError(w, err, statusCode)
		
		if s.auditLogger != nil {
			s.auditLogger.Log(types.AuditEntry{
				Timestamp:  time.Now().UTC(),
				Action:     "chat",
				Model:      req.Model,
				DurationMs: time.Since(start).Milliseconds(),
				Success:    false,
				ErrorCode:  errors.ErrorCode(err),
			})
		}
		return
	}

	if s.auditLogger != nil {
		s.auditLogger.Log(types.AuditEntry{
			Timestamp:    time.Now().UTC(),
			Action:       "chat",
			Model:        req.Model,
			DurationMs:   time.Since(start).Milliseconds(),
			InputTokens:  resp.PromptEvalCount,
			OutputTokens: resp.EvalCount,
			Success:      true,
		})
	}

	s.writeJSON(w, resp, http.StatusOK)
}

// handleChatStream handles streaming chat requests.
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request, req types.ChatRequest) {
	ctx := r.Context()

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, fmt.Errorf("%w: streaming not supported", errors.ErrInternalError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	chunks, errs := s.ollamaClient.ChatStream(ctx, req)

	for chunk := range chunks {
		if err := json.NewEncoder(w).Encode(chunk); err != nil {
			s.logger.Error("stream write error", "error", err)
			return
		}
		flusher.Flush()
	}

	select {
	case err := <-errs:
		if err != nil {
			s.logger.Error("stream error", "error", err)
		}
	default:
	}
}

// === Response Helpers ===

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("json encode error", "error", err)
	}
}

// writeError writes an error response.
func (s *Server) writeError(w http.ResponseWriter, err error, statusCode int) {
	resp := types.ErrorResponse{
		Error: err.Error(),
		Code:  errors.ErrorCode(err),
	}
	s.writeJSON(w, resp, statusCode)
}

// errorStatusCode maps errors to HTTP status codes.
//
// LEARN: Consistent error mapping makes APIs predictable.
// Clients can rely on status codes for retry logic.
func (s *Server) errorStatusCode(err error) int {
	switch {
	case errors.Is(err, errors.ErrInvalidRequest), errors.Is(err, errors.ErrInvalidJSON):
		return http.StatusBadRequest
	case errors.Is(err, errors.ErrModelNotFound):
		return http.StatusNotFound
	case errors.Is(err, errors.ErrRequestCanceled):
		return 499 // Client Closed Request (nginx convention)
	case errors.Is(err, errors.ErrRequestTimeout), errors.Is(err, errors.ErrOllamaTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, errors.ErrOllamaUnavailable):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
