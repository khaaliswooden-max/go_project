// pkg/perf/profile.go
// pprof integration for profiling
//
// LEARN: This is Exercise 4.3 - integrate pprof for production profiling.
//
// pprof provides:
// - CPU profiling: Where is time spent?
// - Heap profiling: What's using memory?
// - Goroutine profiling: What are goroutines doing?
// - Block profiling: Where do goroutines block?
// - Mutex profiling: Where is lock contention?

package perf

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
)

// RegisterPprof adds pprof handlers to a ServeMux.
//
// LEARN: pprof endpoints provide runtime profiling data:
//   - /debug/pprof/           - Index of available profiles
//   - /debug/pprof/profile    - CPU profile (30s by default)
//   - /debug/pprof/heap       - Heap memory allocations
//   - /debug/pprof/goroutine  - All goroutine stack traces
//   - /debug/pprof/allocs     - Allocation profile
//   - /debug/pprof/block      - Goroutine blocking profile
//   - /debug/pprof/mutex      - Mutex contention profile
//   - /debug/pprof/trace      - Execution tracer
//
// Usage after registering:
//
//	# Collect 30-second CPU profile
//	go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
//
//	# View heap profile
//	go tool pprof http://localhost:8080/debug/pprof/heap
//
//	# Interactive web UI
//	go tool pprof -http=:9090 http://localhost:8080/debug/pprof/heap
func RegisterPprof(mux *http.ServeMux) {
	// Standard pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Named profiles
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
}

// RegisterPprofWithPrefix adds pprof handlers with a custom path prefix.
//
// LEARN: Use a custom prefix for:
// - Avoiding conflicts with application routes
// - Adding security through obscurity (though proper auth is better)
// - API versioning
func RegisterPprofWithPrefix(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/", pprof.Index)
	mux.HandleFunc(prefix+"/cmdline", pprof.Cmdline)
	mux.HandleFunc(prefix+"/profile", pprof.Profile)
	mux.HandleFunc(prefix+"/symbol", pprof.Symbol)
	mux.HandleFunc(prefix+"/trace", pprof.Trace)

	mux.Handle(prefix+"/heap", pprof.Handler("heap"))
	mux.Handle(prefix+"/goroutine", pprof.Handler("goroutine"))
	mux.Handle(prefix+"/allocs", pprof.Handler("allocs"))
	mux.Handle(prefix+"/block", pprof.Handler("block"))
	mux.Handle(prefix+"/mutex", pprof.Handler("mutex"))
	mux.Handle(prefix+"/threadcreate", pprof.Handler("threadcreate"))
}

// EnableBlockProfiling turns on goroutine blocking profiler.
//
// LEARN: Block profiling is disabled by default for performance.
// Enable it when debugging concurrency issues.
//
// rate controls sampling:
//   - 0: Disable
//   - 1: Profile all blocking events (expensive!)
//   - >1: Sample every Nth nanosecond of blocking
//
// Recommended: Use rate=1 only during debugging, not production.
func EnableBlockProfiling(rate int) {
	runtime.SetBlockProfileRate(rate)
}

// EnableMutexProfiling turns on mutex contention profiler.
//
// LEARN: Mutex profiling helps identify lock contention.
//
// fraction controls sampling:
//   - 0: Disable
//   - 1: Profile all mutex events
//   - N: Report 1/N of contention events
//
// Recommended: Use fraction >= 10 in production.
func EnableMutexProfiling(fraction int) {
	runtime.SetMutexProfileFraction(fraction)
}

// RuntimeStats returns current runtime statistics.
//
// LEARN: These stats are useful for monitoring without
// the overhead of full pprof profiles.
type RuntimeStats struct {
	// Memory
	HeapAlloc    uint64 // Bytes allocated on heap
	HeapInuse    uint64 // Bytes in in-use spans
	HeapSys      uint64 // Bytes obtained from OS for heap
	HeapObjects  uint64 // Number of allocated objects
	TotalAlloc   uint64 // Cumulative bytes allocated
	Mallocs      uint64 // Cumulative number of allocations
	Frees        uint64 // Cumulative number of frees
	GCCycles     uint32 // Number of completed GC cycles
	NextGC       uint64 // Target heap size of next GC
	LastGCPauseN uint64 // Duration of last GC pause (ns)

	// Goroutines
	NumGoroutine int // Current number of goroutines

	// CPU
	NumCPU int // Number of CPUs available
}

// GetRuntimeStats collects current runtime statistics.
//
// LEARN: This is cheaper than pprof profiles and suitable
// for continuous monitoring or health endpoints.
func GetRuntimeStats() RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return RuntimeStats{
		HeapAlloc:    m.HeapAlloc,
		HeapInuse:    m.HeapInuse,
		HeapSys:      m.HeapSys,
		HeapObjects:  m.HeapObjects,
		TotalAlloc:   m.TotalAlloc,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		GCCycles:     m.NumGC,
		NextGC:       m.NextGC,
		LastGCPauseN: m.PauseNs[(m.NumGC+255)%256],
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
	}
}

// FormatRuntimeStats returns a human-readable stats summary.
func FormatRuntimeStats(s RuntimeStats) string {
	return fmt.Sprintf(
		"Heap: %s (in-use: %s, objects: %d)\n"+
			"Total Allocs: %s (%d mallocs, %d frees)\n"+
			"GC Cycles: %d, Last Pause: %dns, Next GC: %s\n"+
			"Goroutines: %d, CPUs: %d",
		formatBytes(s.HeapAlloc),
		formatBytes(s.HeapInuse),
		s.HeapObjects,
		formatBytes(s.TotalAlloc),
		s.Mallocs,
		s.Frees,
		s.GCCycles,
		s.LastGCPauseN,
		formatBytes(s.NextGC),
		s.NumGoroutine,
		s.NumCPU,
	)
}

// formatBytes formats bytes as human-readable string.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// HealthHandler returns an HTTP handler for runtime health checks.
//
// LEARN: This is useful for:
// - Kubernetes liveness/readiness probes
// - Load balancer health checks
// - Quick system status overview
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := GetRuntimeStats()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Status: OK\n\n%s\n", FormatRuntimeStats(stats))
	}
}

// JSONHealthHandler returns runtime stats as JSON.
func JSONHealthHandler() http.HandlerFunc {
	encoder := NewJSONEncoder(1024)
	return func(w http.ResponseWriter, r *http.Request) {
		stats := GetRuntimeStats()
		w.Header().Set("Content-Type", "application/json")
		if err := encoder.EncodeTo(w, stats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// GCHandler triggers a manual garbage collection.
//
// LEARN: Manual GC is rarely needed but can be useful for:
// - Reducing memory before taking heap profile
// - Testing GC behavior
// - Freeing memory immediately after large operation
//
// WARNING: This is expensive! Don't expose publicly.
func GCHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		before := GetRuntimeStats()
		runtime.GC()
		after := GetRuntimeStats()

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "GC Complete\nBefore: %s\nAfter: %s\nFreed: %s\n",
			formatBytes(before.HeapAlloc),
			formatBytes(after.HeapAlloc),
			formatBytes(before.HeapAlloc-after.HeapAlloc),
		)
	}
}

