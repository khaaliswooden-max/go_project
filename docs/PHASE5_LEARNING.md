# Phase 5: Systems Programming - Learning Guide

This document covers Go systems programming techniques: the `unsafe` package, memory layout, memory-mapped files, and CGO. Phase 5 builds on the performance optimization skills from Phase 4 to enable low-level systems access when absolutely necessary.

---

## Prerequisites

Before starting Phase 5, ensure you've completed:
- [x] Phase 1: Foundations (interfaces, error handling, testing)
- [x] Phase 2: Concurrency (goroutines, channels, sync primitives)
- [x] Phase 3: Generics (type parameters, generic data structures)
- [x] Phase 4: Performance (profiling, benchmarks, memory pooling)

---

## ⚠️ Important Warning

**Systems programming in Go should be a last resort, not a first choice.**

The techniques in this phase bypass Go's safety guarantees. Use them only when:
1. Profiling proves standard Go is insufficient
2. You need to interface with C libraries
3. You require direct memory access (mmap, shared memory)
4. You're implementing low-level primitives

**Rule of thumb:** If you can solve the problem with safe Go, do that instead.

---

## 1. The `unsafe` Package

**Files:** `pkg/systems/unsafe.go`, `pkg/systems/unsafe_test.go`

### Concept

The `unsafe` package allows programs to bypass Go's type safety. It provides operations for arbitrary memory manipulation, enabling interoperability with non-Go code and certain performance optimizations.

### What You'll Learn

```go
// LEARN: unsafe.Pointer is the escape hatch from Go's type system
// It can be converted to/from any pointer type
import "unsafe"

// Basic unsafe pointer operations
func pointerConversion() {
    var x int = 42
    
    // Get unsafe.Pointer from any pointer
    p := unsafe.Pointer(&x)
    
    // Convert to different pointer type (DANGEROUS!)
    pf := (*float64)(p)  // Reinterpret int as float64
    
    // LEARN: This is undefined behavior if types don't match!
    // Only use for interoperability or known-compatible types
}

// unsafe.Sizeof returns the size of a type in bytes
func sizeDemo() {
    fmt.Println(unsafe.Sizeof(int(0)))       // 8 on 64-bit
    fmt.Println(unsafe.Sizeof(int32(0)))     // 4
    fmt.Println(unsafe.Sizeof(int64(0)))     // 8
    fmt.Println(unsafe.Sizeof(string("")))   // 16 (pointer + length)
    fmt.Println(unsafe.Sizeof([]int{}))      // 24 (pointer + length + capacity)
}
```

### Memory Layout and Alignment

```go
// LEARN: Structs have padding for alignment
// Understanding layout is critical for interop and optimization

type Inefficient struct {
    a bool    // 1 byte + 7 padding
    b int64   // 8 bytes
    c bool    // 1 byte + 7 padding
}
// Total: 24 bytes

type Efficient struct {
    b int64   // 8 bytes
    a bool    // 1 byte
    c bool    // 1 byte + 6 padding
}
// Total: 16 bytes

// LEARN: unsafe.Alignof returns alignment requirement
func alignmentDemo() {
    var s Efficient
    fmt.Println(unsafe.Alignof(s))     // 8 (largest field alignment)
    fmt.Println(unsafe.Alignof(s.a))   // 1
    fmt.Println(unsafe.Alignof(s.b))   // 8
}

// LEARN: unsafe.Offsetof returns field offset in struct
func offsetDemo() {
    var s Efficient
    fmt.Println(unsafe.Offsetof(s.b))  // 0
    fmt.Println(unsafe.Offsetof(s.a))  // 8
    fmt.Println(unsafe.Offsetof(s.c))  // 9
}
```

### Zero-Copy String/Byte Conversion

```go
// LEARN: Go 1.20+ provides unsafe.String and unsafe.SliceData
// These enable zero-copy conversions (use with extreme care!)

import "unsafe"

// Standard conversion allocates
func standardConversion(b []byte) string {
    return string(b)  // Copies bytes
}

// Zero-copy conversion (DANGEROUS: b must not be modified!)
func unsafeString(b []byte) string {
    if len(b) == 0 {
        return ""
    }
    // Go 1.20+: unsafe.String and unsafe.SliceData
    return unsafe.String(unsafe.SliceData(b), len(b))
}

// Zero-copy string to bytes (EXTREMELY DANGEROUS!)
// Mutating the result is undefined behavior!
func unsafeBytes(s string) []byte {
    if s == "" {
        return nil
    }
    return unsafe.Slice(unsafe.StringData(s), len(s))
}

// LEARN: Only use zero-copy when:
// 1. Performance is critical (proven by profiling)
// 2. Lifetime is well-understood
// 3. Immutability is guaranteed
```

### Valid unsafe.Pointer Conversions

```go
// LEARN: Go specification defines six valid unsafe.Pointer conversions
// Violating these rules causes undefined behavior!

// Rule 1: Pointer to unsafe.Pointer and back
func rule1() {
    x := 42
    p := unsafe.Pointer(&x)
    x2 := (*int)(p)
    fmt.Println(*x2)  // 42
}

// Rule 2: uintptr to unsafe.Pointer (within same expression only!)
func rule2() {
    x := 42
    p := unsafe.Pointer(&x)
    
    // VALID: Pointer arithmetic in single expression
    field := unsafe.Pointer(uintptr(p) + 8)
    
    // INVALID: Storing uintptr breaks GC tracking!
    u := uintptr(p)  // x may be garbage collected!
    _ = unsafe.Pointer(u)  // UNDEFINED BEHAVIOR
}

// Rule 3: Converting reflect.Value.Pointer() to unsafe.Pointer
// Rule 4: Converting reflect.SliceHeader or StringHeader
// Rule 5: syscall.Syscall arguments (when calling C)
// Rule 6: cgo (passing to/from C)
```

### Key Principles

- **Minimal Usage:** Use unsafe only when absolutely necessary
- **Document Why:** Every unsafe block should explain the safety invariants
- **Lifetime Awareness:** Understand when objects can be garbage collected
- **Platform Specific:** Code using unsafe may be non-portable
- **Testing Required:** Test on multiple platforms and with race detector

---

## 2. Memory-Mapped Files

**Files:** `pkg/systems/mmap.go`, `pkg/systems/mmap_test.go`

### Concept

Memory-mapped files allow treating files as if they were in memory. The OS handles paging data in/out, enabling efficient random access to large files without loading them entirely.

### What You'll Learn

```go
// LEARN: Memory mapping trades complexity for performance
// Use cases:
// - Large file random access (databases, log files)
// - Shared memory between processes
// - Fast file I/O (let OS handle caching)

// Platform-specific implementation required:
// - Unix: mmap(2) syscall
// - Windows: CreateFileMapping/MapViewOfFile

import (
    "os"
    "syscall"
)

// Unix mmap example
func mmapUnix(f *os.File, size int) ([]byte, error) {
    return syscall.Mmap(
        int(f.Fd()),           // File descriptor
        0,                     // Offset
        size,                  // Length
        syscall.PROT_READ,     // Protection flags
        syscall.MAP_SHARED,    // Visibility flags
    )
}

func munmapUnix(b []byte) error {
    return syscall.Munmap(b)
}
```

### Cross-Platform Mmap

```go
// LEARN: Abstract platform differences with interfaces

// MappedFile represents a memory-mapped file
type MappedFile interface {
    // Data returns the mapped memory region
    Data() []byte
    
    // Close unmaps the file and releases resources
    Close() error
    
    // Size returns the size of the mapping
    Size() int
    
    // Sync flushes changes to disk (for writable mappings)
    Sync() error
}

// OpenMapped opens a file for memory-mapped reading
func OpenMapped(path string) (MappedFile, error) {
    // Platform-specific implementation
    return openMappedImpl(path)
}
```

### Mmap Use Cases

```go
// Use Case 1: Fast file search
func searchInMappedFile(mf MappedFile, pattern []byte) int64 {
    data := mf.Data()
    // Search directly in memory - OS handles paging
    return bytes.Index(data, pattern)
}

// Use Case 2: Reading large structured files
type Record struct {
    ID        uint64
    Timestamp int64
    Data      [128]byte
}

const recordSize = 144  // sizeof(Record)

func readRecordMapped(mf MappedFile, index int) (*Record, error) {
    data := mf.Data()
    offset := index * recordSize
    if offset+recordSize > len(data) {
        return nil, ErrOutOfBounds
    }
    
    // Zero-copy: reinterpret bytes as struct
    record := (*Record)(unsafe.Pointer(&data[offset]))
    return record, nil
}

// Use Case 3: Shared memory IPC
func createSharedMemory(name string, size int) (MappedFile, error) {
    // Create anonymous mapping that can be shared with child processes
    // Platform-specific implementation
}
```

### Mmap Best Practices

```go
// LEARN: Mmap pitfalls to avoid

// 1. Always check file size before mapping
func safeMmap(path string) (MappedFile, error) {
    info, err := os.Stat(path)
    if err != nil {
        return nil, err
    }
    
    // Don't map empty files
    if info.Size() == 0 {
        return nil, ErrEmptyFile
    }
    
    // Don't map extremely large files (address space limits)
    const maxSize = 1 << 30  // 1GB
    if info.Size() > maxSize {
        return nil, ErrFileTooLarge
    }
    
    return OpenMapped(path)
}

// 2. Handle mapping failures gracefully
func fallbackRead(path string) ([]byte, error) {
    mf, err := OpenMapped(path)
    if err != nil {
        // Fallback to regular file I/O
        return os.ReadFile(path)
    }
    defer mf.Close()
    
    // Copy data (mf.Data() is only valid until Close)
    result := make([]byte, mf.Size())
    copy(result, mf.Data())
    return result, nil
}

// 3. Don't hold mappings too long
// OS has limits on virtual address space
```

### Key Principles

- **Lazy Loading:** Data is loaded on-demand by OS page faults
- **Shared Caching:** Multiple processes can share the same pages
- **Write-Back:** Modified pages may not be written immediately
- **Address Space:** Mappings consume virtual address space
- **Platform Specific:** Windows and Unix APIs differ significantly

---

## 3. CGO: Calling C from Go

**Files:** `pkg/systems/cgo.go` (optional - requires C compiler)

### Concept

CGO enables Go programs to call C functions and use C libraries. This is essential for interfacing with existing C codebases, operating system APIs, and specialized libraries.

### What You'll Learn

```go
// LEARN: CGO uses magic comments to embed C code
// The import "C" line triggers CGO processing

/*
#include <stdlib.h>
#include <string.h>

// Inline C function
int add(int a, int b) {
    return a + b;
}
*/
import "C"

import (
    "fmt"
    "unsafe"
)

func cgoBasics() {
    // Call C function
    result := C.add(40, 2)
    fmt.Println(int(result))  // 42
}
```

### C Type Conversions

```go
/*
#include <stdlib.h>
#include <string.h>
*/
import "C"

import "unsafe"

// LEARN: Converting between Go and C types

func typeConversions() {
    // Integers convert directly
    var goInt int = 42
    cInt := C.int(goInt)
    goInt = int(cInt)
    
    // Strings require C.CString (allocates!)
    goStr := "hello"
    cStr := C.CString(goStr)
    defer C.free(unsafe.Pointer(cStr))  // MUST free!
    
    // Convert back to Go (copies)
    goStr = C.GoString(cStr)
    
    // Bytes to C with length
    goBytes := []byte("world")
    cBytes := C.CBytes(goBytes)
    defer C.free(cBytes)  // MUST free!
    
    // C bytes to Go (copies)
    goBytes = C.GoBytes(cBytes, C.int(len(goBytes)))
}

// LEARN: Memory rules for CGO
// 1. C.CString allocates: you MUST call C.free
// 2. C.CBytes allocates: you MUST call C.free
// 3. Go memory passed to C must not move: use runtime.Pinner (Go 1.21+)
// 4. Don't store Go pointers in C memory
```

### Linking C Libraries

```go
// LEARN: Use LDFLAGS and CFLAGS to configure C compilation

/*
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lmylib
#cgo pkg-config: libsodium

#include <mylib.h>
*/
import "C"

// Platform-specific flags
/*
#cgo linux LDFLAGS: -ldl
#cgo darwin LDFLAGS: -framework Security
#cgo windows LDFLAGS: -lws2_32
*/
import "C"
```

### CGO Performance Considerations

```go
// LEARN: CGO has significant overhead!
// Each CGO call costs ~100ns due to:
// - Stack switching (Go uses small stacks)
// - Scheduler coordination
// - Signal handling

// BAD: Calling C in a tight loop
func slowLoop() {
    for i := 0; i < 1000000; i++ {
        C.sin(C.double(i))  // ~100ns overhead per call!
    }
}

// BETTER: Batch work in C
func fastLoop() {
    // Do loop in C, return results
    results := C.compute_many_sines(1000000)
    defer C.free(unsafe.Pointer(results))
}

// BEST: Avoid CGO if pure Go solution exists
// math.Sin is pure Go and much faster than C.sin via CGO
```

### CGO Safety

```go
// LEARN: CGO breaks Go's safety guarantees

// 1. C code can corrupt Go memory
// 2. C code can cause segfaults
// 3. C code can deadlock Go runtime
// 4. Race detector doesn't see C code

// Defensive patterns:
func safeCGO() (result int, err error) {
    // Recover from C crashes (limited effectiveness)
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("C code crashed: %v", r)
        }
    }()
    
    // Validate inputs before passing to C
    // Check return values from C
    
    result = int(C.safe_function())
    return result, nil
}
```

### Key Principles

- **Overhead:** Each CGO call costs ~100ns minimum
- **Memory:** C allocations must be manually freed
- **Safety:** C code bypasses Go's safety guarantees
- **Portability:** CGO requires C toolchain on all target platforms
- **Debugging:** C crashes can be hard to diagnose

---

## 4. Low-Level Optimization Techniques

**Files:** Examples in all `pkg/systems/*.go` files

### Concept

Sometimes you need to reach beyond safe Go for maximum performance. These techniques should only be used after profiling proves the need.

### What You'll Learn

```go
// LEARN: Bit manipulation for compact data

// Pack multiple values into one integer
func packRGB(r, g, b uint8) uint32 {
    return uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

func unpackRGB(rgb uint32) (r, g, b uint8) {
    return uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb)
}

// Bitmap for efficient set operations
type Bitmap struct {
    bits []uint64
}

func (b *Bitmap) Set(n int) {
    word, bit := n/64, n%64
    b.bits[word] |= 1 << bit
}

func (b *Bitmap) Get(n int) bool {
    word, bit := n/64, n%64
    return b.bits[word]&(1<<bit) != 0
}
```

### SIMD via Go Assembly

```go
// LEARN: Go supports assembly for hot paths
// File: sum_amd64.s (alongside sum.go)

// Pure Go fallback
func sumGo(data []int64) int64 {
    var sum int64
    for _, v := range data {
        sum += v
    }
    return sum
}

// Assembly declaration (implementation in .s file)
func sumSIMD(data []int64) int64

// Choose implementation at runtime
var Sum = chooseSumImpl()

func chooseSumImpl() func([]int64) int64 {
    if cpu.X86.HasAVX2 {
        return sumSIMD
    }
    return sumGo
}
```

### Cache-Friendly Data Structures

```go
// LEARN: Data layout affects cache performance

// BAD: Pointer-heavy structure, poor cache locality
type LinkedList struct {
    value int
    next  *LinkedList  // Pointer chase = cache miss
}

// BETTER: Array-based, excellent cache locality
type ArrayStack struct {
    data []int  // Contiguous memory
    top  int
}

// LEARN: Structure of Arrays vs Array of Structures

// AoS: Cache-unfriendly for columnar access
type AoS struct {
    items []struct {
        x, y, z float64
        name    string
    }
}

// SoA: Cache-friendly for columnar access
type SoA struct {
    x    []float64
    y    []float64
    z    []float64
    name []string
}

func sumX_AoS(a *AoS) float64 {
    var sum float64
    for _, item := range a.items {
        sum += item.x  // Loads entire struct, wastes bandwidth
    }
    return sum
}

func sumX_SoA(a *SoA) float64 {
    var sum float64
    for _, x := range a.x {
        sum += x  // Only loads x values, cache-friendly
    }
    return sum
}
```

### Key Principles

- **Profile First:** Never optimize without data
- **Benchmark Everything:** Verify improvements are real
- **Document Thoroughly:** Low-level code needs extra comments
- **Test Extensively:** Edge cases multiply with unsafe code
- **Maintain Fallbacks:** Have safe implementations for unsupported platforms

---

## Summary: Phase 5 Competencies

After completing Phase 5, you can:

| Skill | Assessment |
|-------|------------|
| Use unsafe.Pointer | Type conversion, memory layout |
| Analyze struct layout | Sizeof, Alignof, Offsetof |
| Implement mmap | Cross-platform file mapping |
| Write basic CGO | C function calls, type conversion |
| Apply low-level optimizations | Bit manipulation, cache awareness |
| Judge when to use unsafe | Cost/benefit analysis |

---

## Exercises Checklist

- [ ] **Exercise 5.1:** Memory Layout Analyzer
  - Implements: Struct layout visualization tool
  - File: `pkg/systems/unsafe.go`
  - Test: `pkg/systems/unsafe_test.go`
  - Requirements:
    - Analyze struct size, alignment, padding
    - Compare efficient vs inefficient layouts
    - Zero-copy string conversion with safety checks
    - Benchmark showing performance difference

- [ ] **Exercise 5.2:** Memory-Mapped File Reader
  - Implements: Cross-platform mmap abstraction
  - File: `pkg/systems/mmap.go`
  - Test: `pkg/systems/mmap_test.go`
  - Requirements:
    - MappedFile interface for abstraction
    - Platform-specific implementations (Unix/Windows)
    - Safe open/close with proper cleanup
    - Benchmark comparing mmap vs regular I/O

- [ ] **Exercise 5.3:** CGO Integration (Optional)
  - Implements: Simple C library wrapper
  - File: `pkg/systems/cgo.go`
  - Test: `pkg/systems/cgo_test.go`
  - Requirements:
    - Safe string passing to C
    - Proper memory cleanup
    - Error handling for C failures
    - Benchmark CGO vs pure Go

---

## Next Phase Preview

**Phase 6: Static Analysis** will cover:
- go/ast package for parsing Go code
- go/types for type checking
- Custom linter development
- Code generation with templates

The unsafe knowledge from Phase 5 provides insight into why Go's type system is designed as it is, and what static analysis tools need to check for.

