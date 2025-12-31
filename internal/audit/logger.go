// internal/audit/logger.go
// Audit logging for compliance and observability
//
// LEARN: Audit logs serve multiple purposes:
// 1. Compliance (FAR/DFARS, HIPAA, SOC2)
// 2. Debugging and incident response
// 3. Usage analytics and billing
// 4. Security monitoring
//
// Key properties:
// - Append-only (tamper evidence)
// - Structured (machine parseable)
// - Complete (no missing entries)
// - Timestamped (with timezone)

package audit

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// Logger is a structured audit logger.
//
// LEARN: We use a mutex for thread safety. In Phase 4, we'll
// optimize with lock-free ring buffers and async batching.
type Logger struct {
	mu      sync.Mutex
	writer  io.Writer
	encoder *json.Encoder
}

// Config holds logger configuration.
type Config struct {
	Output io.Writer // Where to write logs (default: os.Stdout)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Output: os.Stdout,
	}
}

// New creates a new audit logger.
func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	return &Logger{
		writer:  cfg.Output,
		encoder: json.NewEncoder(cfg.Output),
	}
}

// NewDefault creates a logger with default configuration.
func NewDefault() *Logger {
	return New(DefaultConfig())
}

// Log writes an audit entry.
//
// LEARN: The mutex ensures entries are not interleaved when
// multiple goroutines write concurrently. This is a simple
// but potentially slow approach for high-throughput scenarios.
func (l *Logger) Log(entry types.AuditEntry) error {
	// Ensure timestamp is set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.encoder.Encode(entry)
}

// LogGenerate logs a generate request.
//
// LEARN: Helper methods provide type-safe logging for common actions.
// This reduces errors and ensures consistency.
func (l *Logger) LogGenerate(requestID, model string, durationMs int64, inputTokens, outputTokens int, success bool, errorCode string) error {
	return l.Log(types.AuditEntry{
		Timestamp:    time.Now().UTC(),
		RequestID:    requestID,
		Action:       "generate",
		Model:        model,
		DurationMs:   durationMs,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Success:      success,
		ErrorCode:    errorCode,
	})
}

// LogChat logs a chat request.
func (l *Logger) LogChat(requestID, model string, durationMs int64, inputTokens, outputTokens int, success bool, errorCode string) error {
	return l.Log(types.AuditEntry{
		Timestamp:    time.Now().UTC(),
		RequestID:    requestID,
		Action:       "chat",
		Model:        model,
		DurationMs:   durationMs,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Success:      success,
		ErrorCode:    errorCode,
	})
}

// LogToolCall logs a tool execution (Phase 2+).
func (l *Logger) LogToolCall(requestID, toolName string, durationMs int64, success bool, errorCode string) error {
	return l.Log(types.AuditEntry{
		Timestamp:  time.Now().UTC(),
		RequestID:  requestID,
		Action:     "tool_call:" + toolName,
		DurationMs: durationMs,
		Success:    success,
		ErrorCode:  errorCode,
	})
}

// === File Logger ===

// FileLogger extends Logger to write to a file.
//
// LEARN: For production, you typically want:
// 1. Log rotation (by size or time)
// 2. Compression of old logs
// 3. Retention policies
// These are usually handled by external tools (logrotate, fluentd)
// or dedicated logging libraries.
type FileLogger struct {
	*Logger
	file *os.File
}

// NewFileLogger creates a logger that writes to a file.
func NewFileLogger(path string) (*FileLogger, error) {
	// O_APPEND ensures atomic appends on most systems
	// O_CREATE creates if doesn't exist
	// O_WRONLY is write-only (we never read audit logs in the app)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileLogger{
		Logger: New(Config{Output: file}),
		file:   file,
	}, nil
}

// Close closes the underlying file.
func (l *FileLogger) Close() error {
	return l.file.Close()
}

// Sync flushes writes to disk.
//
// LEARN: os.File.Sync() calls fsync() which forces data to disk.
// This is important for durability but expensive. Use sparingly.
func (l *FileLogger) Sync() error {
	return l.file.Sync()
}

// === Multi Logger ===

// MultiLogger writes to multiple loggers.
//
// LEARN: This pattern (io.MultiWriter equivalent) allows
// simultaneous writing to stdout and file, for example.
type MultiLogger struct {
	loggers []*Logger
}

// NewMultiLogger creates a logger that writes to all provided loggers.
func NewMultiLogger(loggers ...*Logger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

// Log writes to all underlying loggers.
func (m *MultiLogger) Log(entry types.AuditEntry) error {
	var lastErr error
	for _, l := range m.loggers {
		if err := l.Log(entry); err != nil {
			lastErr = err
			// Continue logging to other loggers even if one fails
		}
	}
	return lastErr
}

// === In-Memory Logger (for testing) ===

// MemoryLogger stores entries in memory for testing.
type MemoryLogger struct {
	mu      sync.Mutex
	Entries []types.AuditEntry
}

// NewMemoryLogger creates an in-memory logger.
func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{
		Entries: make([]types.AuditEntry, 0),
	}
}

// Log appends to the in-memory slice.
func (m *MemoryLogger) Log(entry types.AuditEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Entries = append(m.Entries, entry)
	return nil
}

// Clear removes all entries.
func (m *MemoryLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Entries = m.Entries[:0]
}

// Count returns the number of entries.
func (m *MemoryLogger) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Entries)
}

// Last returns the most recent entry.
func (m *MemoryLogger) Last() *types.AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Entries) == 0 {
		return nil
	}
	return &m.Entries[len(m.Entries)-1]
}
