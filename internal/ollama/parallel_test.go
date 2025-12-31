// internal/ollama/parallel_test.go
// Tests for parallel request handler
//
// LEARN: Testing concurrent code with mocks requires:
// 1. Configurable delays to simulate real-world timing
// 2. Atomic counters to track concurrent execution
// 3. Error injection to test failure handling
// 4. Race detector: go test -race

package ollama

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	glrerrors "github.com/khaaliswooden-max/go_project/pkg/errors"
	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// === Mock Generator ===

// mockGenerator implements Generator for testing.
type mockGenerator struct {
	generateFunc  func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error)
	delay         time.Duration
	callCount     int32
	concurrent    int32
	maxConcurrent int32
}

func (m *mockGenerator) Generate(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
	atomic.AddInt32(&m.callCount, 1)

	// Track concurrency
	current := atomic.AddInt32(&m.concurrent, 1)
	defer atomic.AddInt32(&m.concurrent, -1)

	// Track max concurrent
	for {
		max := atomic.LoadInt32(&m.maxConcurrent)
		if current <= max || atomic.CompareAndSwapInt32(&m.maxConcurrent, max, current) {
			break
		}
	}

	// Simulate processing time
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Use custom function if provided
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}

	// Default response
	return &types.GenerateResponse{
		Model:    req.Model,
		Response: "Response to: " + req.Prompt,
		Done:     true,
	}, nil
}

func (m *mockGenerator) GenerateStream(ctx context.Context, req types.GenerateRequest) (<-chan types.GenerateChunk, <-chan error) {
	// Not used in parallel tests
	chunks := make(chan types.GenerateChunk)
	errs := make(chan error, 1)
	close(chunks)
	close(errs)
	return chunks, errs
}

func (m *mockGenerator) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockGenerator) ChatStream(ctx context.Context, req types.ChatRequest) (<-chan types.ChatChunk, <-chan error) {
	chunks := make(chan types.ChatChunk)
	errs := make(chan error, 1)
	close(chunks)
	close(errs)
	return chunks, errs
}

func (m *mockGenerator) Health(ctx context.Context) (*types.OllamaHealth, error) {
	return &types.OllamaHealth{Connected: true}, nil
}

// === Validation Tests ===

func TestParallelGenerator_ValidationErrors(t *testing.T) {
	mock := &mockGenerator{}
	parallel := NewParallelDefault(mock)

	tests := []struct {
		name    string
		req     ParallelGenerateRequest
		wantErr string
	}{
		{
			name:    "empty model",
			req:     ParallelGenerateRequest{Model: "", Prompts: []string{"test"}},
			wantErr: "model is required",
		},
		{
			name:    "empty prompts",
			req:     ParallelGenerateRequest{Model: "llama3.2", Prompts: []string{}},
			wantErr: "prompts is required",
		},
		{
			name:    "nil prompts",
			req:     ParallelGenerateRequest{Model: "llama3.2", Prompts: nil},
			wantErr: "prompts is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parallel.GenerateParallel(context.Background(), tc.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, glrerrors.ErrInvalidRequest) {
				t.Errorf("error = %v, want ErrInvalidRequest", err)
			}
		})
	}
}

// === Basic Functionality Tests ===

func TestParallelGenerator_SinglePrompt(t *testing.T) {
	mock := &mockGenerator{}
	parallel := NewParallelDefault(mock)

	resp, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: []string{"Hello"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Results length = %d, want 1", len(resp.Results))
	}

	if resp.Results[0].Err != nil {
		t.Errorf("Result.Err = %v, want nil", resp.Results[0].Err)
	}

	if resp.Results[0].Response == nil {
		t.Fatal("Result.Response is nil")
	}

	if resp.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", resp.SuccessCount)
	}
}

func TestParallelGenerator_MultiplePrompts(t *testing.T) {
	mock := &mockGenerator{}
	parallel := NewParallelDefault(mock)

	prompts := []string{"First", "Second", "Third", "Fourth", "Fifth"}

	resp, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: prompts,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Results) != len(prompts) {
		t.Fatalf("Results length = %d, want %d", len(resp.Results), len(prompts))
	}

	// Verify results are in correct order
	for i, result := range resp.Results {
		if result.Index != i {
			t.Errorf("Result[%d].Index = %d, want %d", i, result.Index, i)
		}
		if result.Prompt != prompts[i] {
			t.Errorf("Result[%d].Prompt = %q, want %q", i, result.Prompt, prompts[i])
		}
		if result.Err != nil {
			t.Errorf("Result[%d].Err = %v, want nil", i, result.Err)
		}
	}

	if resp.SuccessCount != len(prompts) {
		t.Errorf("SuccessCount = %d, want %d", resp.SuccessCount, len(prompts))
	}
}

func TestParallelGenerator_OrderPreserved(t *testing.T) {
	// LEARN: Results should be returned in the same order as input,
	// regardless of which completes first
	mock := &mockGenerator{
		delay: 10 * time.Millisecond,
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			// Add variable delay to encourage out-of-order completion
			switch req.Prompt {
			case "Fast":
				// No additional delay
			case "Medium":
				time.Sleep(20 * time.Millisecond)
			case "Slow":
				time.Sleep(40 * time.Millisecond)
			}
			return &types.GenerateResponse{
				Model:    req.Model,
				Response: req.Prompt + " response",
				Done:     true,
			}, nil
		},
	}

	parallel := NewParallel(mock, ParallelConfig{MaxParallel: 3})

	// Order: Slow, Fast, Medium
	prompts := []string{"Slow", "Fast", "Medium"}

	resp, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: prompts,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Results should be in original order regardless of completion order
	for i, prompt := range prompts {
		if resp.Results[i].Prompt != prompt {
			t.Errorf("Results[%d].Prompt = %q, want %q", i, resp.Results[i].Prompt, prompt)
		}
	}
}

// === Bounded Concurrency Tests ===

func TestParallelGenerator_BoundedConcurrency(t *testing.T) {
	// LEARN: Verify that maxParallel is respected
	mock := &mockGenerator{
		delay: 50 * time.Millisecond,
	}

	maxParallel := 2
	parallel := NewParallel(mock, ParallelConfig{MaxParallel: maxParallel})

	prompts := make([]string, 6)
	for i := range prompts {
		prompts[i] = "Prompt"
	}

	_, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: prompts,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Max concurrent should not exceed maxParallel
	if max := atomic.LoadInt32(&mock.maxConcurrent); max > int32(maxParallel) {
		t.Errorf("maxConcurrent = %d, want <= %d", max, maxParallel)
	}

	t.Logf("Max concurrent: %d (limit: %d)", mock.maxConcurrent, maxParallel)
}

func TestParallelGenerator_RequestMaxParallel(t *testing.T) {
	// LEARN: Request-level MaxParallel should override config
	mock := &mockGenerator{
		delay: 30 * time.Millisecond,
	}

	// Config allows 4, but request limits to 1
	parallel := NewParallel(mock, ParallelConfig{MaxParallel: 4})

	prompts := []string{"A", "B", "C", "D"}

	_, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:       "llama3.2",
		Prompts:     prompts,
		MaxParallel: 1, // Override to sequential
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With MaxParallel=1, max concurrent should be 1
	if max := atomic.LoadInt32(&mock.maxConcurrent); max != 1 {
		t.Errorf("maxConcurrent = %d, want 1", max)
	}
}

// === Error Handling Tests ===

func TestParallelGenerator_PartialFailures(t *testing.T) {
	// LEARN: Partial failures should not abort other requests
	failError := errors.New("simulated failure")
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			if req.Prompt == "fail" {
				return nil, failError
			}
			return &types.GenerateResponse{
				Model:    req.Model,
				Response: "Success: " + req.Prompt,
				Done:     true,
			}, nil
		},
	}

	parallel := NewParallelDefault(mock)

	resp, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: []string{"success1", "fail", "success2"},
	})

	if err != nil {
		t.Fatalf("GenerateParallel should not return error on partial failure: %v", err)
	}

	if resp.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", resp.SuccessCount)
	}

	if resp.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", resp.ErrorCount)
	}

	// Check individual results
	if resp.Results[0].Err != nil {
		t.Error("Results[0] should succeed")
	}
	if resp.Results[1].Err == nil {
		t.Error("Results[1] should fail")
	}
	if resp.Results[2].Err != nil {
		t.Error("Results[2] should succeed")
	}
}

func TestParallelGenerator_AllFailures(t *testing.T) {
	failError := errors.New("all fail")
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			return nil, failError
		},
	}

	parallel := NewParallelDefault(mock)

	resp, err := parallel.GenerateParallel(context.Background(), ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: []string{"a", "b", "c"},
	})

	if err != nil {
		t.Fatalf("GenerateParallel should not return error even when all fail: %v", err)
	}

	if resp.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", resp.SuccessCount)
	}

	if resp.ErrorCount != 3 {
		t.Errorf("ErrorCount = %d, want 3", resp.ErrorCount)
	}
}

// === Context Cancellation Tests ===

func TestParallelGenerator_ContextCancellation(t *testing.T) {
	// LEARN: Context cancellation should stop new requests
	started := make(chan struct{})
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			select {
			case started <- struct{}{}:
			default:
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return &types.GenerateResponse{
					Model:    req.Model,
					Response: req.Prompt,
					Done:     true,
				}, nil
			}
		},
	}

	parallel := NewParallel(mock, ParallelConfig{MaxParallel: 1}) // Sequential to make test deterministic

	ctx, cancel := context.WithCancel(context.Background())

	// Start in goroutine
	done := make(chan *ParallelGenerateResponse)
	go func() {
		resp, _ := parallel.GenerateParallel(ctx, ParallelGenerateRequest{
			Model:   "llama3.2",
			Prompts: []string{"first", "second", "third"},
		})
		done <- resp
	}()

	// Wait for first request to start
	<-started

	// Cancel context
	cancel()

	// Wait for completion
	select {
	case resp := <-done:
		// At least some should have been cancelled
		if resp.ErrorCount == 0 {
			t.Error("Expected some errors due to cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancellation")
	}
}

func TestParallelGenerator_ContextTimeout(t *testing.T) {
	mock := &mockGenerator{
		delay: 500 * time.Millisecond, // Longer than timeout
	}

	parallel := NewParallelDefault(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := parallel.GenerateParallel(ctx, ParallelGenerateRequest{
		Model:   "llama3.2",
		Prompts: []string{"a", "b"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All should timeout
	if resp.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0 (all should timeout)", resp.SuccessCount)
	}

	// Check errors are context deadline
	for i, result := range resp.Results {
		if !errors.Is(result.Err, context.DeadlineExceeded) && !errors.Is(result.Err, context.Canceled) {
			t.Errorf("Results[%d].Err = %v, want context error", i, result.Err)
		}
	}
}

// === Configuration Tests ===

func TestNewParallel_InvalidConfig(t *testing.T) {
	mock := &mockGenerator{}

	// Zero MaxParallel should use default
	parallel := NewParallel(mock, ParallelConfig{MaxParallel: 0})
	if parallel == nil {
		t.Fatal("NewParallel returned nil")
	}

	// Negative MaxParallel should use default
	parallel = NewParallel(mock, ParallelConfig{MaxParallel: -1})
	if parallel == nil {
		t.Fatal("NewParallel returned nil")
	}
}

// === Convenience Function Tests ===

func TestGenerateAll(t *testing.T) {
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			return &types.GenerateResponse{
				Model:    req.Model,
				Response: "Response: " + req.Prompt,
				Done:     true,
			}, nil
		},
	}

	responses, err := GenerateAll(context.Background(), mock, "llama3.2", []string{"a", "b", "c"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(responses) != 3 {
		t.Errorf("len(responses) = %d, want 3", len(responses))
	}
}

func TestGenerateAll_PartialFailure(t *testing.T) {
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			if req.Prompt == "fail" {
				return nil, errors.New("failed")
			}
			return &types.GenerateResponse{
				Response: "OK: " + req.Prompt,
			}, nil
		},
	}

	responses, err := GenerateAll(context.Background(), mock, "llama3.2", []string{"a", "fail", "b"})

	if err != nil {
		t.Fatalf("GenerateAll should not error on partial failure: %v", err)
	}

	// Should only contain successful responses
	if len(responses) != 2 {
		t.Errorf("len(responses) = %d, want 2", len(responses))
	}
}

func TestGenerateAllStrict(t *testing.T) {
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			return &types.GenerateResponse{
				Response: "OK: " + req.Prompt,
			}, nil
		},
	}

	responses, err := GenerateAllStrict(context.Background(), mock, "llama3.2", []string{"a", "b"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(responses) != 2 {
		t.Errorf("len(responses) = %d, want 2", len(responses))
	}
}

func TestGenerateAllStrict_FailsOnError(t *testing.T) {
	mock := &mockGenerator{
		generateFunc: func(ctx context.Context, req types.GenerateRequest) (*types.GenerateResponse, error) {
			if req.Prompt == "fail" {
				return nil, errors.New("test error")
			}
			return &types.GenerateResponse{Response: "OK"}, nil
		},
	}

	_, err := GenerateAllStrict(context.Background(), mock, "llama3.2", []string{"ok", "fail"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errors.New("test error")) {
		// The error should mention the failure
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

// === Benchmark Tests ===

func BenchmarkParallelGenerator_Small(b *testing.B) {
	mock := &mockGenerator{}
	parallel := NewParallelDefault(mock)

	prompts := []string{"a", "b", "c", "d"}
	req := ParallelGenerateRequest{Model: "llama3.2", Prompts: prompts}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel.GenerateParallel(context.Background(), req)
	}
}

func BenchmarkParallelGenerator_Large(b *testing.B) {
	mock := &mockGenerator{}
	parallel := NewParallel(mock, ParallelConfig{MaxParallel: 8})

	prompts := make([]string, 100)
	for i := range prompts {
		prompts[i] = "prompt"
	}
	req := ParallelGenerateRequest{Model: "llama3.2", Prompts: prompts}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel.GenerateParallel(context.Background(), req)
	}
}
