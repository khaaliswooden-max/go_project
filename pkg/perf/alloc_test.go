// pkg/perf/alloc_test.go
// Tests and benchmarks for allocation reduction utilities
//
// LEARN: This file focuses on escape analysis and allocation patterns.
// Run benchmarks with -benchmem to see allocation counts.

package perf

import (
	"strings"
	"testing"
)

// === Unit Tests ===

func TestStringBuilder(t *testing.T) {
	builder := NewStringBuilder(256)

	result := builder.Build(func(b *strings.Builder) {
		b.WriteString("Hello, ")
		b.WriteString("World!")
	})

	if result != "Hello, World!" {
		t.Errorf("got %q, want %q", result, "Hello, World!")
	}
}

func TestSlicePool(t *testing.T) {
	pool := NewSlicePool[int](100)

	s := pool.Get()
	if cap(s) < 100 {
		t.Errorf("capacity = %d, want >= 100", cap(s))
	}
	if len(s) != 0 {
		t.Errorf("length = %d, want 0", len(s))
	}

	// Use the slice
	s = append(s, 1, 2, 3)

	// Return to pool
	pool.Put(s)

	// Get again - should be reset
	s2 := pool.Get()
	if len(s2) != 0 {
		t.Errorf("slice not reset, len = %d", len(s2))
	}
}

func TestGrowSlice(t *testing.T) {
	s := []int{1, 2, 3}

	// Grow to larger capacity
	grown := GrowSlice(s, 100)
	if cap(grown) < 100 {
		t.Errorf("capacity = %d, want >= 100", cap(grown))
	}
	if len(grown) != 3 {
		t.Errorf("length = %d, want 3", len(grown))
	}
	if grown[0] != 1 || grown[1] != 2 || grown[2] != 3 {
		t.Error("data not preserved")
	}

	// Grow to smaller - should be no-op
	same := GrowSlice(grown, 50)
	if cap(same) != cap(grown) {
		t.Error("should not reallocate for smaller capacity")
	}
}

func TestConcatStrings(t *testing.T) {
	result := ConcatStrings("Hello", ", ", "World", "!")
	expected := "Hello, World!"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestConcatBytes(t *testing.T) {
	result := ConcatBytes([]byte("Hello"), []byte(", "), []byte("World"))
	expected := "Hello, World"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestCountRunes(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"héllo", 5},
		{"日本語", 3},
		{"🎉", 1},
		{"", 0},
	}

	for _, tc := range tests {
		got := CountRunes(tc.input)
		if got != tc.want {
			t.Errorf("CountRunes(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestFirstRune(t *testing.T) {
	r, size := FirstRune("hello")
	if r != 'h' || size != 1 {
		t.Errorf("FirstRune(hello) = %c, %d; want 'h', 1", r, size)
	}

	r, size = FirstRune("日本語")
	if r != '日' || size != 3 {
		t.Errorf("FirstRune(日本語) = %c, %d; want '日', 3", r, size)
	}
}

func TestForEachRune(t *testing.T) {
	var runes []rune
	ForEachRune("hello", func(i int, r rune) {
		runes = append(runes, r)
	})

	if string(runes) != "hello" {
		t.Errorf("got %q, want %q", string(runes), "hello")
	}
}

// === Benchmarks ===

// BenchmarkStringConcat compares string concatenation strategies.
//
// LEARN: Naive concatenation is O(n²) due to allocations.
// strings.Builder is O(n) with proper pre-sizing.
func BenchmarkStringConcat(b *testing.B) {
	parts := []string{"The", " ", "quick", " ", "brown", " ", "fox"}

	b.Run("naive_plus", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := ""
			for _, p := range parts {
				result += p
			}
			_ = result
		}
	})

	b.Run("strings_join", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = strings.Join(parts, "")
		}
	})

	b.Run("concat_strings_func", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ConcatStrings(parts...)
		}
	})

	b.Run("pooled_builder", func(b *testing.B) {
		builder := NewStringBuilder(256)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = builder.Build(func(sb *strings.Builder) {
				for _, p := range parts {
					sb.WriteString(p)
				}
			})
		}
	})
}

// BenchmarkSliceGrowth compares slice growing strategies.
func BenchmarkSliceGrowth(b *testing.B) {
	const n = 1000

	b.Run("dynamic_growth", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var s []int
			for j := 0; j < n; j++ {
				s = append(s, j)
			}
			_ = s
		}
	})

	b.Run("pre_allocated", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			s := make([]int, 0, n)
			for j := 0; j < n; j++ {
				s = append(s, j)
			}
			_ = s
		}
	})

	b.Run("pooled_slice", func(b *testing.B) {
		pool := NewSlicePool[int](n)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s := pool.Get()
			for j := 0; j < n; j++ {
				s = append(s, j)
			}
			pool.Put(s)
		}
	})
}

// BenchmarkEscapeAnalysis demonstrates stack vs heap allocation.
//
// LEARN: Run with: go build -gcflags='-m' ./pkg/perf/...
// to see escape analysis decisions.
func BenchmarkEscapeAnalysis(b *testing.B) {
	b.Run("value_no_escape", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := NewPointValue(1.0, 2.0)
			_ = p.X + p.Y
		}
	})

	b.Run("pointer_escapes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			p := NewPointPointer(1.0, 2.0)
			_ = p.X + p.Y
		}
	})
}

// BenchmarkMapPrealloc compares pre-allocated vs dynamic maps.
func BenchmarkMapPrealloc(b *testing.B) {
	const n = 100

	b.Run("dynamic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m := make(map[int]int)
			for j := 0; j < n; j++ {
				m[j] = j * 2
			}
			_ = m
		}
	})

	b.Run("pre_allocated", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			m := PreallocMap[int, int](n)
			for j := 0; j < n; j++ {
				m[j] = j * 2
			}
			_ = m
		}
	})
}

// BenchmarkRuneCount compares rune counting methods.
func BenchmarkRuneCount(b *testing.B) {
	s := "Hello, 世界! This is a test string with unicode: 🎉🎊🎁"

	b.Run("rune_slice", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = len([]rune(s)) // Allocates!
		}
	})

	b.Run("count_runes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = CountRunes(s) // No allocation
		}
	})
}

// === Allocation Count Tests ===

func TestStringBuilderAllocations(t *testing.T) {
	builder := NewStringBuilder(256)
	parts := []string{"a", "b", "c", "d", "e"}

	// Warm up
	for i := 0; i < 10; i++ {
		builder.Build(func(b *strings.Builder) {
			for _, p := range parts {
				b.WriteString(p)
			}
		})
	}

	// After warmup, should be 1 alloc (for final string)
	allocs := testing.AllocsPerRun(100, func() {
		builder.Build(func(b *strings.Builder) {
			for _, p := range parts {
				b.WriteString(p)
			}
		})
	})

	if allocs > 2 {
		t.Errorf("Too many allocations: %.1f, want <= 2", allocs)
	}
}

func TestSlicePoolAllocations(t *testing.T) {
	pool := NewSlicePool[int](100)

	// Warm up
	for i := 0; i < 10; i++ {
		s := pool.Get()
		s = append(s, 1, 2, 3)
		pool.Put(s)
	}

	// After warmup, should be 0 allocs
	allocs := testing.AllocsPerRun(100, func() {
		s := pool.Get()
		s = append(s, 1, 2, 3)
		pool.Put(s)
	})

	if allocs > 1 {
		t.Errorf("Too many allocations: %.1f, want <= 1", allocs)
	}
}
