//go:build !cgo

// pkg/systems/cgo_nocgo.go
// Stub implementations when CGO is disabled
//
// LEARN: This file provides fallback implementations when CGO is unavailable.
// This enables the package to compile on systems without a C compiler.

package systems

import (
	"unsafe"
)

// CAdd is a pure Go implementation when CGO is disabled.
func CAdd(a, b int) int {
	return a + b
}

// CStrlen is a pure Go implementation when CGO is disabled.
func CStrlen(s string) int {
	return len(s)
}

// CDistance is a pure Go implementation when CGO is disabled.
func CDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return dx*dx + dy*dy // Note: No sqrt (matches CGO behavior for benchmarks)
}

// Point represents a 2D point.
type Point struct {
	X, Y int
}

// CMakePoint is a pure Go implementation when CGO is disabled.
func CMakePoint(x, y int) Point {
	return Point{X: x, Y: y}
}

// CSumArray is a pure Go implementation when CGO is disabled.
func CSumArray(arr []int32) int64 {
	var sum int64
	for _, v := range arr {
		sum += int64(v)
	}
	return sum
}

// CAllocateBuffer panics when CGO is disabled.
func CAllocateBuffer(size int) unsafe.Pointer {
	panic("CGO required for CAllocateBuffer")
}

// CFreeBuffer panics when CGO is disabled.
func CFreeBuffer(ptr unsafe.Pointer) {
	if ptr != nil {
		panic("CGO required for CFreeBuffer")
	}
}

// CopyToC panics when CGO is disabled.
func CopyToC(data []byte) unsafe.Pointer {
	if len(data) == 0 {
		return nil
	}
	panic("CGO required for CopyToC")
}

// CopyFromC panics when CGO is disabled.
func CopyFromC(ptr unsafe.Pointer, size int) []byte {
	panic("CGO required for CopyFromC")
}

// BatchProcess is a pure Go implementation when CGO is disabled.
func BatchProcess(data []int32, addend int32) int64 {
	return CSumArray(data)
}

// GoAdd is a pure Go implementation.
func GoAdd(a, b int) int {
	return a + b
}

// GoDistance is a pure Go implementation.
func GoDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return dx*dx + dy*dy
}

// GoSumArray is a pure Go implementation.
func GoSumArray(arr []int32) int64 {
	var sum int64
	for _, v := range arr {
		sum += int64(v)
	}
	return sum
}
