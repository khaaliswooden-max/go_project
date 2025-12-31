// pkg/generic/cache.go
// Generic LRU Cache with type-safe key and value types
//
// LEARN: This cache demonstrates:
// 1. Multiple type parameters (K for key, V for value)
// 2. Type constraints (K must be comparable for map keys)
// 3. Generic data structures (doubly-linked list for LRU tracking)
//
// This is Exercise 3.3 - implement the TODOs to complete the cache.

package generic

import (
	"sync"
)

// entry represents a cache entry in the LRU list.
//
// LEARN: The entry stores both key and value because when we evict
// the oldest entry, we need to remove it from both the list AND the map.
type entry[K comparable, V any] struct {
	key   K
	value V
	prev  *entry[K, V]
	next  *entry[K, V]
}

// LRUCache is a thread-safe, generic LRU cache.
//
// LEARN: Type constraints in action:
// - K comparable: Keys must support == and != for map operations
// - V any: Values can be any type
//
// The comparable constraint is required because Go maps only accept
// comparable types as keys.
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*entry[K, V]
	head     *entry[K, V] // Most recently used
	tail     *entry[K, V] // Least recently used
	mu       sync.RWMutex
}

// NewLRUCache creates a new LRU cache with the given capacity.
//
// LEARN: Type parameters must be specified when creating the cache:
//
//	cache := NewLRUCache[string, User](100)
//
// Or with type inference from values (less common for caches):
//
//	cache := NewLRUCacheFrom(map[string]User{...})
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		capacity = 100
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*entry[K, V], capacity),
	}
}

// Get retrieves a value from the cache.
//
// LEARN: Get returns (value, found) similar to map access.
// Accessing an item moves it to the front (most recently used).
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO Exercise 3.3: Implement Get
	//
	// Requirements:
	// 1. Look up the key in c.items
	// 2. If found, move the entry to the front (most recently used)
	// 3. Return the value and true
	// 4. If not found, return zero value and false
	//
	// HINT: Use c.moveToFront(e) to update LRU order

	if e, ok := c.items[key]; ok {
		c.moveToFront(e)
		return e.value, true
	}
	var zero V
	return zero, false
}

// Set adds or updates a value in the cache.
//
// LEARN: Set may trigger eviction if the cache is at capacity.
// The evicted entry is the least recently used (tail of list).
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO Exercise 3.3: Implement Set
	//
	// Requirements:
	// 1. If key exists, update its value and move to front
	// 2. If key is new:
	//    a. Check if cache is at capacity
	//    b. If at capacity, evict the least recently used entry (tail)
	//    c. Create a new entry and add to front
	// 3. Update the items map
	//
	// HINT: Use helper methods c.moveToFront(e), c.addToFront(e), c.evictOldest()

	if e, ok := c.items[key]; ok {
		e.value = value
		c.moveToFront(e)
		return
	}

	// Evict if at capacity
	if len(c.items) >= c.capacity {
		c.evictOldest()
	}

	// Create new entry
	e := &entry[K, V]{key: key, value: value}
	c.addToFront(e)
	c.items[key] = e
}

// Delete removes a key from the cache.
func (c *LRUCache[K, V]) Delete(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		return false
	}

	c.removeEntry(e)
	delete(c.items, key)
	return true
}

// Len returns the number of items in the cache.
func (c *LRUCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Capacity returns the maximum number of items the cache can hold.
func (c *LRUCache[K, V]) Capacity() int {
	return c.capacity
}

// Contains checks if a key exists without affecting LRU order.
//
// LEARN: Unlike Get, Contains doesn't move the entry to the front.
// Use when you want to check existence without affecting eviction priority.
func (c *LRUCache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Peek retrieves a value without affecting LRU order.
func (c *LRUCache[K, V]) Peek(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.items[key]; ok {
		return e.value, true
	}
	var zero V
	return zero, false
}

// Keys returns all keys in the cache (most recent first).
func (c *LRUCache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.items))
	for e := c.head; e != nil; e = e.next {
		keys = append(keys, e.key)
	}
	return keys
}

// Clear removes all entries from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*entry[K, V], c.capacity)
	c.head = nil
	c.tail = nil
}

// GetOrSet returns the existing value or sets and returns the new value.
//
// LEARN: This is an atomic "get or create" pattern. The function is only
// called if the key doesn't exist, preventing redundant computation.
func (c *LRUCache[K, V]) GetOrSet(key K, fn func() V) V {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		c.moveToFront(e)
		return e.value
	}

	// Key doesn't exist, compute and store
	value := fn()

	if len(c.items) >= c.capacity {
		c.evictOldest()
	}

	e := &entry[K, V]{key: key, value: value}
	c.addToFront(e)
	c.items[key] = e
	return value
}

// === Helper Methods ===
// LEARN: These methods manage the doubly-linked list for LRU tracking.

// moveToFront moves an existing entry to the front of the list.
func (c *LRUCache[K, V]) moveToFront(e *entry[K, V]) {
	if e == c.head {
		return // Already at front
	}

	c.removeEntry(e)
	c.addToFront(e)
}

// addToFront adds a new entry to the front of the list.
func (c *LRUCache[K, V]) addToFront(e *entry[K, V]) {
	e.prev = nil
	e.next = c.head

	if c.head != nil {
		c.head.prev = e
	}
	c.head = e

	if c.tail == nil {
		c.tail = e
	}
}

// removeEntry removes an entry from the list (but not from the map).
func (c *LRUCache[K, V]) removeEntry(e *entry[K, V]) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
}

// evictOldest removes the least recently used entry.
func (c *LRUCache[K, V]) evictOldest() {
	if c.tail == nil {
		return
	}

	oldest := c.tail
	c.removeEntry(oldest)
	delete(c.items, oldest.key)
}

// === Callback-based API ===
// LEARN: These methods allow attaching callbacks for cache events.

// CacheWithCallbacks wraps LRUCache with eviction callbacks.
type CacheWithCallbacks[K comparable, V any] struct {
	*LRUCache[K, V]
	onEvict func(key K, value V)
}

// NewCacheWithCallbacks creates a cache that calls onEvict when entries are evicted.
func NewCacheWithCallbacks[K comparable, V any](capacity int, onEvict func(K, V)) *CacheWithCallbacks[K, V] {
	return &CacheWithCallbacks[K, V]{
		LRUCache: NewLRUCache[K, V](capacity),
		onEvict:  onEvict,
	}
}

// Set with eviction callback.
func (c *CacheWithCallbacks[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		e.value = value
		c.moveToFront(e)
		return
	}

	// Evict if at capacity
	if len(c.items) >= c.capacity && c.tail != nil {
		oldest := c.tail
		if c.onEvict != nil {
			c.onEvict(oldest.key, oldest.value)
		}
		c.removeEntry(oldest)
		delete(c.items, oldest.key)
	}

	e := &entry[K, V]{key: key, value: value}
	c.addToFront(e)
	c.items[key] = e
}
