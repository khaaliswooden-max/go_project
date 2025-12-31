// pkg/systems/cgo.go
// CGO integration examples - calling C from Go
//
// LEARN: CGO enables Go programs to call C functions and use C libraries.
// This file demonstrates common patterns and pitfalls.
//
// Build requirements:
// - C compiler (gcc, clang, or MSVC)
// - CGO_ENABLED=1 (default on most platforms)
//
// Key concepts:
// 1. C code is embedded in comments before "import C"
// 2. C types and functions are accessed via the C pseudo-package
// 3. Memory must be manually managed (C.free for C.malloc)
// 4. Each CGO call has ~100ns overhead

//go:build cgo

package systems

/*
#include <stdlib.h>
#include <string.h>
#include <math.h>

// LEARN: Inline C functions can be defined in the comment block
// They're compiled with the Go package

// Simple function demonstrating basic CGO
int c_add(int a, int b) {
    return a + b;
}

// String length in C
size_t c_strlen(const char* s) {
    return strlen(s);
}

// Mathematical function
double c_distance(double x1, double y1, double x2, double y2) {
    double dx = x2 - x1;
    double dy = y2 - y1;
    return sqrt(dx*dx + dy*dy);
}

// Struct example
typedef struct {
    int x;
    int y;
} Point;

Point c_make_point(int x, int y) {
    Point p = {x, y};
    return p;
}

// Array processing in C (demonstrates batching)
long long c_sum_array(int* arr, int len) {
    long long sum = 0;
    for (int i = 0; i < len; i++) {
        sum += arr[i];
    }
    return sum;
}
*/
import "C"

import (
	"unsafe"
)

// === Basic CGO Examples ===

// CAdd demonstrates the simplest CGO call.
//
// LEARN: C types need explicit conversion:
// - Go int -> C.int
// - C.int -> Go int
func CAdd(a, b int) int {
	return int(C.c_add(C.int(a), C.int(b)))
}

// CStrlen demonstrates string passing to C.
//
// LEARN: Go strings must be converted to C strings:
// 1. C.CString allocates memory (malloc)
// 2. You MUST call C.free to avoid leaks
// 3. The memory is NOT garbage collected
func CStrlen(s string) int {
	// Convert Go string to C string
	cs := C.CString(s)
	
	// CRITICAL: Free the C string when done
	defer C.free(unsafe.Pointer(cs))
	
	// Call C function
	return int(C.c_strlen(cs))
}

// CDistance demonstrates returning C doubles.
//
// LEARN: Floating point types convert directly:
// - Go float64 -> C.double
// - C.double -> Go float64
func CDistance(x1, y1, x2, y2 float64) float64 {
	return float64(C.c_distance(
		C.double(x1), C.double(y1),
		C.double(x2), C.double(y2),
	))
}

// === Struct Examples ===

// Point represents a 2D point (mirrors C struct).
type Point struct {
	X, Y int
}

// CMakePoint demonstrates creating C structs.
//
// LEARN: Go structs and C structs with the same layout
// can be converted, but be careful of:
// - Padding differences
// - Field ordering
// - Type size differences (int vs int32)
func CMakePoint(x, y int) Point {
	cp := C.c_make_point(C.int(x), C.int(y))
	return Point{X: int(cp.x), Y: int(cp.y)}
}

// === Array/Slice Examples ===

// CSumArray demonstrates passing Go slices to C.
//
// LEARN: Passing slices to C requires getting the underlying pointer.
// The slice MUST remain valid during the C call (no GC).
// Go 1.21+ provides runtime.Pinner for this purpose.
func CSumArray(arr []int32) int64 {
	if len(arr) == 0 {
		return 0
	}
	
	// Get pointer to first element
	// LEARN: This is safe because:
	// 1. We're passing to C immediately
	// 2. The slice exists for the duration of the call
	ptr := (*C.int)(unsafe.Pointer(&arr[0]))
	
	result := C.c_sum_array(ptr, C.int(len(arr)))
	return int64(result)
}

// === Memory Management Examples ===

// CAllocateBuffer demonstrates C memory allocation.
//
// LEARN: Memory allocated with C.malloc:
// 1. Is NOT garbage collected
// 2. Must be freed with C.free
// 3. Returns unsafe.Pointer (void*)
func CAllocateBuffer(size int) unsafe.Pointer {
	return C.malloc(C.size_t(size))
}

// CFreeBuffer frees C-allocated memory.
func CFreeBuffer(ptr unsafe.Pointer) {
	if ptr != nil {
		C.free(ptr)
	}
}

// CopyToC copies Go bytes to C memory.
//
// LEARN: C.CBytes is a convenience function that:
// 1. Allocates C memory
// 2. Copies Go bytes to it
// 3. Returns the C pointer
// You MUST call C.free on the result!
func CopyToC(data []byte) unsafe.Pointer {
	if len(data) == 0 {
		return nil
	}
	return C.CBytes(data)
}

// CopyFromC copies bytes from C memory to Go.
//
// LEARN: C.GoBytes copies from C to Go:
// 1. Creates a new Go []byte
// 2. Copies the data
// 3. The result is garbage collected normally
func CopyFromC(ptr unsafe.Pointer, size int) []byte {
	return C.GoBytes(ptr, C.int(size))
}

// === CGO Best Practices ===

// BatchProcess demonstrates reducing CGO overhead by batching.
//
// LEARN: CGO has ~100ns overhead per call. For small operations,
// this dominates execution time. Solution: batch work in C.
//
// Bad:  for i := range data { result += CAdd(data[i], 1) }  // N calls
// Good: result = CBatchAdd(data, 1)                          // 1 call
func BatchProcess(data []int32, addend int32) int64 {
	if len(data) == 0 {
		return 0
	}
	
	// Single CGO call processes entire array
	return CSumArray(data)
}

// === CGO Overhead Demonstration ===

// These functions show why CGO should be avoided for simple operations

// GoAdd is a pure Go implementation for comparison.
func GoAdd(a, b int) int {
	return a + b
}

// GoDistance is a pure Go implementation for comparison.
func GoDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return dx*dx + dy*dy // Skip sqrt for fair comparison
}

// GoSumArray is a pure Go implementation for comparison.
func GoSumArray(arr []int32) int64 {
	var sum int64
	for _, v := range arr {
		sum += int64(v)
	}
	return sum
}

