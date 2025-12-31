# Phase 3: Generics - Learning Guide

This document covers Go's generics (type parameters), introduced in Go 1.18. Phase 3 builds on the concurrency patterns from Phase 2 to create type-safe, reusable data structures.

---

## Prerequisites

Before starting Phase 3, ensure you've completed:
- [x] Phase 1: Foundations (interfaces, error handling, testing)
- [x] Phase 2: Concurrency (goroutines, channels, sync primitives)

---

## 1. Type Parameters

**Files:** `pkg/generic/result.go`, `pkg/generic/pool.go`

### Concept
Generics allow you to write functions and types that work with any type, while maintaining compile-time type safety. Before generics, we used `interface{}` (now `any`) which required type assertions at runtime.

### What You'll Learn
```go
// Before generics: type assertions required
func GetFirst(slice []interface{}) interface{} {
    if len(slice) == 0 {
        return nil
    }
    return slice[0]
}
val := GetFirst(mySlice).(string) // Runtime panic if wrong type!

// With generics: compile-time type safety
func GetFirst[T any](slice []T) (T, bool) {
    var zero T
    if len(slice) == 0 {
        return zero, false
    }
    return slice[0], true
}
val, ok := GetFirst(myStrings) // val is already string type!
```

### Syntax
```go
// Type parameter in function
func Map[T, U any](slice []T, fn func(T) U) []U {
    result := make([]U, len(slice))
    for i, v := range slice {
        result[i] = fn(v)
    }
    return result
}

// Type parameter in struct
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
    var zero T
    if len(s.items) == 0 {
        return zero, false
    }
    item := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return item, true
}
```

### Key Principles
- **Type Safety:** Errors caught at compile time, not runtime
- **Zero Value:** Use `var zero T` to get the zero value of any type
- **Type Inference:** Go often infers type parameters from arguments
- **No Runtime Overhead:** Generics are resolved at compile time

---

## 2. Type Constraints

**Files:** `pkg/generic/cache.go`, `pkg/generic/pool.go`

### Concept
Constraints specify what operations are allowed on type parameters. The `any` constraint (alias for `interface{}`) allows any type but limits operations to those common to all types.

### What You'll Learn
```go
// Built-in constraint: comparable (supports == and !=)
func Contains[T comparable](slice []T, target T) bool {
    for _, v := range slice {
        if v == target {
            return true
        }
    }
    return false
}

// Built-in constraint: any (allows any type)
func First[T any](slice []T) (T, bool) { ... }

// Custom constraint using interface
type Number interface {
    ~int | ~int32 | ~int64 | ~float32 | ~float64
}

func Sum[T Number](values []T) T {
    var total T
    for _, v := range values {
        total += v
    }
    return total
}
```

### Constraint Syntax
```go
// Interface as constraint
type Stringer interface {
    String() string
}

func Print[T Stringer](v T) {
    fmt.Println(v.String())
}

// Union constraint (type set)
type Signed interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Approximation constraint (~) includes underlying types
type MyInt int
// ~int matches both int and MyInt

// Combined constraints
type OrderedNumber interface {
    Number
    comparable
}
```

### Standard Library Constraints
```go
import "golang.org/x/exp/constraints"

// constraints.Ordered - types that support < > <= >=
func Max[T constraints.Ordered](a, b T) T {
    if a > b {
        return a
    }
    return b
}

// constraints.Integer, constraints.Float, constraints.Complex
// constraints.Signed, constraints.Unsigned
```

### Key Principles
- **Minimal Constraints:** Use the narrowest constraint that works
- **comparable:** For map keys and equality checks
- **Ordered:** For sorting and comparison operations
- **Approximation (~):** Include type aliases and derived types

---

## 3. Generic Data Structures

**Files:** `pkg/generic/cache.go`

### Concept
Generics shine in data structures where the same logic applies to any type. Before generics, Go developers either used `interface{}` (losing type safety) or code generation.

### What You'll Learn
```go
// Generic Result type (like Rust's Result<T, E>)
// LEARN: This pattern eliminates the (value, error) tuple pattern
// while maintaining type safety for both success and failure cases.
type Result[T any] struct {
    value T
    err   error
    ok    bool
}

func Ok[T any](value T) Result[T] {
    return Result[T]{value: value, ok: true}
}

func Err[T any](err error) Result[T] {
    return Result[T]{err: err, ok: false}
}

func (r Result[T]) Unwrap() (T, error) {
    if !r.ok {
        var zero T
        return zero, r.err
    }
    return r.value, nil
}

func (r Result[T]) UnwrapOr(defaultVal T) T {
    if !r.ok {
        return defaultVal
    }
    return r.value
}
```

### Generic Pool Example
```go
// LEARN: Converting our Phase 2 worker pool to generics provides:
// 1. Type-safe job inputs
// 2. Type-safe results (no type assertions)
// 3. Compile-time error detection

type Job[In, Out any] struct {
    Input   In
    Handler func(context.Context, In) (Out, error)
}

type Result[Out any] struct {
    Value Out
    Err   error
    JobID int
}

type Pool[In, Out any] struct {
    workers int
    jobs    chan Job[In, Out]
    results chan Result[Out]
    // ...
}
```

### Generic Cache Example
```go
// LEARN: Type-safe LRU cache with generic key and value types.
// Constraint: K must be comparable (usable as map key).

type LRUCache[K comparable, V any] struct {
    capacity int
    items    map[K]*entry[K, V]
    order    *list[K] // Doubly linked list for LRU tracking
    mu       sync.RWMutex
}

func (c *LRUCache[K, V]) Get(key K) (V, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if e, ok := c.items[key]; ok {
        c.moveToFront(e)
        return e.value, true
    }
    var zero V
    return zero, false
}
```

### Key Principles
- **Replace interface{}:** Generics provide type safety without assertions
- **Key Constraints:** Map keys must be `comparable`
- **Method Receivers:** Use `[T any]` in method signatures
- **Instantiation:** `cache := NewLRUCache[string, User](100)`

---

## 4. Type Inference

**Files:** All generic code

### Concept
Go can often infer type parameters from function arguments, reducing verbosity while maintaining type safety.

### What You'll Learn
```go
// Explicit type parameters
result := Map[int, string](numbers, strconv.Itoa)

// Inferred from arguments (preferred when possible)
result := Map(numbers, strconv.Itoa)
// Go infers: T=int (from numbers []int), U=string (from strconv.Itoa return)

// Type inference limits
func New[T any]() *Container[T] { ... }
// Must be explicit: container := New[string]()
// Cannot infer: container := New() // Error!
```

### Inference Rules
```go
// 1. Infer from arguments
func Print[T any](v T) { fmt.Println(v) }
Print("hello") // T inferred as string

// 2. Infer from return context (limited)
func Zero[T any]() T { var z T; return z }
var s string = Zero[string]() // Must specify

// 3. Partial inference
func Convert[From, To any](v From, fn func(From) To) To {
    return fn(v)
}
Convert(42, strconv.Itoa) // Both inferred!
```

### Key Principles
- **Let Go Infer:** Omit type parameters when inference works
- **Be Explicit:** Specify when return type determines T
- **Readability:** Sometimes explicit is clearer even when inference works

---

## 5. Generics Best Practices

### When to Use Generics
```go
// YES: Data structures (containers, collections)
type Set[T comparable] map[T]struct{}

// YES: Utility functions that work on any type
func Filter[T any](slice []T, predicate func(T) bool) []T

// YES: Type-safe alternatives to interface{}
type Result[T any] struct { ... }

// MAYBE: When you have 3+ implementations of the same pattern
// Consider generics vs. interfaces

// NO: When an interface would work just as well
// Bad: func Process[T Processor](p T) { p.Process() }
// Good: func Process(p Processor) { p.Process() }
```

### Common Patterns
```go
// Pattern 1: Optional/Maybe type
type Optional[T any] struct {
    value T
    valid bool
}

// Pattern 2: Pair/Tuple
type Pair[T, U any] struct {
    First  T
    Second U
}

// Pattern 3: Generic constructor
func NewSlice[T any](capacity int) []T {
    return make([]T, 0, capacity)
}

// Pattern 4: Method chaining with generics
func (r Result[T]) Map[U any](fn func(T) U) Result[U] {
    if !r.ok {
        return Err[U](r.err)
    }
    return Ok(fn(r.value))
}
// Note: Go doesn't support type parameters on methods,
// so this pattern requires helper functions.
```

### Anti-Patterns
```go
// Anti-pattern 1: Over-generification
// Bad: Every function is generic "just in case"
func Add[T Number](a, b T) T { return a + b }
// Good: Just use concrete types when you know them
func Add(a, b int) int { return a + b }

// Anti-pattern 2: Generic interfaces (usually unnecessary)
// Bad:
type Repository[T any] interface {
    Find(id string) (T, error)
}
// Good: Use concrete interfaces; generify implementations
type UserRepository interface {
    Find(id string) (User, error)
}

// Anti-pattern 3: Ignoring existing abstractions
// Don't create generic sorting when sort.Slice exists
```

---

## Summary: Phase 3 Competencies

After completing Phase 3, you can:

| Skill | Assessment |
|-------|------------|
| Write generic functions | Type parameters, constraints |
| Create generic types | Structs, methods with type params |
| Apply constraints | comparable, custom interfaces, unions |
| Use type inference | Know when explicit vs. inferred |
| Design generic APIs | Result types, containers, utilities |
| Avoid anti-patterns | Interface vs. generics decisions |

---

## Exercises Checklist

- [ ] **Exercise 3.1:** Generic Worker Pool
  - Converts: Phase 2 worker pool to type-safe generics
  - File: `pkg/generic/pool.go`
  - Test: `pkg/generic/pool_test.go`
  - Requirements:
    - Generic input type `In`
    - Generic output type `Out`
    - Type-safe Submit and Results
    - No type assertions needed by caller

- [ ] **Exercise 3.2:** Generic Result Type
  - Implements: Rust-inspired Result[T] for error handling
  - File: `pkg/generic/result.go`
  - Test: `pkg/generic/result_test.go`
  - Requirements:
    - Ok[T](value) and Err[T](error) constructors
    - Unwrap(), UnwrapOr(default), UnwrapOrElse(fn)
    - IsOk(), IsErr() predicates
    - Map(fn) for value transformation

- [ ] **Exercise 3.3:** Generic LRU Cache
  - Implements: Type-safe LRU cache with eviction
  - File: `pkg/generic/cache.go`
  - Test: `pkg/generic/cache_test.go`
  - Requirements:
    - Generic key type (comparable constraint)
    - Generic value type (any)
    - Get, Set, Delete operations
    - LRU eviction when over capacity
    - Thread-safe with RWMutex

---

## Next Phase Preview

**Phase 4: Performance** will cover:
- Profiling with pprof
- Benchmarking with testing.B
- Memory pooling with sync.Pool
- Escape analysis
- Reducing allocations

The generic data structures you build in Phase 3 will be optimized for performance in Phase 4.


