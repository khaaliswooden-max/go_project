// internal/ollama/parallel.go
// Parallel request handler with bounded concurrency
//
// LEARN: This module demonstrates the fan-out/fan-in concurrency pattern:
// - Fan-out: Distribute multiple requests across goroutines
// - Fan-in: Collect results into a single channel/slice
// - Bounded: Use semaphore to limit concurrent requests (rate limiting)
//
// This is Exercise 2.2 - Parallel Request Handler

package ollama

import (
	"context"
	"fmt"
	"sync"

	"github.com/khaaliswooden-max/go_project/pkg/errors"
	glrsync "github.com/khaaliswooden-max/go_project/pkg/sync"
	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// === Interfaces ===

// ParallelGenerator handles multiple generation requests in parallel.
//
// LEARN: Separate interface for parallel operations allows independent
// mocking and evolution from single-request Generator.
type ParallelGenerator interface {
	// GenerateParallel processes multiple prompts concurrently.
	// Results are returned in the same order as the input prompts.
	// Partial failures are captured in individual results, not as a single error.
	GenerateParallel(ctx context.Context, req ParallelGenerateRequest) (*ParallelGenerateResponse, error)
}

// === Types ===

// ParallelGenerateRequest represents a batch of generation requests.
type ParallelGenerateRequest struct {
	Model       string   // Model to use for all prompts
	Prompts     []string // Prompts to process in parallel
	System      string   // Optional system prompt for all requests
	MaxParallel int      // Maximum concurrent requests (0 = default to 4)
}

// ParallelGenerateResponse contains results from parallel generation.
type ParallelGenerateResponse struct {
	Results      []GenerateResult // Results in same order as input prompts
	TotalTimeMs  int64            // Total wall-clock time in milliseconds
	SuccessCount int              // Number of successful generations
	ErrorCount   int              // Number of failed generations
}

// GenerateResult represents the result of a single generation request.
type GenerateResult struct {
	Index    int                     // Original index in the prompts slice
	Prompt   string                  // The original prompt
	Response *types.GenerateResponse // Response on success (nil on error)
	Err      error                   // Error on failure (nil on success)
}

// === Configuration ===

// ParallelConfig holds configuration for parallel operations.
type ParallelConfig struct {
	MaxParallel int // Max concurrent requests (default: 4)
}

// DefaultParallelConfig returns sensible defaults.
func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		MaxParallel: 4,
	}
}

// === Implementation ===

// parallelClient wraps a Generator to add parallel capabilities.
//
// LEARN: Composition over inheritance - we wrap Generator rather than
// modifying it. This is the Decorator pattern.
type parallelClient struct {
	generator Generator
	config    ParallelConfig
}

// NewParallel creates a parallel generator that wraps an existing Generator.
//
// LEARN: Factory function returns interface, not concrete type.
// This enables dependency injection for testing.
func NewParallel(gen Generator, cfg ParallelConfig) ParallelGenerator {
	if cfg.MaxParallel <= 0 {
		cfg.MaxParallel = DefaultParallelConfig().MaxParallel
	}
	return &parallelClient{
		generator: gen,
		config:    cfg,
	}
}

// NewParallelDefault creates a parallel generator with default configuration.
func NewParallelDefault(gen Generator) ParallelGenerator {
	return NewParallel(gen, DefaultParallelConfig())
}

// GenerateParallel processes multiple prompts concurrently with bounded parallelism.
//
// LEARN: This implements the fan-out/fan-in pattern:
//
//	Fan-out: Each prompt spawns a goroutine (limited by semaphore)
//	Fan-in: Results collected via channel, then sorted by index
//
// Key design decisions:
//  1. Bounded concurrency via semaphore (prevents overwhelming the LLM server)
//  2. Results maintain input order (predictable API behavior)
//  3. Partial failures don't abort other requests (resilient)
//  4. Context cancellation stops new requests but lets in-flight complete
func (p *parallelClient) GenerateParallel(ctx context.Context, req ParallelGenerateRequest) (*ParallelGenerateResponse, error) {
	// Validate request
	if req.Model == "" {
		return nil, fmt.Errorf("%w: model is required", errors.ErrInvalidRequest)
	}
	if len(req.Prompts) == 0 {
		return nil, fmt.Errorf("%w: prompts is required", errors.ErrInvalidRequest)
	}

	// Determine parallelism level
	maxParallel := req.MaxParallel
	if maxParallel <= 0 {
		maxParallel = p.config.MaxParallel
	}
	// Don't use more workers than prompts
	if maxParallel > len(req.Prompts) {
		maxParallel = len(req.Prompts)
	}

	// Create semaphore for rate limiting
	// LEARN: Semaphore ensures we don't overwhelm the LLM server
	sem := glrsync.New(maxParallel)
	defer sem.Close()

	// Channel to collect results
	// LEARN: Buffered channel prevents goroutines from blocking on send
	resultsCh := make(chan GenerateResult, len(req.Prompts))

	// WaitGroup to track completion
	var wg sync.WaitGroup

	// Fan-out: Launch a goroutine for each prompt
	// LEARN: Each goroutine acquires semaphore before processing
	for i, prompt := range req.Prompts {
		wg.Add(1)
		go func(index int, promptText string) {
			defer wg.Done()

			result := GenerateResult{
				Index:  index,
				Prompt: promptText,
			}

			// Acquire semaphore (rate limit)
			// LEARN: This blocks until a slot is available
			if err := sem.Acquire(ctx); err != nil {
				// Context cancelled while waiting
				result.Err = err
				resultsCh <- result
				return
			}
			defer sem.Release()

			// Check context before making request
			if ctx.Err() != nil {
				result.Err = ctx.Err()
				resultsCh <- result
				return
			}

			// Execute the generation request
			genReq := types.GenerateRequest{
				Model:  req.Model,
				Prompt: promptText,
				System: req.System,
			}

			resp, err := p.generator.Generate(ctx, genReq)
			if err != nil {
				result.Err = err
			} else {
				result.Response = resp
			}

			resultsCh <- result
		}(i, prompt)
	}

	// Close results channel when all goroutines complete
	// LEARN: This must run in a separate goroutine to avoid deadlock
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Fan-in: Collect results
	// LEARN: We collect into a slice, then sort by index to preserve order
	results := make([]GenerateResult, len(req.Prompts))
	successCount := 0
	errorCount := 0

	for result := range resultsCh {
		results[result.Index] = result
		if result.Err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	return &ParallelGenerateResponse{
		Results:      results,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
	}, nil
}

// === Convenience Functions ===

// GenerateAll is a convenience function that processes prompts in parallel
// and returns only the successful responses.
//
// LEARN: Higher-level convenience functions make the API easier to use
// for common cases.
func GenerateAll(ctx context.Context, gen Generator, model string, prompts []string) ([]string, error) {
	parallel := NewParallelDefault(gen)

	resp, err := parallel.GenerateParallel(ctx, ParallelGenerateRequest{
		Model:   model,
		Prompts: prompts,
	})
	if err != nil {
		return nil, err
	}

	// Collect successful responses
	responses := make([]string, 0, resp.SuccessCount)
	for _, result := range resp.Results {
		if result.Err == nil && result.Response != nil {
			responses = append(responses, result.Response.Response)
		}
	}

	return responses, nil
}

// GenerateAllStrict is like GenerateAll but returns an error if any request fails.
func GenerateAllStrict(ctx context.Context, gen Generator, model string, prompts []string) ([]string, error) {
	parallel := NewParallelDefault(gen)

	resp, err := parallel.GenerateParallel(ctx, ParallelGenerateRequest{
		Model:   model,
		Prompts: prompts,
	})
	if err != nil {
		return nil, err
	}

	if resp.ErrorCount > 0 {
		// Find first error for the error message
		for _, result := range resp.Results {
			if result.Err != nil {
				return nil, fmt.Errorf("parallel generation failed: %d/%d requests failed, first error: %w",
					resp.ErrorCount, len(prompts), result.Err)
			}
		}
	}

	// Collect responses in order
	responses := make([]string, len(resp.Results))
	for i, result := range resp.Results {
		if result.Response != nil {
			responses[i] = result.Response.Response
		}
	}

	return responses, nil
}
