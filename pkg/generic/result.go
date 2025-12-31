// pkg/generic/result.go
// Generic Result type for type-safe error handling
//
// LEARN: This pattern is inspired by Rust's Result<T, E> type.
// Instead of returning (value, error) tuples, we encapsulate both
// in a single type with methods for safe access.
//
// Benefits over (value, error):
// 1. Cannot accidentally ignore the error
// 2. Enables method chaining
// 3. Provides utility methods (UnwrapOr, Map, etc.)
//
// This is Exercise 3.2 - implement the TODOs to complete the Result type.

package generic

import "errors"

// Result represents either a successful value of type T or an error.
//
// LEARN: The zero value of Result is neither Ok nor Err, which we handle
// by checking the 'ok' field. This is different from Go's (value, error)
// pattern where nil error means success.
type Result[T any] struct {
	value T
	err   error
	ok    bool
}

// Ok creates a successful Result containing the given value.
//
// LEARN: Constructor functions are the idiomatic way to create
// generic instances. Direct struct literals work too but are less readable.
func Ok[T any](value T) Result[T] {
	return Result[T]{
		value: value,
		ok:    true,
	}
}

// Err creates a failed Result containing the given error.
//
// LEARN: We still need to specify the type parameter even though
// we're only storing an error. The type is needed for method signatures.
func Err[T any](err error) Result[T] {
	return Result[T]{
		err: err,
		ok:  false,
	}
}

// FromPair converts a traditional (value, error) return to a Result.
//
// LEARN: This bridges idiomatic Go returns with the Result pattern.
// Useful when calling standard library functions.
func FromPair[T any](value T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(value)
}

// IsOk returns true if the Result contains a success value.
func (r Result[T]) IsOk() bool {
	return r.ok
}

// IsErr returns true if the Result contains an error.
func (r Result[T]) IsErr() bool {
	return !r.ok
}

// Unwrap returns the success value and error.
//
// LEARN: This provides compatibility with Go's standard error handling:
//
//	value, err := result.Unwrap()
//	if err != nil { return err }
func (r Result[T]) Unwrap() (T, error) {
	if !r.ok {
		var zero T
		return zero, r.err
	}
	return r.value, nil
}

// MustUnwrap returns the success value or panics if it's an error.
//
// LEARN: Use sparingly! Only when you're certain the Result is Ok,
// such as in tests or after an explicit IsOk() check.
func (r Result[T]) MustUnwrap() T {
	if !r.ok {
		panic("called MustUnwrap on an Err Result: " + r.err.Error())
	}
	return r.value
}

// UnwrapOr returns the success value or the provided default.
//
// LEARN: This is useful when you have a sensible fallback:
//
//	config := loadConfig().UnwrapOr(defaultConfig)
func (r Result[T]) UnwrapOr(defaultVal T) T {
	if !r.ok {
		return defaultVal
	}
	return r.value
}

// UnwrapOrElse returns the success value or calls the function to get a default.
//
// LEARN: Unlike UnwrapOr, the default is lazily computed:
//
//	value := result.UnwrapOrElse(func() int { return expensiveComputation() })
//
// The function is only called if the Result is an error.
func (r Result[T]) UnwrapOrElse(fn func() T) T {
	if !r.ok {
		return fn()
	}
	return r.value
}

// Error returns the error if present, nil otherwise.
func (r Result[T]) Error() error {
	if !r.ok {
		return r.err
	}
	return nil
}

// TODO Exercise 3.2: Implement the following methods

// Map transforms the success value using the provided function.
//
// LEARN: Map is a fundamental functional programming concept.
// It allows chaining transformations without explicit error checking:
//
//	result := fetchUser(id).
//	    Map(func(u User) string { return u.Name }).
//	    Map(func(name string) string { return "Hello, " + name })
//
// Note: Go doesn't allow type parameters on methods, so Map can only
// return the same type. Use MapResult function for type transformation.
func (r Result[T]) Map(fn func(T) T) Result[T] {
	// TODO: Implement Map
	// If r is Ok, apply fn to the value and return Ok with the result
	// If r is Err, return the same error Result

	if !r.ok {
		return r
	}
	return Ok(fn(r.value))
}

// MapOr transforms the success value or returns the default.
//
// LEARN: Combines Map and UnwrapOr into a single operation:
//
//	length := getString().MapOr(0, func(s string) int { return len(s) })
func (r Result[T]) MapOr(defaultVal T, fn func(T) T) T {
	// TODO: Implement MapOr
	// If r is Ok, apply fn to the value and return the result
	// If r is Err, return defaultVal

	if !r.ok {
		return defaultVal
	}
	return fn(r.value)
}

// And returns other if r is Ok, otherwise returns r's error.
//
// LEARN: Useful for chaining operations that must all succeed:
//
//	result := step1().And(step2()).And(step3())
func (r Result[T]) And(other Result[T]) Result[T] {
	// TODO: Implement And
	// If r is Ok, return other
	// If r is Err, return r

	if !r.ok {
		return r
	}
	return other
}

// Or returns r if it's Ok, otherwise returns other.
//
// LEARN: Useful for fallback chains:
//
//	result := primarySource().Or(fallbackSource())
func (r Result[T]) Or(other Result[T]) Result[T] {
	// TODO: Implement Or
	// If r is Ok, return r
	// If r is Err, return other

	if r.ok {
		return r
	}
	return other
}

// OrElse returns r if it's Ok, otherwise calls fn and returns its result.
//
// LEARN: Like Or, but the alternative is lazily computed:
//
//	result := cache.Get(key).OrElse(func() Result[Value] {
//	    return database.Fetch(key)
//	})
func (r Result[T]) OrElse(fn func() Result[T]) Result[T] {
	// TODO: Implement OrElse
	// If r is Ok, return r
	// If r is Err, call fn and return its result

	if r.ok {
		return r
	}
	return fn()
}

// === Helper Functions for Type Transformation ===
// LEARN: Since Go doesn't allow type parameters on methods,
// we use standalone functions for operations that change types.

// MapResult transforms a Result[T] to Result[U] using the provided function.
//
// Usage:
//
//	userResult := fetchUser(id)
//	nameResult := MapResult(userResult, func(u User) string { return u.Name })
func MapResult[T, U any](r Result[T], fn func(T) U) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return Ok(fn(r.value))
}

// FlatMapResult chains Results, avoiding nested Result[Result[U]].
//
// LEARN: Also known as "bind" or "andThen" in other languages.
// Essential for chaining operations that return Results:
//
//	result := FlatMapResult(getUserID(), func(id string) Result[User] {
//	    return fetchUser(id)
//	})
func FlatMapResult[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return fn(r.value)
}

// Collect converts a slice of Results to a Result of slice.
//
// LEARN: This is an "all or nothing" operation. If any Result is Err,
// the entire collection fails. Useful for parallel operations:
//
//	results := make([]Result[Response], len(requests))
//	for i, req := range requests {
//	    results[i] = fetch(req)
//	}
//	allResponses := Collect(results)
func Collect[T any](results []Result[T]) Result[[]T] {
	values := make([]T, len(results))
	for i, r := range results {
		if !r.ok {
			return Err[[]T](r.err)
		}
		values[i] = r.value
	}
	return Ok(values)
}

// Partition separates a slice of Results into successful values and errors.
//
// LEARN: Unlike Collect, this preserves partial success:
//
//	values, errs := Partition(results)
//	log.Printf("Succeeded: %d, Failed: %d", len(values), len(errs))
func Partition[T any](results []Result[T]) ([]T, []error) {
	var values []T
	var errs []error
	for _, r := range results {
		if r.ok {
			values = append(values, r.value)
		} else {
			errs = append(errs, r.err)
		}
	}
	return values, errs
}

// === Common Errors ===

// ErrNone is returned when unwrapping an empty or uninitialized Result.
var ErrNone = errors.New("result: no value present")


