// pkg/systems/unsafe.go
// Low-level memory operations using the unsafe package
//
// LEARN: The unsafe package is Go's escape hatch from type safety.
// Use it sparingly and document all invariants carefully.
//
// Key concepts:
// 1. unsafe.Pointer can be converted to/from any pointer type
// 2. Struct layout follows platform alignment rules
// 3. Zero-copy conversions trade safety for performance
// 4. All unsafe code should have safe alternatives

package systems

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

// === Memory Layout Analysis ===

// StructField describes a single field in a struct layout.
type StructField struct {
	Name      string  // Field name
	Type      string  // Field type as string
	Size      uintptr // Size in bytes
	Alignment uintptr // Alignment requirement
	Offset    uintptr // Offset from struct start
	Padding   uintptr // Padding bytes before this field
}

// StructLayout describes the complete memory layout of a struct.
//
// LEARN: Understanding struct layout is critical for:
// - Memory optimization (reducing padding)
// - Interoperability with C (matching C struct layouts)
// - Binary file formats (predictable byte positions)
type StructLayout struct {
	Name         string        // Struct type name
	Size         uintptr       // Total size including padding
	Alignment    uintptr       // Alignment requirement
	Fields       []StructField // Individual field info
	TotalPadding uintptr       // Total wasted padding bytes
}

// AnalyzeStruct returns the memory layout of any struct type.
//
// LEARN: This uses reflection to analyze layout at runtime.
// The same information is available at compile time via unsafe.Sizeof,
// unsafe.Alignof, and unsafe.Offsetof, but reflection allows generic analysis.
//
// Example:
//
//	type Example struct {
//	    a bool    // 1 byte
//	    b int64   // 8 bytes, needs 8-byte alignment
//	    c bool    // 1 byte
//	}
//	layout := AnalyzeStruct(Example{})
//	// Shows padding between a and b, and after c
func AnalyzeStruct(v any) StructLayout {
	t := reflect.TypeOf(v)

	// Handle pointer to struct
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return StructLayout{
			Name:      t.String(),
			Size:      t.Size(),
			Alignment: uintptr(t.Align()),
		}
	}

	layout := StructLayout{
		Name:      t.String(),
		Size:      t.Size(),
		Alignment: uintptr(t.Align()),
		Fields:    make([]StructField, t.NumField()),
	}

	var prevEnd uintptr
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		padding := field.Offset - prevEnd
		layout.TotalPadding += padding

		layout.Fields[i] = StructField{
			Name:      field.Name,
			Type:      field.Type.String(),
			Size:      field.Type.Size(),
			Alignment: uintptr(field.Type.Align()),
			Offset:    field.Offset,
			Padding:   padding,
		}

		prevEnd = field.Offset + field.Type.Size()
	}

	// Trailing padding
	if prevEnd < t.Size() {
		layout.TotalPadding += t.Size() - prevEnd
	}

	return layout
}

// FormatLayout returns a human-readable representation of struct layout.
//
// LEARN: Visualizing layout helps understand why field order matters.
// Reordering fields can significantly reduce struct size.
func (l StructLayout) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "=== Struct: %s ===\n", l.Name)
	fmt.Fprintf(&b, "Size: %d bytes, Alignment: %d bytes, Padding: %d bytes (%.1f%%)\n\n",
		l.Size, l.Alignment, l.TotalPadding,
		float64(l.TotalPadding)/float64(l.Size)*100)

	fmt.Fprintf(&b, "Offset | Size | Align | Pad | Field\n")
	fmt.Fprintf(&b, "-------|------|-------|-----|------\n")

	for _, f := range l.Fields {
		padStr := ""
		if f.Padding > 0 {
			padStr = fmt.Sprintf("+%d", f.Padding)
		}
		fmt.Fprintf(&b, "%6d | %4d | %5d | %3s | %s %s\n",
			f.Offset, f.Size, f.Alignment, padStr, f.Name, f.Type)
	}

	// Show trailing padding if any
	if len(l.Fields) > 0 {
		lastField := l.Fields[len(l.Fields)-1]
		trailing := l.Size - (lastField.Offset + lastField.Size)
		if trailing > 0 {
			fmt.Fprintf(&b, "%6d | %4s | %5s | +%d | [trailing padding]\n",
				lastField.Offset+lastField.Size, "", "", trailing)
		}
	}

	return b.String()
}

// OptimizeSuggestion returns a suggested field ordering to minimize padding.
//
// LEARN: The simple optimization strategy is to order fields by
// alignment (largest first). This minimizes gaps between fields.
func (l StructLayout) OptimizeSuggestion() string {
	if l.TotalPadding == 0 {
		return fmt.Sprintf("Struct %s is already optimally ordered (no padding).", l.Name)
	}

	// Sort fields by alignment (descending), then by size (descending)
	type fieldSort struct {
		name  string
		align uintptr
		size  uintptr
	}

	fields := make([]fieldSort, len(l.Fields))
	for i, f := range l.Fields {
		fields[i] = fieldSort{f.Name, f.Alignment, f.Size}
	}

	// Simple bubble sort for clarity
	for i := 0; i < len(fields); i++ {
		for j := i + 1; j < len(fields); j++ {
			if fields[j].align > fields[i].align ||
				(fields[j].align == fields[i].align && fields[j].size > fields[i].size) {
				fields[i], fields[j] = fields[j], fields[i]
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Suggested field order for %s:\n", l.Name)
	for _, f := range fields {
		fmt.Fprintf(&b, "  %s (align: %d, size: %d)\n", f.name, f.align, f.size)
	}

	return b.String()
}

// === Zero-Copy String Conversions ===

// UnsafeStringToBytes converts a string to []byte without allocation.
//
// LEARN: This is EXTREMELY DANGEROUS. The resulting slice:
// - MUST NOT be modified (undefined behavior)
// - MUST NOT outlive the source string
// - Should only be used for read-only operations in hot paths
//
// Use only after profiling proves the standard conversion is a bottleneck.
//
// Safe alternative: []byte(s) - this allocates but is always safe.
func UnsafeStringToBytes(s string) []byte {
	if s == "" {
		return nil
	}

	// LEARN: Go 1.20+ provides unsafe.StringData and unsafe.Slice
	// for this exact purpose, making it "blessed" but still dangerous.
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// UnsafeBytesToString converts []byte to string without allocation.
//
// LEARN: This is DANGEROUS. The source bytes:
// - MUST NOT be modified after this call (undefined behavior)
// - The resulting string is only valid while bytes exist
//
// Use only when:
// 1. You control the byte slice lifecycle
// 2. You guarantee no modifications
// 3. Profiling proves allocation is a bottleneck
//
// Safe alternative: string(b) - this allocates but is always safe.
func UnsafeBytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	// LEARN: Go 1.20+ provides unsafe.String for this purpose
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// === Pointer Arithmetic ===

// PtrAdd performs pointer arithmetic, adding offset bytes to a pointer.
//
// LEARN: This is how C-style pointer arithmetic works in Go.
// It's required for:
// - Walking through memory-mapped structures
// - Implementing custom allocators
// - Interoperating with C code
//
// WARNING: No bounds checking! Caller must ensure validity.
func PtrAdd(ptr unsafe.Pointer, offset uintptr) unsafe.Pointer {
	// LEARN: This must be a single expression!
	// Storing uintptr in a variable allows GC to move the object,
	// making the pointer invalid.
	return unsafe.Pointer(uintptr(ptr) + offset)
}

// ReadAt reads a value of type T at the given byte offset from ptr.
//
// LEARN: Generic unsafe read. This is how you implement zero-copy
// deserialization of binary data.
//
// WARNING: No type safety! Caller must ensure the memory layout is correct.
func ReadAt[T any](ptr unsafe.Pointer, offset uintptr) T {
	return *(*T)(PtrAdd(ptr, offset))
}

// WriteAt writes a value of type T at the given byte offset from ptr.
//
// LEARN: Generic unsafe write. Useful for binary serialization.
//
// WARNING: No bounds checking! No type safety!
func WriteAt[T any](ptr unsafe.Pointer, offset uintptr, value T) {
	*(*T)(PtrAdd(ptr, offset)) = value
}

// === Size and Alignment Utilities ===

// SizeOf returns the size of type T in bytes.
//
// LEARN: This is a type-safe wrapper around unsafe.Sizeof.
// Unlike reflect.TypeOf(v).Size(), this works at compile time.
func SizeOf[T any]() uintptr {
	var zero T
	return unsafe.Sizeof(zero)
}

// AlignOf returns the alignment of type T in bytes.
//
// LEARN: Alignment determines valid addresses for a type.
// A type with alignment N can only be placed at addresses
// that are multiples of N.
func AlignOf[T any]() uintptr {
	var zero T
	return unsafe.Alignof(zero)
}

// === Example Structs for Layout Analysis ===

// InefficientLayout demonstrates poor struct field ordering.
//
// LEARN: Small fields between large fields cause padding.
// Size: 24 bytes (8 bytes wasted on padding)
type InefficientLayout struct {
	A bool  // 1 byte + 7 padding
	B int64 // 8 bytes
	C bool  // 1 byte + 7 padding
}

// EfficientLayout demonstrates optimal struct field ordering.
//
// LEARN: Grouping fields by size minimizes padding.
// Size: 16 bytes (6 bytes padding at end only)
type EfficientLayout struct {
	B int64 // 8 bytes
	A bool  // 1 byte
	C bool  // 1 byte + 6 trailing padding
}

// ComplexLayout demonstrates real-world struct optimization.
//
// LEARN: Fields are ordered by alignment (largest first).
// This is the pattern used in high-performance Go code.
type ComplexLayout struct {
	// 8-byte aligned fields first
	Ptr    *int   // 8 bytes
	Slice  []byte // 24 bytes (ptr + len + cap)
	String string // 16 bytes (ptr + len)
	Int64  int64  // 8 bytes

	// 4-byte aligned fields
	Int32   int32   // 4 bytes
	Float32 float32 // 4 bytes

	// 2-byte aligned fields
	Int16 int16 // 2 bytes

	// 1-byte aligned fields (grouped at end)
	Bool1 bool // 1 byte
	Bool2 bool // 1 byte
	Byte  byte // 1 byte
	// + padding to reach 8-byte alignment
}
