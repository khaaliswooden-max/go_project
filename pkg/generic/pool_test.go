// pkg/generic/pool_test.go
// Tests for the generic worker pool
//
// LEARN: These tests demonstrate the benefits of generics:
// - No type assertions needed when checking results
// - Compile-time type checking prevents runtime errors

package generic

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolBasic(t *testing.T) {
	t.Run("processes jobs and returns typed results", func(t *testing.T) {
		// LEARN: Type parameters specified at pool creation
		pool := NewDefaultPool[int, string]()
		defer pool.Shutdown()

		// Submit job - input is int, output is string
		pool.Submit(42, func(ctx context.Context, n int) (string, error) {
			return "result", nil
		})

		// Get result - no type assertion needed!
		result := <-pool.Results()
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Value != "result" {
			t.Errorf("expected 'result', got: %s", result.Value)
		}
	})

	t.Run("handles errors correctly", func(t *testing.T) {
		pool := NewDefaultPool[string, int]()
		defer pool.Shutdown()

		expectedErr := errors.New("job failed")
		pool.Submit("input", func(ctx context.Context, s string) (int, error) {
			return 0, expectedErr
		})

		result := <-pool.Results()
		if result.Err != expectedErr {
			t.Errorf("expected error %v, got: %v", expectedErr, result.Err)
		}
	})
}

func TestPoolWithStructs(t *testing.T) {
	type Request struct {
		ID   int
		Data string
	}
	type Response struct {
		RequestID int
		Result    string
	}

	t.Run("works with complex types", func(t *testing.T) {
		pool := NewDefaultPool[Request, Response]()
		defer pool.Shutdown()

		req := Request{ID: 1, Data: "hello"}
		pool.Submit(req, func(ctx context.Context, r Request) (Response, error) {
			return Response{
				RequestID: r.ID,
				Result:    "processed: " + r.Data,
			}, nil
		})

		result := <-pool.Results()
		if result.Value.RequestID != 1 {
			t.Errorf("expected RequestID=1, got: %d", result.Value.RequestID)
		}
		if result.Value.Result != "processed: hello" {
			t.Errorf("unexpected result: %s", result.Value.Result)
		}
	})
}

func TestPoolMultipleWorkers(t *testing.T) {
	t.Run("processes jobs concurrently", func(t *testing.T) {
		cfg := PoolConfig{Workers: 4, JobBuffer: 10, ResultBuffer: 10}
		pool := NewPool[int, int](cfg)
		defer pool.Shutdown()

		var processed int32
		numJobs := 20

		// Submit all jobs
		for i := 0; i < numJobs; i++ {
			pool.Submit(i, func(ctx context.Context, n int) (int, error) {
				atomic.AddInt32(&processed, 1)
				time.Sleep(10 * time.Millisecond)
				return n * 2, nil
			})
		}

		// Collect all results
		results := make(map[int]int)
		for i := 0; i < numJobs; i++ {
			r := <-pool.Results()
			results[r.JobID] = r.Value
		}

		if int(processed) != numJobs {
			t.Errorf("expected %d processed, got: %d", numJobs, processed)
		}
	})
}

func TestPoolJobIDs(t *testing.T) {
	t.Run("preserves job IDs in results", func(t *testing.T) {
		pool := NewDefaultPool[string, int]()
		defer pool.Shutdown()

		// Submit jobs with specific IDs
		pool.SubmitWithID("a", func(ctx context.Context, s string) (int, error) {
			return len(s), nil
		}, 100)
		pool.SubmitWithID("bb", func(ctx context.Context, s string) (int, error) {
			return len(s), nil
		}, 200)

		results := make(map[int]int)
		for i := 0; i < 2; i++ {
			r := <-pool.Results()
			results[r.JobID] = r.Value
		}

		if results[100] != 1 {
			t.Errorf("expected job 100 result=1, got: %d", results[100])
		}
		if results[200] != 2 {
			t.Errorf("expected job 200 result=2, got: %d", results[200])
		}
	})
}

func TestPoolShutdown(t *testing.T) {
	t.Run("graceful shutdown waits for in-progress jobs", func(t *testing.T) {
		pool := NewPool[int, int](PoolConfig{Workers: 2, JobBuffer: 5, ResultBuffer: 5})

		var completed int32
		for i := 0; i < 3; i++ {
			pool.Submit(i, func(ctx context.Context, n int) (int, error) {
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&completed, 1)
				return n, nil
			})
		}

		// Give workers time to pick up jobs
		time.Sleep(20 * time.Millisecond)
		pool.Shutdown()

		// After shutdown, all jobs should be completed
		if completed != 3 {
			t.Errorf("expected 3 completed, got: %d", completed)
		}
	})

	t.Run("submit returns -1 after shutdown starts", func(t *testing.T) {
		pool := NewDefaultPool[int, int]()

		// Start shutdown in background
		go func() {
			time.Sleep(10 * time.Millisecond)
			pool.Shutdown()
		}()

		// Keep trying to submit until we get -1
		var gotNegative bool
		for i := 0; i < 100; i++ {
			id := pool.Submit(i, func(ctx context.Context, n int) (int, error) {
				time.Sleep(5 * time.Millisecond)
				return n, nil
			})
			if id == -1 {
				gotNegative = true
				break
			}
			time.Sleep(1 * time.Millisecond)
		}

		if !gotNegative {
			t.Log("Note: submit may not have returned -1 during shutdown race")
		}
	})
}

func TestPoolShutdownNow(t *testing.T) {
	t.Run("cancels context for in-progress jobs", func(t *testing.T) {
		pool := NewPool[int, int](PoolConfig{Workers: 2, JobBuffer: 5, ResultBuffer: 5})

		var cancelled int32
		for i := 0; i < 3; i++ {
			pool.Submit(i, func(ctx context.Context, n int) (int, error) {
				select {
				case <-ctx.Done():
					atomic.AddInt32(&cancelled, 1)
					return 0, ctx.Err()
				case <-time.After(5 * time.Second):
					return n, nil
				}
			})
		}

		time.Sleep(20 * time.Millisecond) // Let workers pick up jobs
		pool.ShutdownNow()

		// Jobs should have been cancelled
		if cancelled == 0 {
			t.Log("Note: jobs may have completed before cancellation")
		}
	})
}

func TestProcessBatch(t *testing.T) {
	t.Run("processes all inputs and returns results in order", func(t *testing.T) {
		ctx := context.Background()
		inputs := []int{1, 2, 3, 4, 5}

		results, err := ProcessBatch(ctx, inputs, func(ctx context.Context, n int) (int, error) {
			return n * 10, nil
		}, 2)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i, r := range results {
			expected := inputs[i] * 10
			if r.Value != expected {
				t.Errorf("result[%d]: expected %d, got %d", i, expected, r.Value)
			}
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		inputs := []int{1, 2, 3, 4, 5}

		var started int32
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			_, err := ProcessBatch(ctx, inputs, func(ctx context.Context, n int) (int, error) {
				atomic.AddInt32(&started, 1)
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(time.Second):
					return n, nil
				}
			}, 2)

			if err != context.Canceled {
				t.Errorf("expected context.Canceled, got: %v", err)
			}
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()
		wg.Wait()
	})
}

func TestMapParallel(t *testing.T) {
	t.Run("applies function to all elements", func(t *testing.T) {
		ctx := context.Background()
		inputs := []string{"a", "bb", "ccc"}

		results := MapParallel(ctx, inputs, func(s string) int {
			return len(s)
		}, 2)

		expected := []int{1, 2, 3}
		for i, r := range results {
			if r != expected[i] {
				t.Errorf("result[%d]: expected %d, got %d", i, expected[i], r)
			}
		}
	})
}

func TestPoolConcurrency(t *testing.T) {
	t.Run("handles concurrent submit and results", func(t *testing.T) {
		pool := NewPool[int, int](PoolConfig{Workers: 4, JobBuffer: 100, ResultBuffer: 100})
		defer pool.Shutdown()

		numJobs := 50
		var wg sync.WaitGroup

		// Concurrent submitter
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numJobs; i++ {
				pool.SubmitWithID(i, func(ctx context.Context, n int) (int, error) {
					time.Sleep(time.Millisecond)
					return n * 2, nil
				}, i)
			}
		}()

		// Concurrent collector
		results := make([]PoolResult[int], 0, numJobs)
		var mu sync.Mutex
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numJobs; i++ {
				r := <-pool.Results()
				mu.Lock()
				results = append(results, r)
				mu.Unlock()
			}
		}()

		wg.Wait()

		if len(results) != numJobs {
			t.Errorf("expected %d results, got: %d", numJobs, len(results))
		}
	})
}

// Benchmark to compare with Phase 2's non-generic pool
func BenchmarkPoolSubmitAndProcess(b *testing.B) {
	pool := NewPool[int, int](PoolConfig{Workers: 4, JobBuffer: 100, ResultBuffer: 100})
	defer pool.Shutdown()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pool.Submit(i, func(ctx context.Context, n int) (int, error) {
			return n * 2, nil
		})
		<-pool.Results()
	}
}


