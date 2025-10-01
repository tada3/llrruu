package lru

import (
	"container/list"
	"errors"
	"sync"
)

// Memoria is a generic LRU cache that holds keys of type K and values of type V.
// It is safe for concurrent use by multiple goroutines.
type Memoria[K comparable, V any] struct {
    capacity int
    mu       sync.Mutex
    dict    map[K]*list.Element
    ll       *list.List
    len int
}

// entry is the internal data stored in each list.Element.
type entry[K comparable, V any] struct {
    key   K
    value V
}

// New creates a new Memoria (LRU cache) with the specified capacity.
// Panics if capacity is less than or equal to zero.
func New[K comparable, V any](capacity int) (*Memoria[K, V], error) {
    if capacity <= 0 {
        return nil, errors.New("capacity must be greater than 0")   
    }
    return &Memoria[K, V]{
        capacity: capacity,
        dict:    make(map[K]*list.Element, capacity),
        ll:       list.New(),
    }, nil
}

// Get returns the value associated with the given key if present, and marks
// the entry as recently used. The second return value is true if the key was found.
// If the key is not present, returns (zero value, false).
func (m *Memoria[K, V]) Get(key K) (V, bool) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if ele, ok := m.dict[key]; ok {
        // Move the accessed element to the front (most recently used)
        m.ll.MoveToFront(ele)
        ent := ele.Value.(*entry[K, V])
        return ent.value, true
    }
    var zero V
    return zero, false
}

// Put inserts or updates the key-value pair into the cache. If the key already exists,
// its value is updated and the entry is moved to the front as most recently used. If insertion
// causes the cache to exceed its capacity, the least recently used entry is evicted.
func (m *Memoria[K, V]) Put(key K, value V) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if ele, ok := m.dict[key]; ok {
        // Existing entry: update value and move to front
        m.ll.MoveToFront(ele)
        ent := ele.Value.(*entry[K, V])
        ent.value = value
        return
    }

    // Insert new element at front
    ent := &entry[K, V]{key: key, value: value}
    ele := m.ll.PushFront(ent)
    m.dict[key] = ele
    m.len++

    // If over capacity, evict least recently used entry
    if m.len > m.capacity {
        m.evict()
    }
}

// Clear removes all entries from the cache.
func (m *Memoria[K, V]) Clear() {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.dict = make(map[K]*list.Element, m.capacity)
    m.ll.Init()
    m.len = 0    
}

// evict removes the least recently used entry (from the back of the list).
// It must be called with the lock held.
func (m *Memoria[K, V]) evict() {
    ele := m.ll.Back()
    if ele == nil {
        return
    }
    m.ll.Remove(ele)
    ent := ele.Value.(*entry[K, V])
    delete(m.dict, ent.key)
    m.len--
}

// Len returns the current number of entries in the cache.
func (m *Memoria[K, V]) Len() int {
    m.mu.Lock()
    defer m.mu.Unlock()

    return m.len
}

// Keys returns a slice of keys ordered from least recently used to most recently used.
// This function is mainly for testing or debugging; it acquires a lock during execution.
func (m *Memoria[K, V]) Keys() []K {
    m.mu.Lock()
    defer m.mu.Unlock()

    keys := make([]K, 0, m.len)
    for e := m.ll.Back(); e != nil; e = e.Prev() {
        ent := e.Value.(*entry[K, V])
        keys = append(keys, ent.key)
    }
    return keys
}