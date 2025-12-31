// pkg/types/types.go
// Shared domain types for GLR
//
// LEARN: Go's type system is structural, not nominal. Two types with
// identical structure are assignment-compatible. We use named types
// here for documentation and to enable method attachment.

package types

import "time"

// === Ollama API Types ===
// Reference: https://github.com/ollama/ollama/blob/main/docs/api.md

// GenerateRequest represents a request to the /api/generate endpoint.
//
// LEARN: Struct tags control JSON serialization. `omitempty` excludes
// zero-valued fields from output. This is crucial for API compatibility
// where absence and zero have different meanings.
type GenerateRequest struct {
	Model  string `json:"model"`            // Required: model name (e.g., "llama3.2")
	Prompt string `json:"prompt"`           // Required: the prompt to generate from
	System string `json:"system,omitempty"` // Optional: system prompt
	Stream bool   `json:"stream"`           // If true, response streams as chunks

	// Advanced options - Phase 2+
	Options *ModelOptions `json:"options,omitempty"`
}

// GenerateResponse represents a non-streaming response from /api/generate.
type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason,omitempty"`
	Context            []int     `json:"context,omitempty"` // Token IDs for conversation continuation
	TotalDuration      int64     `json:"total_duration"`    // Nanoseconds
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

// GenerateChunk represents a single chunk in a streaming response.
//
// LEARN: Streaming responses reuse the same structure but with partial
// data. The `Done` field signals the final chunk.
type GenerateChunk struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"` // Partial token(s)
	Done      bool      `json:"done"`

	// Final chunk only
	DoneReason         string `json:"done_reason,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// ChatRequest represents a request to the /api/chat endpoint.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []Message     `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  *ModelOptions `json:"options,omitempty"`
}

// Message represents a single message in a chat conversation.
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatResponse represents a non-streaming response from /api/chat.
type ChatResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            Message   `json:"message"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason,omitempty"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

// ChatChunk represents a single chunk in a streaming chat response.
type ChatChunk struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            Message   `json:"message"`
	Done               bool      `json:"done"`
	DoneReason         string    `json:"done_reason,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// ModelOptions contains model-specific parameters.
//
// LEARN: Using pointers for optional numeric fields allows distinguishing
// between "not set" (nil) and "set to zero" (0). This is important for
// parameters like Temperature where 0 is a valid, meaningful value.
type ModelOptions struct {
	Temperature *float64 `json:"temperature,omitempty"` // 0.0-2.0, default varies by model
	TopP        *float64 `json:"top_p,omitempty"`       // 0.0-1.0
	TopK        *int     `json:"top_k,omitempty"`       // Number of tokens to consider
	NumPredict  *int     `json:"num_predict,omitempty"` // Max tokens to generate
	Stop        []string `json:"stop,omitempty"`        // Stop sequences
	Seed        *int     `json:"seed,omitempty"`        // For reproducibility
	NumCtx      *int     `json:"num_ctx,omitempty"`     // Context window size
}

// === Server Types ===

// HealthResponse is the response from the health check endpoint.
type HealthResponse struct {
	Status    string        `json:"status"` // "ok", "degraded", "unhealthy"
	Timestamp time.Time     `json:"timestamp"`
	Version   string        `json:"version"`
	Ollama    *OllamaHealth `json:"ollama,omitempty"`
}

// OllamaHealth represents the health status of the Ollama backend.
type OllamaHealth struct {
	Connected bool     `json:"connected"`
	Models    []string `json:"models,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ErrorResponse is a standardized error response.
//
// LEARN: Consistent error responses make API debugging easier.
// Include enough context to diagnose without exposing internals.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`       // Machine-readable error code
	RequestID string `json:"request_id,omitempty"` // Correlation ID for tracing
	Details   any    `json:"details,omitempty"`    // Additional context (Phase 2+)
}

// === Audit Types ===

// AuditEntry represents a single audit log entry.
//
// LEARN: Audit logs are critical for compliance (FAR/DFARS, HIPAA).
// Design them to be append-only and tamper-evident.
type AuditEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	RequestID    string    `json:"request_id"`
	Action       string    `json:"action"` // "generate", "chat", "tool_call"
	Model        string    `json:"model,omitempty"`
	DurationMs   int64     `json:"duration_ms"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	Success      bool      `json:"success"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ClientIP     string    `json:"client_ip,omitempty"` // Hashed for privacy
	UserAgent    string    `json:"user_agent,omitempty"`
}
