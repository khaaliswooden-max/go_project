// pkg/errors/errors.go
// Centralized error definitions for GLR
//
// LEARN: Sentinel errors are package-level variables that callers can
// compare against using errors.Is(). This pattern enables programmatic
// error handling while keeping error messages consistent.

package errors

import (
	stderrors "errors"
	"fmt"
)

// Re-export stdlib errors functions for convenience.
// This allows callers to use errors.Is() without importing both packages.
var (
	Is     = stderrors.Is
	As     = stderrors.As
	Unwrap = stderrors.Unwrap
)

// === Sentinel Errors ===
// Use errors.Is(err, ErrX) to check for these errors.

var (
	// Client errors (4xx equivalent)
	ErrInvalidRequest   = stderrors.New("invalid request")
	ErrModelNotFound    = stderrors.New("model not found")
	ErrModelNotLoaded   = stderrors.New("model not loaded")
	ErrRequestCanceled  = stderrors.New("request canceled")
	ErrRequestTimeout   = stderrors.New("request timeout")
	ErrInvalidJSON      = stderrors.New("invalid JSON")

	// Server errors (5xx equivalent)
	ErrOllamaUnavailable = stderrors.New("ollama service unavailable")
	ErrOllamaTimeout     = stderrors.New("ollama request timeout")
	ErrInternalError     = stderrors.New("internal server error")
	ErrStreamingError    = stderrors.New("streaming error")

	// Tool errors (Phase 2+)
	ErrToolNotFound      = stderrors.New("tool not found")
	ErrToolExecFailed    = stderrors.New("tool execution failed")
	ErrToolTimeout       = stderrors.New("tool execution timeout")
)

// === Wrapped Errors ===
// Use these to add context while preserving the underlying error.

// LEARN: Error wrapping with %w allows errors.Is() and errors.As()
// to unwrap and match the underlying error. This preserves the error
// chain for debugging while enabling programmatic handling.

// WrapOllamaError wraps an error from the Ollama client with context.
func WrapOllamaError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("ollama %s: %w", operation, err)
}

// WrapValidationError wraps a validation error with field context.
func WrapValidationError(field string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("validation error for %s: %w", field, err)
}

// === Error Codes ===
// Machine-readable error codes for API responses.

// LEARN: Error codes provide stable identifiers that clients can switch on,
// even as human-readable messages change. Use reverse domain notation
// or simple prefixed strings.

const (
	CodeInvalidRequest   = "GLR_INVALID_REQUEST"
	CodeModelNotFound    = "GLR_MODEL_NOT_FOUND"
	CodeOllamaUnavailable = "GLR_OLLAMA_UNAVAILABLE"
	CodeTimeout          = "GLR_TIMEOUT"
	CodeInternalError    = "GLR_INTERNAL_ERROR"
	CodeToolError        = "GLR_TOOL_ERROR"
)

// ErrorCode returns the appropriate error code for a given error.
//
// LEARN: Using errors.Is() allows matching wrapped errors.
// Order matters: check most specific errors first.
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidJSON):
		return CodeInvalidRequest
	case errors.Is(err, ErrModelNotFound), errors.Is(err, ErrModelNotLoaded):
		return CodeModelNotFound
	case errors.Is(err, ErrOllamaUnavailable):
		return CodeOllamaUnavailable
	case errors.Is(err, ErrRequestTimeout), errors.Is(err, ErrOllamaTimeout), errors.Is(err, ErrToolTimeout):
		return CodeTimeout
	case errors.Is(err, ErrToolNotFound), errors.Is(err, ErrToolExecFailed):
		return CodeToolError
	default:
		return CodeInternalError
	}
}

// IsRetryable returns true if the error is potentially transient and
// the operation should be retried.
//
// LEARN: Not all errors should trigger retries. Network issues and
// timeouts are retryable; validation errors are not. Getting this
// wrong wastes resources or causes silent failures.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrOllamaUnavailable) ||
		errors.Is(err, ErrOllamaTimeout) ||
		errors.Is(err, ErrStreamingError)
}

// IsClientError returns true if the error represents a client mistake
// (4xx equivalent) rather than a server problem.
func IsClientError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrInvalidRequest) ||
		errors.Is(err, ErrInvalidJSON) ||
		errors.Is(err, ErrModelNotFound) ||
		errors.Is(err, ErrRequestCanceled)
}
