// internal/worker/pool.go
// Worker pool with bounded parallelism for concurrent task execution
//
// LEARN: Worker pools are a fundamental concurrency pattern that:
// 1. Limits resource usage (bounded parallelism)
// 2. Maximizes throughput (parallel execution)
// 3. Provides backpressure (buffered channels)
// 4. Supports graceful shutdown (context cancellation)
//
// This is Exercise 2.1 - implement the TODOs to complete the worker pool.

package worker

import (
	"context"
	"sync"
)

// Job represents a unit of work to be processed.
// LEARN: Using function type allows arbitrary work to be submitted.
type Job func(ctx context.Context) (any, error)

// Result contains the outcome of a processed job.
type Result struct {
	Value any   // Result value on success
	Err   error // Error on failure
	JobID int   // Identifier for ordering/correlation
}

// Pool manages a fixed number of worker goroutines.
//
// LEARN: The pool pattern uses channels to:
// - jobs channel: Distribute work to workers
// - results channel: Collect results from workers
// - WaitGroup: Track worker lifecycle
type Pool struct {
	workers int
	jobs    chan jobWrapper
	results chan Result
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// jobWrapper adds metadata to a job for tracking.
type jobWrapper struct {
	job   Job
	jobID int
}

// Config holds pool configuration.
type Config struct {
	Workers    int // Number of worker goroutines (default: runtime.NumCPU())
	JobBuffer  int // Size of job queue buffer (default: workers * 2)
	ResultBuffer int // Size of result buffer (default: workers * 2)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Workers:      4,
		JobBuffer:    8,
		ResultBuffer: 8,
	}
}

// New creates a new worker pool with the given configuration.
//
// LEARN: The constructor starts workers immediately. This is the "eager start"
// pattern. An alternative is "lazy start" where workers start on first submit.
func New(cfg Config) *Pool {
	if cfg.Workers <= 0 {
		cfg.Workers = DefaultConfig().Workers
	}
	if cfg.JobBuffer <= 0 {
		cfg.JobBuffer = cfg.Workers * 2
	}
	if cfg.ResultBuffer <= 0 {
		cfg.ResultBuffer = cfg.Workers * 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool{
		workers: cfg.Workers,
		jobs:    make(chan jobWrapper, cfg.JobBuffer),
		results: make(chan Result, cfg.ResultBuffer),
		ctx:     ctx,
		cancel:  cancel,
	}

	p.start()
	return p
}

// NewDefault creates a pool with default configuration.
func NewDefault() *Pool {
	return New(DefaultConfig())
}

// start launches the worker goroutines.
//
// LEARN: Each worker is a goroutine running an infinite loop that:
// 1. Waits for a job from the jobs channel
// 2. Executes the job
// 3. Sends the result to the results channel
// 4. Repeats until jobs channel is closed
func (p *Pool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the main loop for each worker goroutine.
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	// TODO Exercise 2.1: Implement the worker loop
	// 
	// Requirements:
	// 1. Loop over p.jobs channel (use range)
	// 2. Check p.ctx.Done() before processing each job
	// 3. Execute the job: value, err := jw.job(p.ctx)
	// 4. Send Result{Value: value, Err: err, JobID: jw.jobID} to p.results
	// 5. Continue until jobs channel is closed
	//
	// LEARN: The select statement allows checking context cancellation
	// while also waiting for jobs:
	//
	// for {
	//     select {
	//     case <-p.ctx.Done():
	//         return
	//     case jw, ok := <-p.jobs:
	//         if !ok {
	//             return // Channel closed
	//         }
	//         // Process job...
	//     }
	// }

	for {
		select {
		case <-p.ctx.Done():
			return
		case jw, ok := <-p.jobs:
			if !ok {
				return // Jobs channel closed, worker exits
			}

			// Execute the job with the pool's context
			value, err := jw.job(p.ctx)

			// Send result (non-blocking send might be needed for shutdown)
			select {
			case p.results <- Result{Value: value, Err: err, JobID: jw.jobID}:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// Submit adds a job to the pool for processing.
//
// LEARN: Submit blocks if the job buffer is full (backpressure).
// This prevents unbounded memory growth when jobs are submitted
// faster than they can be processed.
func (p *Pool) Submit(job Job) int {
	return p.SubmitWithID(job, 0)
}

// SubmitWithID adds a job with a specific ID for result correlation.
//
// LEARN: There's a race condition between Submit and Shutdown - the jobs
// channel might be closed while we're trying to send. We use defer/recover
// to handle this gracefully rather than panicking.
func (p *Pool) SubmitWithID(job Job, jobID int) (id int) {
	defer func() {
		if r := recover(); r != nil {
			// Channel was closed while we were trying to send
			id = -1
		}
	}()

	select {
	case <-p.ctx.Done():
		return -1 // Pool is shutting down
	case p.jobs <- jobWrapper{job: job, jobID: jobID}:
		return jobID
	}
}

// Results returns the results channel for consuming processed jobs.
//
// LEARN: Exposing the channel allows the caller to use range or select:
//
//	for result := range pool.Results() {
//	    process(result)
//	}
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Shutdown gracefully stops the pool.
//
// LEARN: Graceful shutdown sequence:
// 1. Signal no more jobs (close jobs channel)
// 2. Wait for workers to finish current jobs (wg.Wait)
// 3. Close results channel (safe because all workers are done)
// 4. Cancel context (cleanup)
func (p *Pool) Shutdown() {
	close(p.jobs)    // Step 1: Signal workers to stop accepting jobs
	p.wg.Wait()      // Step 2: Wait for all workers to complete
	close(p.results) // Step 3: Safe to close results now
	p.cancel()       // Step 4: Cancel context for cleanup
}

// ShutdownNow immediately cancels all in-progress work.
//
// LEARN: This is for emergency shutdown. Cancel context first to
// stop workers immediately, then clean up channels.
func (p *Pool) ShutdownNow() {
	p.cancel()       // Cancel context immediately
	close(p.jobs)    // Close jobs channel
	p.wg.Wait()      // Wait for workers to notice cancellation
	close(p.results) // Close results
}

// WorkerCount returns the number of workers in the pool.
func (p *Pool) WorkerCount() int {
	return p.workers
}

