// internal/ollama/client_test.go
// Table-driven tests for Ollama client
//
// LEARN: Table-driven tests are the standard Go testing pattern.
// They provide:
// 1. Clear documentation of expected behavior
// 2. Easy addition of new test cases
// 3. Consistent error reporting
// 4. DRY test code

package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/khaaliswooden-max/go_project/pkg/errors"
	"github.com/khaaliswooden-max/go_project/pkg/types"
)

// === Test Helpers ===

// LEARN: Test helpers should call t.Helper() to improve error reporting.
// When a helper fails, the test output shows the caller's line number,
// not the helper's.

// newTestServer creates a mock Ollama server with the given handler.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// newTestClient creates a client pointing to the test server.
func newTestClient(t *testing.T, serverURL string) Client {
	t.Helper()
	cfg := Config{
		BaseURL: serverURL,
		Timeout: 5 * time.Second,
	}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

// === Config Tests ===

func TestConfig_Validate(t *testing.T) {
	// LEARN: Table-driven test structure:
	// 1. Define test cases as a slice of structs
	// 2. Each case has a name, inputs, and expected outputs
	// 3. Loop through cases, calling t.Run() for each

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: Config{
				BaseURL: "http://custom:8080",
				Timeout: 60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing BaseURL",
			config: Config{
				BaseURL: "",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: Config{
				BaseURL: "http://localhost:11434",
				Timeout: 0,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: Config{
				BaseURL: "http://localhost:11434",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		// LEARN: t.Run() creates a subtest with its own name.
		// This enables running specific tests: go test -run TestConfig_Validate/missing_BaseURL
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// === Generate Tests ===

func TestClient_Generate(t *testing.T) {
	// LEARN: For HTTP tests, define both the mock response and expected behavior.
	// This makes the test self-documenting.

	tests := []struct {
		name           string
		request        types.GenerateRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantResponse   *types.GenerateResponse
		wantErr        bool
		wantErrType    error // For errors.Is() checking
	}{
		{
			name: "successful generation",
			request: types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "Hello, world!",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// LEARN: Verify the request was formed correctly
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/generate" {
					t.Errorf("expected /api/generate, got %s", r.URL.Path)
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", ct)
				}

				// Verify request body
				var req types.GenerateRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("failed to decode request: %v", err)
				}
				if req.Model != "llama3.2" {
					t.Errorf("expected model llama3.2, got %s", req.Model)
				}
				if req.Stream != false {
					t.Errorf("expected stream false for non-streaming call")
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(types.GenerateResponse{
					Model:         "llama3.2",
					Response:      "Hello! How can I help you?",
					Done:          true,
					TotalDuration: 1000000000, // 1 second in nanoseconds
				})
			},
			wantResponse: &types.GenerateResponse{
				Model:         "llama3.2",
				Response:      "Hello! How can I help you?",
				Done:          true,
				TotalDuration: 1000000000,
			},
			wantErr: false,
		},
		{
			name: "missing model returns error",
			request: types.GenerateRequest{
				Model:  "",
				Prompt: "Hello",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called for invalid request")
			},
			wantErr:     true,
			wantErrType: errors.ErrInvalidRequest,
		},
		{
			name: "missing prompt returns error",
			request: types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called for invalid request")
			},
			wantErr:     true,
			wantErrType: errors.ErrInvalidRequest,
		},
		{
			name: "model not found error",
			request: types.GenerateRequest{
				Model:  "nonexistent-model",
				Prompt: "Hello",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "model 'nonexistent-model' not found",
				})
			},
			wantErr:     true,
			wantErrType: errors.ErrModelNotFound,
		},
		{
			name: "server error",
			request: types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "Hello",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "internal server error",
				})
			},
			wantErr:     true,
			wantErrType: errors.ErrOllamaUnavailable,
		},
		{
			name: "invalid JSON response",
			request: types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "Hello",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("not valid json"))
			},
			wantErr:     true,
			wantErrType: errors.ErrInvalidJSON,
		},
		{
			name: "with system prompt",
			request: types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "What is Go?",
				System: "You are a programming assistant.",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var req types.GenerateRequest
				json.NewDecoder(r.Body).Decode(&req)
				if req.System != "You are a programming assistant." {
					t.Errorf("system prompt not passed correctly")
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(types.GenerateResponse{
					Model:    "llama3.2",
					Response: "Go is a programming language.",
					Done:     true,
				})
			},
			wantResponse: &types.GenerateResponse{
				Model:    "llama3.2",
				Response: "Go is a programming language.",
				Done:     true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := newTestServer(t, tt.serverResponse)
			defer server.Close()

			// Create client
			client := newTestClient(t, server.URL)

			// Execute
			ctx := context.Background()
			resp, err := client.Generate(ctx, tt.request)

			// Verify error
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// LEARN: Use errors.Is() to check for wrapped errors
			if tt.wantErrType != nil && err != nil {
				if !errors.Is(err, tt.wantErrType) {
					t.Errorf("Generate() error type = %T, want %T (error: %v)", err, tt.wantErrType, err)
				}
			}

			// Verify response
			if tt.wantResponse != nil {
				if resp == nil {
					t.Fatal("expected response, got nil")
				}
				if resp.Model != tt.wantResponse.Model {
					t.Errorf("Model = %v, want %v", resp.Model, tt.wantResponse.Model)
				}
				if resp.Response != tt.wantResponse.Response {
					t.Errorf("Response = %v, want %v", resp.Response, tt.wantResponse.Response)
				}
				if resp.Done != tt.wantResponse.Done {
					t.Errorf("Done = %v, want %v", resp.Done, tt.wantResponse.Done)
				}
			}
		})
	}
}

// === Context Cancellation Tests ===

func TestClient_Generate_ContextCancellation(t *testing.T) {
	// LEARN: Testing context cancellation requires a slow server
	// and a pre-canceled or timed-out context.

	tests := []struct {
		name        string
		setupCtx    func() (context.Context, context.CancelFunc)
		wantErrType error
	}{
		{
			name: "canceled context",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			wantErrType: errors.ErrRequestCanceled,
		},
		{
			name: "timeout context",
			setupCtx: func() (context.Context, context.CancelFunc) {
				// Create already-expired context
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(1 * time.Millisecond) // Ensure timeout
				return ctx, cancel
			},
			wantErrType: errors.ErrRequestTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Slow server that never responds
			server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Second)
			})
			defer server.Close()

			client := newTestClient(t, server.URL)
			ctx, cancel := tt.setupCtx()
			defer cancel()

			_, err := client.Generate(ctx, types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "Hello",
			})

			if err == nil {
				t.Error("expected error for canceled/timed out context")
			}
			// Note: The exact error depends on when cancellation occurs
			// Just verify we get an error
		})
	}
}

// === Streaming Tests ===

func TestClient_GenerateStream(t *testing.T) {
	tests := []struct {
		name           string
		request        types.GenerateRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantChunks     int
		wantErr        bool
	}{
		{
			name: "successful streaming",
			request: types.GenerateRequest{
				Model:  "llama3.2",
				Prompt: "Count to 3",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify streaming is requested
				var req types.GenerateRequest
				json.NewDecoder(r.Body).Decode(&req)
				if !req.Stream {
					t.Error("expected stream=true")
				}

				// Send NDJSON chunks
				// LEARN: Ollama uses newline-delimited JSON for streaming
				w.Header().Set("Content-Type", "application/x-ndjson")
				flusher := w.(http.Flusher)

				chunks := []types.GenerateChunk{
					{Model: "llama3.2", Response: "1", Done: false},
					{Model: "llama3.2", Response: "2", Done: false},
					{Model: "llama3.2", Response: "3", Done: true, TotalDuration: 1000000000},
				}

				for _, chunk := range chunks {
					json.NewEncoder(w).Encode(chunk)
					flusher.Flush()
				}
			},
			wantChunks: 3,
			wantErr:    false,
		},
		{
			name: "validation error before stream",
			request: types.GenerateRequest{
				Model:  "",
				Prompt: "Hello",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				t.Error("server should not be called")
			},
			wantChunks: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t, tt.serverResponse)
			defer server.Close()

			client := newTestClient(t, server.URL)
			ctx := context.Background()

			chunks, errs := client.GenerateStream(ctx, tt.request)

			// Collect chunks
			var received []types.GenerateChunk
			for chunk := range chunks {
				received = append(received, chunk)
			}

			// Check for errors
			var streamErr error
			select {
			case streamErr = <-errs:
			default:
			}

			if (streamErr != nil) != tt.wantErr {
				t.Errorf("GenerateStream() error = %v, wantErr %v", streamErr, tt.wantErr)
			}

			if len(received) != tt.wantChunks {
				t.Errorf("received %d chunks, want %d", len(received), tt.wantChunks)
			}
		})
	}
}

// === Health Check Tests ===

func TestClient_Health(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantConnected  bool
		wantModels     []string
	}{
		{
			name: "healthy with models",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					t.Errorf("expected /api/tags, got %s", r.URL.Path)
				}
				json.NewEncoder(w).Encode(map[string]any{
					"models": []map[string]string{
						{"name": "llama3.2"},
						{"name": "mistral"},
					},
				})
			},
			wantConnected: true,
			wantModels:    []string{"llama3.2", "mistral"},
		},
		{
			name: "healthy with no models",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{
					"models": []map[string]string{},
				})
			},
			wantConnected: true,
			wantModels:    []string{},
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantConnected: false,
			wantModels:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t, tt.serverResponse)
			defer server.Close()

			client := newTestClient(t, server.URL)
			ctx := context.Background()

			health, err := client.Health(ctx)

			// Health check returns health struct even on error
			if health == nil {
				t.Fatal("expected health struct, got nil")
			}

			if health.Connected != tt.wantConnected {
				t.Errorf("Connected = %v, want %v", health.Connected, tt.wantConnected)
			}

			if tt.wantModels != nil {
				if len(health.Models) != len(tt.wantModels) {
					t.Errorf("Models count = %d, want %d", len(health.Models), len(tt.wantModels))
				}
				for i, model := range health.Models {
					if i < len(tt.wantModels) && model != tt.wantModels[i] {
						t.Errorf("Model[%d] = %s, want %s", i, model, tt.wantModels[i])
					}
				}
			}

			// If not connected, should have error (either returned or in health.Error)
			if !tt.wantConnected && err == nil && health.Error == "" {
				t.Error("expected error or health.Error for unhealthy state")
			}
		})
	}
}

// === Benchmark Tests ===
// LEARN: Benchmarks use testing.B and measure performance.
// Run with: go test -bench=. -benchmem

func BenchmarkClient_Generate(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.GenerateResponse{
			Model:    "llama3.2",
			Response: "Hello!",
			Done:     true,
		})
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, Timeout: 5 * time.Second}
	client, _ := New(cfg)
	ctx := context.Background()
	req := types.GenerateRequest{Model: "llama3.2", Prompt: "Hello"}

	b.ResetTimer() // Don't count setup time

	for i := 0; i < b.N; i++ {
		_, err := client.Generate(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// === Example Tests ===
// LEARN: Example tests serve as documentation and are verified by go test.
// The output comment must match exactly.

func ExampleNew() {
	cfg := Config{
		BaseURL: "http://localhost:11434",
		Timeout: 30 * time.Second,
	}

	client, err := New(cfg)
	if err != nil {
		panic(err)
	}

	// Check health
	ctx := context.Background()
	health, _ := client.Health(ctx)
	if health.Connected {
		println("Connected to Ollama")
	}
}
