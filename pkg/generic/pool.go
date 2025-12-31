// pkg/generic/pool.go
// Generic worker pool with type-safe inputs and outputs
//
// LEARN: This is a generic version of the Phase 2 worker pool.
// Benefits of generics here:
// 1. No type assertions needed when submitting jobs or reading results
// 2. Compile-time type checking for job functions
// 3. Better IDE support with proper type information
//
// This is Exercise 3.1 - implement the TODOs to complete the generic pool.

package generic

import (
	"context"
	"sync"
)

// Job represents a unit of work with typed input and output.
//
// LEARN: Compare to Phase 2's Job type:
// - Phase 2: type Job func(ctx context.Context) (any, error)
// - Phase 3: type Job[In, Out any] struct { Input In; Fn func(In) Out }
//
// The generic version provides type safety for both inputs and outputs.
type Job[In, Out any] struct {
	Input In
	Fn    func(ctx context.Context, input In) (Out, error)
	ID    int
}

// PoolResult contains the typed outcome of a processed job.
//
// LEARN: Compare to Phase 2's Result:
// - Phase 2: Value any (requires type assertion)
// - Phase 3: Value Out (type-safe, no assertion needed)
type PoolResult[Out any] struct {
	Value Out
	Err   error
	JobID int
}

// Pool manages a fixed number of worker goroutines with typed jobs.
//
// LEARN: The type parameters In and Out flow through the entire pool:
// - jobs channel carries Job[In, Out]
// - results channel carries PoolResult[Out]
// - All operations are type-safe at compile time
type Pool[In, Out any] struct {
	workers int
	jobs    chan Job[In, Out]
	results chan PoolResult[Out]
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// PoolConfig holds pool configuration.
type PoolConfig struct {
	Workers      int // Number of worker goroutines (default: 4)
	JobBuffer    int // Size of job queue buffer (default: workers * 2)
	ResultBuffer int // Size of result buffer (default: workers * 2)
}

// DefaultPoolConfig returns sensible defaults.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		Workers:      4,
		JobBuffer:    8,
		ResultBuffer: 8,
	}
}

// NewPool creates a new typed worker pool.
//
// LEARN: The constructor must specify type parameters:
//
//	pool := NewPool[string, int](config)  // string input, int output
//
// Or with type inference from a handler function:
//
//	pool := NewPoolWithHandler(config, myHandler)
func NewPool[In, Out any](cfg PoolConfig) *Pool[In, Out] {
	if cfg.Workers <= 0 {
		cfg.Workers = DefaultPoolConfig().Workers
	}
	if cfg.JobBuffer <= 0 {
		cfg.JobBuffer = cfg.Workers * 2
	}
	if cfg.ResultBuffer <= 0 {
		cfg.ResultBuffer = cfg.Workers * 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool[In, Out]{
		workers: cfg.Workers,
		jobs:    make(chan Job[In, Out], cfg.JobBuffer),
		results: make(chan PoolResult[Out], cfg.ResultBuffer),
		ctx:     ctx,
		cancel:  cancel,
	}

	p.start()
	return p
}

// NewDefaultPool creates a pool with default configuration.
func NewDefaultPool[In, Out any]() *Pool[In, Out] {
	return NewPool[In, Out](DefaultPoolConfig())
}

// start launches the worker goroutines.
func (p *Pool[In, Out]) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the main loop for each worker goroutine.
//
// LEARN: The worker processes Job[In, Out] and produces PoolResult[Out].
// All type information is preserved throughout the pipeline.
func (p *Pool[In, Out]) worker(id int) {
	defer p.wg.Done()

	// TODO Exercise 3.1: Implement the worker loop
	//
	// This should be similar to Phase 2, but with typed jobs and results.
	//
	// Requirements:
	// 1. Use select to check both p.ctx.Done() and p.jobs
	// 2. When a job arrives, execute: value, err := job.Fn(p.ctx, job.Input)
	// 3. Send PoolResult[Out]{Value: value, Err: err, JobID: job.ID}
	// 4. Handle context cancellation gracefully
	//
	// HINT: The implementation is nearly identical to Phase 2,
	// but now everything is type-safe!

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return // Channel closed
			}

			// Execute the typed job function
			value, err := job.Fn(p.ctx, job.Input)

			// Send typed result
			select {
			case p.results <- PoolResult[Out]{Value: value, Err: err, JobID: job.ID}:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// Submit adds a job to the pool for processing.
//
// LEARN: Compare to Phase 2's Submit:
// - Phase 2: pool.Submit(func(ctx) (any, error) { return processString(s), nil })
// - Phase 3: pool.Submit(s, processFunc) // Type-safe!
//
// The input type must match the pool's In type parameter.
func (p *Pool[In, Out]) Submit(input In, fn func(context.Context, In) (Out, error)) int {
	return p.SubmitWithID(input, fn, 0)
}

// SubmitWithID adds a job with a specific ID for result correlation.
func (p *Pool[In, Out]) SubmitWithID(input In, fn func(context.Context, In) (Out, error), jobID int) (id int) {
	defer func() {
		if r := recover(); r != nil {
			id = -1
		}
	}()

	select {
	case <-p.ctx.Done():
		return -1
	case p.jobs <- Job[In, Out]{Input: input, Fn: fn, ID: jobID}:
		return jobID
	}
}

// SubmitJob adds a pre-constructed job to the pool.
//
// LEARN: This variant is useful when you've already created the Job struct,
// perhaps from a job factory or when resubmitting failed jobs.
func (p *Pool[In, Out]) SubmitJob(job Job[In, Out]) int {
	defer func() {
		if recover() != nil {
			// Swallow panic from closed channel
		}
	}()

	select {
	case <-p.ctx.Done():
		return -1
	case p.jobs <- job:
		return job.ID
	}
}

// Results returns the typed results channel.
//
// LEARN: The result type is PoolResult[Out], so no type assertion needed:
//
//	for result := range pool.Results() {
//	    if result.Err != nil { ... }
//	    process(result.Value) // Value is already the correct type!
//	}
func (p *Pool[In, Out]) Results() <-chan PoolResult[Out] {
	return p.results
}

// Shutdown gracefully stops the pool.
func (p *Pool[In, Out]) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
	p.cancel()
}

// ShutdownNow immediately cancels all in-progress work.
func (p *Pool[In, Out]) ShutdownNow() {
	p.cancel()
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// WorkerCount returns the number of workers in the pool.
func (p *Pool[In, Out]) WorkerCount() int {
	return p.workers
}

// === Utility Functions ===

// ProcessBatch submits multiple inputs and collects all results.
//
// LEARN: This is a common pattern for batch processing:
// 1. Submit all inputs
// 2. Collect all results
// 3. Return when all are complete
//
// The generic version ensures type safety for both inputs and outputs.
func ProcessBatch[In, Out any](
	ctx context.Context,
	inputs []In,
	fn func(context.Context, In) (Out, error),
	workers int,
) ([]PoolResult[Out], error) {
	if workers <= 0 {
		workers = 4
	}

	pool := NewPool[In, Out](PoolConfig{
		Workers:      workers,
		JobBuffer:    len(inputs),
		ResultBuffer: len(inputs),
	})
	defer pool.Shutdown()

	// Submit all jobs
	for i, input := range inputs {
		pool.SubmitWithID(input, fn, i)
	}

	// Collect results
	results := make([]PoolResult[Out], len(inputs))
	for i := 0; i < len(inputs); i++ {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case result := <-pool.Results():
			results[result.JobID] = result
		}
	}

	return results, nil
}

// MapParallel applies a function to each element in parallel.
//
// LEARN: This combines the worker pool with the Map pattern from
// functional programming. It's similar to parallel map in other languages.
func MapParallel[In, Out any](
	ctx context.Context,
	inputs []In,
	fn func(In) Out,
	workers int,
) []Out {
	wrapper := func(_ context.Context, in In) (Out, error) {
		return fn(in), nil
	}

	results, _ := ProcessBatch(ctx, inputs, wrapper, workers)

	outputs := make([]Out, len(inputs))
	for i, r := range results {
		outputs[i] = r.Value
	}
	return outputs
}


