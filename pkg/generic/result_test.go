// pkg/generic/result_test.go
// Tests for the generic Result type
//
// LEARN: Testing generic types follows the same patterns as non-generic code.
// We use concrete type parameters in each test case.

package generic

import (
	"errors"
	"testing"
)

func TestOk(t *testing.T) {
	t.Run("creates successful result with value", func(t *testing.T) {
		result := Ok(42)

		if !result.IsOk() {
			t.Error("expected IsOk() to be true")
		}
		if result.IsErr() {
			t.Error("expected IsErr() to be false")
		}

		val, err := result.Unwrap()
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if val != 42 {
			t.Errorf("expected 42, got: %d", val)
		}
	})

	t.Run("works with different types", func(t *testing.T) {
		// String
		strResult := Ok("hello")
		if v, _ := strResult.Unwrap(); v != "hello" {
			t.Errorf("expected 'hello', got: %s", v)
		}

		// Slice
		sliceResult := Ok([]int{1, 2, 3})
		if v, _ := sliceResult.Unwrap(); len(v) != 3 {
			t.Errorf("expected slice of length 3, got: %d", len(v))
		}

		// Struct
		type User struct{ Name string }
		userResult := Ok(User{Name: "Alice"})
		if v, _ := userResult.Unwrap(); v.Name != "Alice" {
			t.Errorf("expected 'Alice', got: %s", v.Name)
		}
	})
}

func TestErr(t *testing.T) {
	t.Run("creates failed result with error", func(t *testing.T) {
		testErr := errors.New("something went wrong")
		result := Err[int](testErr)

		if result.IsOk() {
			t.Error("expected IsOk() to be false")
		}
		if !result.IsErr() {
			t.Error("expected IsErr() to be true")
		}

		_, err := result.Unwrap()
		if err != testErr {
			t.Errorf("expected error to be testErr, got: %v", err)
		}
	})
}

func TestFromPair(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		err       error
		expectOk  bool
		expectVal int
	}{
		{
			name:      "no error returns Ok",
			value:     42,
			err:       nil,
			expectOk:  true,
			expectVal: 42,
		},
		{
			name:      "error returns Err",
			value:     0,
			err:       errors.New("failed"),
			expectOk:  false,
			expectVal: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FromPair(tc.value, tc.err)
			if result.IsOk() != tc.expectOk {
				t.Errorf("expected IsOk=%v, got %v", tc.expectOk, result.IsOk())
			}
		})
	}
}

func TestMustUnwrap(t *testing.T) {
	t.Run("returns value for Ok", func(t *testing.T) {
		result := Ok("success")
		if result.MustUnwrap() != "success" {
			t.Error("expected 'success'")
		}
	})

	t.Run("panics for Err", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()

		result := Err[string](errors.New("fail"))
		_ = result.MustUnwrap() // Should panic
	})
}

func TestUnwrapOr(t *testing.T) {
	t.Run("returns value for Ok", func(t *testing.T) {
		result := Ok(42)
		if result.UnwrapOr(0) != 42 {
			t.Error("expected 42")
		}
	})

	t.Run("returns default for Err", func(t *testing.T) {
		result := Err[int](errors.New("fail"))
		if result.UnwrapOr(99) != 99 {
			t.Error("expected 99")
		}
	})
}

func TestUnwrapOrElse(t *testing.T) {
	t.Run("returns value for Ok without calling function", func(t *testing.T) {
		called := false
		result := Ok(42)
		val := result.UnwrapOrElse(func() int {
			called = true
			return 0
		})

		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
		if called {
			t.Error("function should not have been called")
		}
	})

	t.Run("calls function for Err", func(t *testing.T) {
		called := false
		result := Err[int](errors.New("fail"))
		val := result.UnwrapOrElse(func() int {
			called = true
			return 99
		})

		if val != 99 {
			t.Errorf("expected 99, got %d", val)
		}
		if !called {
			t.Error("function should have been called")
		}
	})
}

func TestMap(t *testing.T) {
	t.Run("transforms Ok value", func(t *testing.T) {
		result := Ok(5)
		mapped := result.Map(func(n int) int { return n * 2 })

		if !mapped.IsOk() {
			t.Error("expected IsOk")
		}
		if v, _ := mapped.Unwrap(); v != 10 {
			t.Errorf("expected 10, got %d", v)
		}
	})

	t.Run("passes through Err", func(t *testing.T) {
		testErr := errors.New("original error")
		result := Err[int](testErr)
		mapped := result.Map(func(n int) int { return n * 2 })

		if !mapped.IsErr() {
			t.Error("expected IsErr")
		}
		if mapped.Error() != testErr {
			t.Error("expected original error to be preserved")
		}
	})
}

func TestMapOr(t *testing.T) {
	t.Run("transforms Ok value", func(t *testing.T) {
		result := Ok(5)
		val := result.MapOr(0, func(n int) int { return n * 2 })
		if val != 10 {
			t.Errorf("expected 10, got %d", val)
		}
	})

	t.Run("returns default for Err", func(t *testing.T) {
		result := Err[int](errors.New("fail"))
		val := result.MapOr(99, func(n int) int { return n * 2 })
		if val != 99 {
			t.Errorf("expected 99, got %d", val)
		}
	})
}

func TestAnd(t *testing.T) {
	t.Run("returns other for Ok", func(t *testing.T) {
		first := Ok(1)
		second := Ok(2)
		result := first.And(second)

		if v, _ := result.Unwrap(); v != 2 {
			t.Errorf("expected 2, got %d", v)
		}
	})

	t.Run("returns first error for Err", func(t *testing.T) {
		firstErr := errors.New("first error")
		first := Err[int](firstErr)
		second := Ok(2)
		result := first.And(second)

		if result.Error() != firstErr {
			t.Error("expected first error")
		}
	})
}

func TestOr(t *testing.T) {
	t.Run("returns first for Ok", func(t *testing.T) {
		first := Ok(1)
		second := Ok(2)
		result := first.Or(second)

		if v, _ := result.Unwrap(); v != 1 {
			t.Errorf("expected 1, got %d", v)
		}
	})

	t.Run("returns second for Err", func(t *testing.T) {
		first := Err[int](errors.New("fail"))
		second := Ok(2)
		result := first.Or(second)

		if v, _ := result.Unwrap(); v != 2 {
			t.Errorf("expected 2, got %d", v)
		}
	})
}

func TestOrElse(t *testing.T) {
	t.Run("returns first for Ok without calling function", func(t *testing.T) {
		called := false
		first := Ok(1)
		result := first.OrElse(func() Result[int] {
			called = true
			return Ok(2)
		})

		if v, _ := result.Unwrap(); v != 1 {
			t.Errorf("expected 1, got %d", v)
		}
		if called {
			t.Error("function should not have been called")
		}
	})

	t.Run("calls function for Err", func(t *testing.T) {
		called := false
		first := Err[int](errors.New("fail"))
		result := first.OrElse(func() Result[int] {
			called = true
			return Ok(2)
		})

		if v, _ := result.Unwrap(); v != 2 {
			t.Errorf("expected 2, got %d", v)
		}
		if !called {
			t.Error("function should have been called")
		}
	})
}

func TestMapResult(t *testing.T) {
	t.Run("transforms type for Ok", func(t *testing.T) {
		result := Ok(42)
		mapped := MapResult(result, func(n int) string {
			return "number"
		})

		if !mapped.IsOk() {
			t.Error("expected IsOk")
		}
		if v, _ := mapped.Unwrap(); v != "number" {
			t.Errorf("expected 'number', got %s", v)
		}
	})

	t.Run("preserves error for Err", func(t *testing.T) {
		testErr := errors.New("original")
		result := Err[int](testErr)
		mapped := MapResult(result, func(n int) string {
			return "number"
		})

		if !mapped.IsErr() {
			t.Error("expected IsErr")
		}
		if mapped.Error() != testErr {
			t.Error("expected original error")
		}
	})
}

func TestFlatMapResult(t *testing.T) {
	t.Run("chains successful results", func(t *testing.T) {
		result := Ok(10)
		chained := FlatMapResult(result, func(n int) Result[string] {
			if n > 5 {
				return Ok("big")
			}
			return Ok("small")
		})

		if v, _ := chained.Unwrap(); v != "big" {
			t.Errorf("expected 'big', got %s", v)
		}
	})

	t.Run("short-circuits on error", func(t *testing.T) {
		testErr := errors.New("early fail")
		result := Err[int](testErr)
		called := false
		chained := FlatMapResult(result, func(n int) Result[string] {
			called = true
			return Ok("never")
		})

		if !chained.IsErr() {
			t.Error("expected IsErr")
		}
		if called {
			t.Error("function should not have been called")
		}
	})

	t.Run("propagates inner error", func(t *testing.T) {
		innerErr := errors.New("inner fail")
		result := Ok(10)
		chained := FlatMapResult(result, func(n int) Result[string] {
			return Err[string](innerErr)
		})

		if chained.Error() != innerErr {
			t.Error("expected inner error")
		}
	})
}

func TestCollect(t *testing.T) {
	t.Run("collects all successful results", func(t *testing.T) {
		results := []Result[int]{Ok(1), Ok(2), Ok(3)}
		collected := Collect(results)

		if !collected.IsOk() {
			t.Error("expected IsOk")
		}
		vals, _ := collected.Unwrap()
		if len(vals) != 3 || vals[0] != 1 || vals[1] != 2 || vals[2] != 3 {
			t.Errorf("expected [1,2,3], got %v", vals)
		}
	})

	t.Run("returns first error", func(t *testing.T) {
		testErr := errors.New("second failed")
		results := []Result[int]{Ok(1), Err[int](testErr), Ok(3)}
		collected := Collect(results)

		if !collected.IsErr() {
			t.Error("expected IsErr")
		}
		if collected.Error() != testErr {
			t.Error("expected testErr")
		}
	})
}

func TestPartition(t *testing.T) {
	t.Run("separates successes and failures", func(t *testing.T) {
		err1 := errors.New("err1")
		err2 := errors.New("err2")
		results := []Result[int]{
			Ok(1),
			Err[int](err1),
			Ok(2),
			Err[int](err2),
			Ok(3),
		}

		values, errs := Partition(results)

		if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
			t.Errorf("expected [1,2,3], got %v", values)
		}
		if len(errs) != 2 || errs[0] != err1 || errs[1] != err2 {
			t.Errorf("expected [err1,err2], got %v", errs)
		}
	})
}
