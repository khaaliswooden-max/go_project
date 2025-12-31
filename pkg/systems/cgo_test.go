//go:build cgo

// pkg/systems/cgo_test.go
// Tests and benchmarks for CGO integration

package systems

import (
	"fmt"
	"math"
	"testing"
	"unsafe"
)

// === Basic Function Tests ===

func TestCAdd(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 3},
		{0, 0, 0},
		{-1, 1, 0},
		{100, 200, 300},
		{-50, -50, -100},
	}

	for _, tc := range tests {
		got := CAdd(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("CAdd(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCStrlen(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"", 0},
		{"hello", 5},
		{"hello world", 11},
		{"你好", 6}, // UTF-8 bytes, not characters
	}

	for _, tc := range tests {
		got := CStrlen(tc.s)
		if got != tc.want {
			t.Errorf("CStrlen(%q) = %d, want %d", tc.s, got, tc.want)
		}
	}
}

func TestCDistance(t *testing.T) {
	tests := []struct {
		x1, y1, x2, y2 float64
		want           float64
	}{
		{0, 0, 3, 4, 5}, // 3-4-5 triangle
		{0, 0, 1, 0, 1}, // Unit distance
		{0, 0, 0, 0, 0}, // Same point
		{1, 1, 4, 5, 5}, // 3-4-5 offset
	}

	for _, tc := range tests {
		got := CDistance(tc.x1, tc.y1, tc.x2, tc.y2)
		if math.Abs(got-tc.want) > 0.0001 {
			t.Errorf("CDistance(%v, %v, %v, %v) = %v, want %v",
				tc.x1, tc.y1, tc.x2, tc.y2, got, tc.want)
		}
	}
}

// === Struct Tests ===

func TestCMakePoint(t *testing.T) {
	tests := []struct {
		x, y int
	}{
		{0, 0},
		{10, 20},
		{-5, 5},
	}

	for _, tc := range tests {
		got := CMakePoint(tc.x, tc.y)
		if got.X != tc.x || got.Y != tc.y {
			t.Errorf("CMakePoint(%d, %d) = %+v, want {X:%d Y:%d}",
				tc.x, tc.y, got, tc.x, tc.y)
		}
	}
}

// === Array Tests ===

func TestCSumArray(t *testing.T) {
	tests := []struct {
		arr  []int32
		want int64
	}{
		{nil, 0},
		{[]int32{}, 0},
		{[]int32{1, 2, 3, 4, 5}, 15},
		{[]int32{-1, -2, -3}, -6},
		{[]int32{1000000, 2000000, 3000000}, 6000000},
	}

	for _, tc := range tests {
		got := CSumArray(tc.arr)
		if got != tc.want {
			t.Errorf("CSumArray(%v) = %d, want %d", tc.arr, got, tc.want)
		}
	}
}

// === Memory Management Tests ===

func TestCAllocateAndFree(t *testing.T) {
	// Allocate memory
	ptr := CAllocateBuffer(1024)
	if ptr == nil {
		t.Fatal("CAllocateBuffer returned nil")
	}

	// Free should not panic
	CFreeBuffer(ptr)

	// Freeing nil should be safe
	CFreeBuffer(nil)
}

func TestCopyToAndFromC(t *testing.T) {
	original := []byte("Hello, CGO!")

	// Copy to C
	cptr := CopyToC(original)
	if cptr == nil {
		t.Fatal("CopyToC returned nil")
	}
	defer CFreeBuffer(cptr)

	// Copy back
	result := CopyFromC(cptr, len(original))

	if string(result) != string(original) {
		t.Errorf("Round trip failed: got %q, want %q", result, original)
	}
}

func TestCopyToC_Empty(t *testing.T) {
	ptr := CopyToC(nil)
	if ptr != nil {
		t.Error("CopyToC(nil) should return nil")
	}

	ptr = CopyToC([]byte{})
	if ptr != nil {
		t.Error("CopyToC([]) should return nil")
	}
}

// === Comparison with Pure Go ===

func TestGoVsCgo_Add(t *testing.T) {
	// Verify both implementations produce same results
	for i := -100; i <= 100; i++ {
		for j := -100; j <= 100; j++ {
			cgo := CAdd(i, j)
			go_ := GoAdd(i, j)
			if cgo != go_ {
				t.Fatalf("CAdd(%d, %d)=%d != GoAdd=%d", i, j, cgo, go_)
			}
		}
	}
}

func TestGoVsCgo_SumArray(t *testing.T) {
	arr := make([]int32, 1000)
	for i := range arr {
		arr[i] = int32(i)
	}

	cgo := CSumArray(arr)
	go_ := GoSumArray(arr)

	if cgo != go_ {
		t.Errorf("CSumArray=%d != GoSumArray=%d", cgo, go_)
	}
}

// === Benchmarks ===

// BenchmarkCGOOverhead demonstrates CGO call overhead
func BenchmarkCGOOverhead(b *testing.B) {
	b.Run("CGO_Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CAdd(1, 2)
		}
	})

	b.Run("Go_Add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GoAdd(1, 2)
		}
	})
}

// BenchmarkCGOString demonstrates string passing overhead
func BenchmarkCGOString(b *testing.B) {
	s := "Hello, World! This is a test string."

	b.Run("CGO_Strlen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = CStrlen(s)
		}
	})

	b.Run("Go_Strlen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = len(s)
		}
	})
}

// BenchmarkCGOArray demonstrates when CGO batching helps
func BenchmarkCGOArray(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		arr := make([]int32, size)
		for i := range arr {
			arr[i] = int32(i)
		}

		b.Run("CGO_"+string(rune(size)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = CSumArray(arr)
			}
		})

		b.Run("Go_"+string(rune(size)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = GoSumArray(arr)
			}
		})
	}
}

// BenchmarkMemoryCopy compares C vs Go memory operations
func BenchmarkMemoryCopy(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		b.Run("CGO_"+string(rune(size)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ptr := CopyToC(data)
				result := CopyFromC(ptr, size)
				CFreeBuffer(ptr)
				_ = result
			}
		})

		b.Run("Go_"+string(rune(size)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				result := make([]byte, size)
				copy(result, data)
				_ = result
			}
		})
	}
}

// === Example Tests ===

func ExampleCAdd() {
	result := CAdd(40, 2)
	fmt.Println(result)
	// Output: 42
}

func ExampleCStrlen() {
	length := CStrlen("Hello, World!")
	fmt.Println(length)
	// Output: 13
}

func ExampleCSumArray() {
	arr := []int32{1, 2, 3, 4, 5}
	sum := CSumArray(arr)
	fmt.Println(sum)
	// Output: 15
}

// === Safety Tests ===

func TestCGOMemorySafety(t *testing.T) {
	// Ensure we don't leak memory in normal usage
	for i := 0; i < 1000; i++ {
		ptr := CAllocateBuffer(1024)
		CFreeBuffer(ptr)
	}

	// String operations shouldn't leak
	for i := 0; i < 1000; i++ {
		_ = CStrlen("test string that should not leak")
	}

	// Copy operations shouldn't leak
	data := []byte("test data")
	for i := 0; i < 1000; i++ {
		ptr := CopyToC(data)
		CFreeBuffer(ptr)
	}
}

func TestCGONilSafety(t *testing.T) {
	// These should not panic
	CFreeBuffer(nil)
	_ = CopyToC(nil)
	_ = CSumArray(nil)
}

func TestCGOSliceLifetime(t *testing.T) {
	// LEARN: This test verifies that slice data isn't moved
	// during the CGO call (important for GC safety)

	arr := make([]int32, 10)
	for i := range arr {
		arr[i] = int32(i + 1)
	}

	// Store reference to prevent GC
	_ = unsafe.Pointer(&arr[0])

	// Call should work correctly
	sum := CSumArray(arr)
	if sum != 55 { // 1+2+3+...+10 = 55
		t.Errorf("CSumArray returned %d, expected 55", sum)
	}
}
