// pkg/systems/unsafe_test.go
// Tests for unsafe memory operations

package systems

import (
	"bytes"
	"fmt"
	"testing"
	"unsafe"
)

// === Struct Layout Tests ===

func TestAnalyzeStruct_Inefficient(t *testing.T) {
	layout := AnalyzeStruct(InefficientLayout{})

	// LEARN: Testing struct layout helps catch platform-specific issues
	t.Logf("\n%s", layout.String())

	// On 64-bit systems, this struct wastes 8 bytes on padding
	expectedSize := uintptr(24)
	if layout.Size != expectedSize {
		t.Errorf("InefficientLayout size = %d, want %d", layout.Size, expectedSize)
	}

	// Should have significant padding
	if layout.TotalPadding == 0 {
		t.Error("InefficientLayout should have padding")
	}

	// Field count
	if len(layout.Fields) != 3 {
		t.Errorf("Field count = %d, want 3", len(layout.Fields))
	}
}

func TestAnalyzeStruct_Efficient(t *testing.T) {
	layout := AnalyzeStruct(EfficientLayout{})

	t.Logf("\n%s", layout.String())

	// This struct should be smaller due to better ordering
	expectedSize := uintptr(16)
	if layout.Size != expectedSize {
		t.Errorf("EfficientLayout size = %d, want %d", layout.Size, expectedSize)
	}

	// Should have less padding than inefficient version
	inefficientLayout := AnalyzeStruct(InefficientLayout{})
	if layout.TotalPadding >= inefficientLayout.TotalPadding {
		t.Errorf("EfficientLayout padding (%d) should be less than InefficientLayout (%d)",
			layout.TotalPadding, inefficientLayout.TotalPadding)
	}
}

func TestAnalyzeStruct_Complex(t *testing.T) {
	layout := AnalyzeStruct(ComplexLayout{})

	t.Logf("\n%s", layout.String())
	t.Logf("\n%s", layout.OptimizeSuggestion())

	// Just verify it doesn't panic and produces valid output
	if layout.Size == 0 {
		t.Error("ComplexLayout size should not be zero")
	}
}

func TestAnalyzeStruct_Pointer(t *testing.T) {
	// Should work with pointer to struct
	layout := AnalyzeStruct(&EfficientLayout{})

	if layout.Name != "systems.EfficientLayout" {
		t.Errorf("Layout name = %s, want systems.EfficientLayout", layout.Name)
	}
}

func TestAnalyzeStruct_NonStruct(t *testing.T) {
	// Should handle non-struct types gracefully
	layout := AnalyzeStruct(42)

	if layout.Size != unsafe.Sizeof(int(0)) {
		t.Errorf("int size mismatch")
	}

	if len(layout.Fields) != 0 {
		t.Error("Non-struct should have no fields")
	}
}

// === Zero-Copy Conversion Tests ===

func TestUnsafeBytesToString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty", []byte{}, ""},
		{"nil", nil, ""},
		{"simple", []byte("hello"), "hello"},
		{"unicode", []byte("你好世界"), "你好世界"},
		{"binary", []byte{0x00, 0xFF, 0x7F}, "\x00\xff\x7f"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UnsafeBytesToString(tc.input)
			if got != tc.want {
				t.Errorf("UnsafeBytesToString(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestUnsafeStringToBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []byte
	}{
		{"empty", "", nil},
		{"simple", "hello", []byte("hello")},
		{"unicode", "你好世界", []byte("你好世界")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UnsafeStringToBytes(tc.input)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("UnsafeStringToBytes(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestZeroCopyRoundTrip(t *testing.T) {
	// LEARN: Test that conversions preserve data
	original := "The quick brown fox"

	// String -> Bytes -> String
	asBytes := UnsafeStringToBytes(original)
	backToString := UnsafeBytesToString(asBytes)

	if backToString != original {
		t.Errorf("Round trip failed: %q != %q", backToString, original)
	}
}

// === Size and Alignment Tests ===

func TestSizeOf(t *testing.T) {
	tests := []struct {
		name string
		size uintptr
		want uintptr
	}{
		{"int8", SizeOf[int8](), 1},
		{"int16", SizeOf[int16](), 2},
		{"int32", SizeOf[int32](), 4},
		{"int64", SizeOf[int64](), 8},
		{"bool", SizeOf[bool](), 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.size != tc.want {
				t.Errorf("SizeOf = %d, want %d", tc.size, tc.want)
			}
		})
	}
}

func TestAlignOf(t *testing.T) {
	// Alignment is typically equal to size for primitive types
	tests := []struct {
		name  string
		align uintptr
		min   uintptr
	}{
		{"int64", AlignOf[int64](), 8},
		{"int32", AlignOf[int32](), 4},
		{"int16", AlignOf[int16](), 2},
		{"int8", AlignOf[int8](), 1},
		{"bool", AlignOf[bool](), 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.align < tc.min {
				t.Errorf("AlignOf = %d, want >= %d", tc.align, tc.min)
			}
		})
	}
}

// === Pointer Arithmetic Tests ===

func TestPtrAdd(t *testing.T) {
	// Create a test buffer
	data := [4]int32{10, 20, 30, 40}
	ptr := unsafe.Pointer(&data[0])

	// Read each element using pointer arithmetic
	for i := 0; i < 4; i++ {
		offset := uintptr(i) * unsafe.Sizeof(int32(0))
		elemPtr := PtrAdd(ptr, offset)
		value := *(*int32)(elemPtr)

		expected := int32((i + 1) * 10)
		if value != expected {
			t.Errorf("data[%d] = %d, want %d", i, value, expected)
		}
	}
}

func TestReadAt(t *testing.T) {
	// Test with a struct
	type Point struct {
		X, Y int32
	}

	p := Point{X: 100, Y: 200}
	ptr := unsafe.Pointer(&p)

	// Read X at offset 0
	x := ReadAt[int32](ptr, 0)
	if x != 100 {
		t.Errorf("X = %d, want 100", x)
	}

	// Read Y at offset 4
	y := ReadAt[int32](ptr, 4)
	if y != 200 {
		t.Errorf("Y = %d, want 200", y)
	}
}

func TestWriteAt(t *testing.T) {
	// Test with a struct
	type Point struct {
		X, Y int32
	}

	p := Point{X: 0, Y: 0}
	ptr := unsafe.Pointer(&p)

	// Write X at offset 0
	WriteAt(ptr, 0, int32(100))
	// Write Y at offset 4
	WriteAt(ptr, 4, int32(200))

	if p.X != 100 {
		t.Errorf("X = %d, want 100", p.X)
	}
	if p.Y != 200 {
		t.Errorf("Y = %d, want 200", p.Y)
	}
}

// === Benchmarks ===

// BenchmarkStringConversion compares safe vs unsafe string conversion
func BenchmarkStringConversion(b *testing.B) {
	data := bytes.Repeat([]byte("hello world "), 100) // 1200 bytes

	b.Run("Safe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := string(data)
			_ = s
		}
	})

	b.Run("Unsafe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := UnsafeBytesToString(data)
			_ = s
		}
	})
}

// BenchmarkBytesConversion compares safe vs unsafe bytes conversion
func BenchmarkBytesConversion(b *testing.B) {
	str := string(bytes.Repeat([]byte("hello world "), 100)) // 1200 chars

	b.Run("Safe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data := []byte(str)
			_ = data
		}
	})

	b.Run("Unsafe", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data := UnsafeStringToBytes(str)
			_ = data
		}
	})
}

// BenchmarkStructCreation compares the two struct layouts
func BenchmarkStructCreation(b *testing.B) {
	b.Run("Inefficient", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := InefficientLayout{A: true, B: 42, C: false}
			_ = s
		}
	})

	b.Run("Efficient", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := EfficientLayout{A: true, B: 42, C: false}
			_ = s
		}
	})
}

// BenchmarkPointerArithmetic compares indexing methods
func BenchmarkPointerArithmetic(b *testing.B) {
	data := make([]int64, 1000)
	for i := range data {
		data[i] = int64(i)
	}

	b.Run("SliceIndex", func(b *testing.B) {
		var sum int64
		for i := 0; i < b.N; i++ {
			sum = 0
			for j := 0; j < len(data); j++ {
				sum += data[j]
			}
		}
		_ = sum
	})

	b.Run("PointerArith", func(b *testing.B) {
		var sum int64
		ptr := unsafe.Pointer(&data[0])
		stride := unsafe.Sizeof(int64(0))

		for i := 0; i < b.N; i++ {
			sum = 0
			for j := uintptr(0); j < uintptr(len(data)); j++ {
				sum += *(*int64)(PtrAdd(ptr, j*stride))
			}
		}
		_ = sum
	})
}

// === Example Tests ===

func ExampleAnalyzeStruct() {
	layout := AnalyzeStruct(InefficientLayout{})

	// Print layout information
	fmt.Printf("Size: %d bytes\n", layout.Size)
	fmt.Printf("Padding: %d bytes\n", layout.TotalPadding)
}
