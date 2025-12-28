// internal/ollama/client.go
// Ollama API client with interface-based design
//
// LEARN: Interface Segregation Principle (ISP)
// Define interfaces at the point of use, not at the point of implementation.
// Small, focused interfaces are easier to mock and compose.

package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/khaaliswooden-max/go_project/pkg/errors"
	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// === Interfaces ===
// LEARN: Define interfaces for what you need, not what you have.
// This enables dependency injection and testability.

// Generator handles text generation requests.
// Separated from Chat to allow independent mocking and evolution.
type Generator interface {
	// Generate performs a non-streaming text generation.
	// Returns the complete response or an error.
	Generate(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error)

	// GenerateStream performs a streaming text generation.
	// Returns a channel that emits chunks until done or error.
	// The channel is closed when generation completes.
	// Errors are returned via the last chunk or the error channel.
	GenerateStream(ctx context.Context, req types.GenerateRequest) (<-chan types.GenerateChunk, <-chan error)
}

// Chatter handles chat-based interactions.
type Chatter interface {
	// Chat performs a non-streaming chat completion.
	Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error)

	// ChatStream performs a streaming chat completion.
	ChatStream(ctx context.Context, req types.ChatRequest) (<-chan types.ChatChunk, <-chan error)
}

// HealthChecker verifies Ollama service health.
type HealthChecker interface {
	// Health checks if Ollama is reachable and returns available models.
	Health(ctx context.Context) (*types.OllamaHealth, error)
}

// Client combines all Ollama capabilities.
// LEARN: Embedding interfaces creates a composite interface.
// Callers can depend on the specific interface they need.
type Client interface {
	Generator
	Chatter
	HealthChecker
}

// === Configuration ===

// Config holds client configuration.
//
// LEARN: Use structs for configuration rather than many parameters.
// This enables named fields, defaults, and validation.
type Config struct {
	BaseURL    string        // Ollama API URL (default: http://localhost:11434)
	Timeout    time.Duration // Request timeout (default: 30s)
	HTTPClient *http.Client  // Optional custom HTTP client
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL: "http://localhost:11434",
		Timeout: 30 * time.Second,
	}
}

// Validate checks configuration validity.
//
// LEARN: Fail fast on invalid configuration. It's better to error
// at startup than to fail mysteriously at runtime.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("%w: BaseURL is required", errors.ErrInvalidRequest)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("%w: Timeout must be positive", errors.ErrInvalidRequest)
	}
	return nil
}

// === Implementation ===

// client implements the Client interface.
// LEARN: Unexported struct with exported interface - this is idiomatic Go.
// Callers depend on the interface, not the concrete type.
type client struct {
	config     Config
	httpClient *http.Client
}

// New creates a new Ollama client with the given configuration.
//
// LEARN: Constructor functions return interfaces, not concrete types.
// This enforces interface-based design.
func New(cfg Config) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.Timeout,
		}
	}

	return &client{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

// NewDefault creates a client with default configuration.
func NewDefault() (Client, error) {
	return New(DefaultConfig())
}

// === Generate Implementation ===

// Generate performs a non-streaming text generation.
func (c *client) Generate(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
	// LEARN: Always validate inputs at API boundaries.
	if req.Model == "" {
		return nil, fmt.Errorf("%w: model is required", errors.ErrInvalidRequest)
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("%w: prompt is required", errors.ErrInvalidRequest)
	}

	// Force non-streaming for this method
	req.Stream = false

	// Serialize request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal request: %v", errors.ErrInvalidJSON, err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, errors.WrapOllamaError("create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	// LEARN: Always use the context-aware methods. This ensures
	// cancellation and timeouts propagate correctly.
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Check for context cancellation
		if ctx.Err() == context.Canceled {
			return nil, errors.ErrRequestCanceled
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errors.ErrRequestTimeout
		}
		return nil, fmt.Errorf("%w: %v", errors.ErrOllamaUnavailable, err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	// Parse response
	var result types.GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", errors.ErrInvalidJSON, err)
	}

	return &result, nil
}

// GenerateStream performs a streaming text generation.
//
// LEARN: Returning channels is idiomatic Go for streaming APIs.
// The caller ranges over the channel; the producer closes it when done.
func (c *client) GenerateStream(ctx context.Context, req types.GenerateRequest) (<-chan types.GenerateChunk, <-chan error) {
	chunks := make(chan types.GenerateChunk)
	errs := make(chan error, 1) // Buffered to avoid goroutine leak

	go func() {
		defer close(chunks)
		defer close(errs)

		// Validate
		if req.Model == "" {
			errs <- fmt.Errorf("%w: model is required", errors.ErrInvalidRequest)
			return
		}
		if req.Prompt == "" {
			errs <- fmt.Errorf("%w: prompt is required", errors.ErrInvalidRequest)
			return
		}

		// Force streaming
		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			errs <- fmt.Errorf("%w: failed to marshal request: %v", errors.ErrInvalidJSON, err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/generate", bytes.NewReader(body))
		if err != nil {
			errs <- errors.WrapOllamaError("create request", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			if ctx.Err() == context.Canceled {
				errs <- errors.ErrRequestCanceled
				return
			}
			if ctx.Err() == context.DeadlineExceeded {
				errs <- errors.ErrRequestTimeout
				return
			}
			errs <- fmt.Errorf("%w: %v", errors.ErrOllamaUnavailable, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errs <- c.parseErrorResponse(resp)
			return
		}

		// LEARN: bufio.Scanner is efficient for line-delimited streaming.
		// Each line is a JSON object (NDJSON format).
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			// Check context cancellation between chunks
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			var chunk types.GenerateChunk
			if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
				errs <- fmt.Errorf("%w: failed to decode chunk: %v", errors.ErrStreamingError, err)
				return
			}

			chunks <- chunk

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("%w: scanner error: %v", errors.ErrStreamingError, err)
		}
	}()

	return chunks, errs
}

// === Chat Implementation ===

// Chat performs a non-streaming chat completion.
func (c *client) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("%w: model is required", errors.ErrInvalidRequest)
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("%w: messages is required", errors.ErrInvalidRequest)
	}

	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal request: %v", errors.ErrInvalidJSON, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, errors.WrapOllamaError("create request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, errors.ErrRequestCanceled
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errors.ErrRequestTimeout
		}
		return nil, fmt.Errorf("%w: %v", errors.ErrOllamaUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var result types.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", errors.ErrInvalidJSON, err)
	}

	return &result, nil
}

// ChatStream performs a streaming chat completion.
func (c *client) ChatStream(ctx context.Context, req types.ChatRequest) (<-chan types.ChatChunk, <-chan error) {
	chunks := make(chan types.ChatChunk)
	errs := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errs)

		if req.Model == "" {
			errs <- fmt.Errorf("%w: model is required", errors.ErrInvalidRequest)
			return
		}
		if len(req.Messages) == 0 {
			errs <- fmt.Errorf("%w: messages is required", errors.ErrInvalidRequest)
			return
		}

		req.Stream = true

		body, err := json.Marshal(req)
		if err != nil {
			errs <- fmt.Errorf("%w: failed to marshal request: %v", errors.ErrInvalidJSON, err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errs <- errors.WrapOllamaError("create request", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			if ctx.Err() == context.Canceled {
				errs <- errors.ErrRequestCanceled
				return
			}
			if ctx.Err() == context.DeadlineExceeded {
				errs <- errors.ErrRequestTimeout
				return
			}
			errs <- fmt.Errorf("%w: %v", errors.ErrOllamaUnavailable, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errs <- c.parseErrorResponse(resp)
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			var chunk types.ChatChunk
			if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
				errs <- fmt.Errorf("%w: failed to decode chunk: %v", errors.ErrStreamingError, err)
				return
			}

			chunks <- chunk

			if chunk.Done {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("%w: scanner error: %v", errors.ErrStreamingError, err)
		}
	}()

	return chunks, errs
}

// === Health Check Implementation ===

// Health checks if Ollama is reachable and returns available models.
func (c *client) Health(ctx context.Context) (*types.OllamaHealth, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return &types.OllamaHealth{Connected: false, Error: err.Error()}, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &types.OllamaHealth{Connected: false, Error: err.Error()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &types.OllamaHealth{Connected: false, Error: fmt.Sprintf("unexpected status: %d", resp.StatusCode)}, nil
	}

	// Parse models list
	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return &types.OllamaHealth{Connected: true, Error: "failed to parse models"}, nil
	}

	models := make([]string, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = m.Name
	}

	return &types.OllamaHealth{Connected: true, Models: models}, nil
}

// === Helper Methods ===

// parseErrorResponse extracts error information from an HTTP response.
func (c *client) parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse as JSON error
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		// Check for specific error types
		switch {
		case resp.StatusCode == http.StatusNotFound:
			return fmt.Errorf("%w: %s", errors.ErrModelNotFound, errResp.Error)
		default:
			return fmt.Errorf("%w: %s", errors.ErrOllamaUnavailable, errResp.Error)
		}
	}

	// Generic error
	return fmt.Errorf("%w: HTTP %d: %s", errors.ErrOllamaUnavailable, resp.StatusCode, string(body))
}
