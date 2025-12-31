// pkg/generic/cache_test.go
// Tests for the generic LRU cache
//
// LEARN: These tests verify both functionality and thread safety.
// Run with -race to detect data races.

package generic

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestLRUCacheBasic(t *testing.T) {
	t.Run("set and get", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)

		cache.Set("one", 1)
		cache.Set("two", 2)

		val, ok := cache.Get("one")
		if !ok {
			t.Error("expected to find 'one'")
		}
		if val != 1 {
			t.Errorf("expected 1, got %d", val)
		}

		val, ok = cache.Get("two")
		if !ok {
			t.Error("expected to find 'two'")
		}
		if val != 2 {
			t.Errorf("expected 2, got %d", val)
		}
	})

	t.Run("get missing key returns zero value", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)

		val, ok := cache.Get("missing")
		if ok {
			t.Error("expected not to find 'missing'")
		}
		if val != 0 {
			t.Errorf("expected zero value, got %d", val)
		}
	})

	t.Run("update existing key", func(t *testing.T) {
		cache := NewLRUCache[string, string](10)

		cache.Set("key", "original")
		cache.Set("key", "updated")

		val, _ := cache.Get("key")
		if val != "updated" {
			t.Errorf("expected 'updated', got %s", val)
		}

		if cache.Len() != 1 {
			t.Errorf("expected length 1, got %d", cache.Len())
		}
	})
}

func TestLRUCacheEviction(t *testing.T) {
	t.Run("evicts least recently used when at capacity", func(t *testing.T) {
		cache := NewLRUCache[string, int](3)

		cache.Set("a", 1)
		cache.Set("b", 2)
		cache.Set("c", 3)

		// Cache is full: [c, b, a] (most recent first)
		// Adding "d" should evict "a"
		cache.Set("d", 4)

		if _, ok := cache.Get("a"); ok {
			t.Error("'a' should have been evicted")
		}

		// b, c, d should still be present
		for _, key := range []string{"b", "c", "d"} {
			if _, ok := cache.Get(key); !ok {
				t.Errorf("'%s' should still be in cache", key)
			}
		}
	})

	t.Run("access updates LRU order", func(t *testing.T) {
		cache := NewLRUCache[string, int](3)

		cache.Set("a", 1)
		cache.Set("b", 2)
		cache.Set("c", 3)

		// Access "a" to make it most recently used
		cache.Get("a")

		// Now add "d" - should evict "b" (least recently used)
		cache.Set("d", 4)

		if _, ok := cache.Get("b"); ok {
			t.Error("'b' should have been evicted")
		}
		if _, ok := cache.Get("a"); !ok {
			t.Error("'a' should still be in cache after access")
		}
	})

	t.Run("set existing key updates LRU order", func(t *testing.T) {
		cache := NewLRUCache[string, int](3)

		cache.Set("a", 1)
		cache.Set("b", 2)
		cache.Set("c", 3)

		// Update "a" to make it most recently used
		cache.Set("a", 10)

		// Add "d" - should evict "b"
		cache.Set("d", 4)

		if _, ok := cache.Get("b"); ok {
			t.Error("'b' should have been evicted")
		}
		if v, ok := cache.Get("a"); !ok || v != 10 {
			t.Errorf("'a' should be in cache with value 10, got %d", v)
		}
	})
}

func TestLRUCacheDelete(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)

		cache.Set("key", 42)
		deleted := cache.Delete("key")

		if !deleted {
			t.Error("expected Delete to return true")
		}
		if _, ok := cache.Get("key"); ok {
			t.Error("key should not exist after delete")
		}
		if cache.Len() != 0 {
			t.Errorf("expected length 0, got %d", cache.Len())
		}
	})

	t.Run("delete non-existing key", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)

		deleted := cache.Delete("missing")

		if deleted {
			t.Error("expected Delete to return false for missing key")
		}
	})
}

func TestLRUCacheContainsAndPeek(t *testing.T) {
	t.Run("contains does not affect LRU order", func(t *testing.T) {
		cache := NewLRUCache[string, int](2)

		cache.Set("a", 1)
		cache.Set("b", 2)

		// Contains "a" but shouldn't move it to front
		if !cache.Contains("a") {
			t.Error("expected Contains('a') to be true")
		}

		// Add "c" - should evict "a" if Contains didn't affect order
		cache.Set("c", 3)

		if _, ok := cache.Get("a"); ok {
			t.Error("'a' should have been evicted (Contains shouldn't affect LRU)")
		}
	})

	t.Run("peek does not affect LRU order", func(t *testing.T) {
		cache := NewLRUCache[string, int](2)

		cache.Set("a", 1)
		cache.Set("b", 2)

		val, ok := cache.Peek("a")
		if !ok || val != 1 {
			t.Errorf("expected Peek to return 1, got %d", val)
		}

		// Add "c" - should evict "a" if Peek didn't affect order
		cache.Set("c", 3)

		if _, ok := cache.Get("a"); ok {
			t.Error("'a' should have been evicted (Peek shouldn't affect LRU)")
		}
	})
}

func TestLRUCacheKeys(t *testing.T) {
	t.Run("returns keys in LRU order", func(t *testing.T) {
		cache := NewLRUCache[string, int](5)

		cache.Set("a", 1)
		cache.Set("b", 2)
		cache.Set("c", 3)

		// Access "a" to make it most recent
		cache.Get("a")

		keys := cache.Keys()

		// Expected order: a (most recent), c, b (least recent)
		if len(keys) != 3 {
			t.Fatalf("expected 3 keys, got %d", len(keys))
		}
		if keys[0] != "a" {
			t.Errorf("expected first key to be 'a', got '%s'", keys[0])
		}
	})
}

func TestLRUCacheClear(t *testing.T) {
	t.Run("removes all entries", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)

		cache.Set("a", 1)
		cache.Set("b", 2)
		cache.Set("c", 3)

		cache.Clear()

		if cache.Len() != 0 {
			t.Errorf("expected length 0 after Clear, got %d", cache.Len())
		}
		if _, ok := cache.Get("a"); ok {
			t.Error("cache should be empty after Clear")
		}
	})
}

func TestLRUCacheGetOrSet(t *testing.T) {
	t.Run("returns existing value without calling function", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)
		cache.Set("key", 42)

		called := false
		val := cache.GetOrSet("key", func() int {
			called = true
			return 99
		})

		if called {
			t.Error("function should not have been called")
		}
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
	})

	t.Run("calls function and stores result for missing key", func(t *testing.T) {
		cache := NewLRUCache[string, int](10)

		called := false
		val := cache.GetOrSet("key", func() int {
			called = true
			return 99
		})

		if !called {
			t.Error("function should have been called")
		}
		if val != 99 {
			t.Errorf("expected 99, got %d", val)
		}

		// Verify it was stored
		storedVal, _ := cache.Get("key")
		if storedVal != 99 {
			t.Errorf("stored value should be 99, got %d", storedVal)
		}
	})
}

func TestLRUCacheWithComplexTypes(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	t.Run("works with struct values", func(t *testing.T) {
		cache := NewLRUCache[int, User](10)

		cache.Set(1, User{ID: 1, Name: "Alice"})
		cache.Set(2, User{ID: 2, Name: "Bob"})

		user, ok := cache.Get(1)
		if !ok {
			t.Error("expected to find user 1")
		}
		if user.Name != "Alice" {
			t.Errorf("expected 'Alice', got '%s'", user.Name)
		}
	})

	t.Run("works with pointer values", func(t *testing.T) {
		cache := NewLRUCache[string, *User](10)

		alice := &User{ID: 1, Name: "Alice"}
		cache.Set("alice", alice)

		retrieved, ok := cache.Get("alice")
		if !ok {
			t.Error("expected to find alice")
		}
		if retrieved != alice {
			t.Error("expected same pointer")
		}
	})
}

func TestLRUCacheConcurrency(t *testing.T) {
	t.Run("handles concurrent access safely", func(t *testing.T) {
		cache := NewLRUCache[int, int](100)
		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(base int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					cache.Set(base*100+j, j)
				}
			}(i)
		}

		// Readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					cache.Get(j)
				}
			}()
		}

		// Deleters
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(start int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					cache.Delete(start + j)
				}
			}(i * 10)
		}

		wg.Wait()

		// Just verify no panics occurred
		t.Logf("Final cache size: %d", cache.Len())
	})
}

func TestCacheWithCallbacks(t *testing.T) {
	t.Run("calls onEvict when entry is evicted", func(t *testing.T) {
		var evictedKey string
		var evictedValue int
		var evictCount int32

		cache := NewCacheWithCallbacks[string, int](2, func(k string, v int) {
			evictedKey = k
			evictedValue = v
			atomic.AddInt32(&evictCount, 1)
		})

		cache.Set("a", 1)
		cache.Set("b", 2)
		cache.Set("c", 3) // Should evict "a"

		if evictedKey != "a" {
			t.Errorf("expected evicted key 'a', got '%s'", evictedKey)
		}
		if evictedValue != 1 {
			t.Errorf("expected evicted value 1, got %d", evictedValue)
		}
		if evictCount != 1 {
			t.Errorf("expected 1 eviction, got %d", evictCount)
		}
	})
}

func TestLRUCacheCapacity(t *testing.T) {
	t.Run("returns correct capacity", func(t *testing.T) {
		cache := NewLRUCache[string, int](50)

		if cache.Capacity() != 50 {
			t.Errorf("expected capacity 50, got %d", cache.Capacity())
		}
	})

	t.Run("defaults to 100 for invalid capacity", func(t *testing.T) {
		cache := NewLRUCache[string, int](0)
		if cache.Capacity() != 100 {
			t.Errorf("expected default capacity 100, got %d", cache.Capacity())
		}

		cache2 := NewLRUCache[string, int](-5)
		if cache2.Capacity() != 100 {
			t.Errorf("expected default capacity 100, got %d", cache2.Capacity())
		}
	})
}

// Benchmarks
func BenchmarkLRUCacheSet(b *testing.B) {
	cache := NewLRUCache[int, int](1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(i%1000, i)
	}
}

func BenchmarkLRUCacheGet(b *testing.B) {
	cache := NewLRUCache[int, int](1000)
	for i := 0; i < 1000; i++ {
		cache.Set(i, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(i % 1000)
	}
}

func BenchmarkLRUCacheSetGetMixed(b *testing.B) {
	cache := NewLRUCache[int, int](1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			cache.Set(i%1000, i)
		} else {
			cache.Get(i % 1000)
		}
	}
}


