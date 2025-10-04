package lru

import (
	"container/list"
	"errors"
	"sync"
)

// entry is the internal data stored in each list.Element.
type entry[K comparable, V any] struct {
	key   K
	value V
}

// Memoria is a generic LRU cache that holds keys of type K and values of type V.
// It is safe for concurrent use by multiple goroutines.
type Memoria[K comparable, V any] struct {
	capacity int
	mu       sync.RWMutex
	dict     map[K]*list.Element
	ll       *list.List
	len      int

	ch   chan *list.Element
	done chan struct{}

	closed bool
	once   sync.Once
}

// New creates a new Memoria (LRU cache) with the specified capacity.
// Panics if capacity is less than or equal to zero.
func New[K comparable, V any](capacity int) (*Memoria[K, V], error) {
	if capacity <= 0 {
		return nil, errors.New("capacity must be greater than 0")
	}
	m := &Memoria[K, V]{
		capacity: capacity,
		dict:     make(map[K]*list.Element, capacity),
		ll:       list.New(),

		ch:   make(chan *list.Element /* buffer size */, 1024),
		done: make(chan struct{}),
	}

	go m.processEvents()
	return m, nil
}

// Get returns the value associated with the given key if present, and marks
// the entry as recently used. The second return value is true if the key was found.
// If the key is not present, returns (zero value, false).
func (m *Memoria[K, V]) Get(key K) (V, bool) {
	// 1. check dict
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		var zero V
		return zero, false
	}
	ele, ok := m.dict[key]
	m.mu.RUnlock()
	if !ok {
		var zero V
		return zero, false
	}

	// 2. send event to channel
	select {
	case m.ch <- ele:
	case <-m.done:
		var zero V
		return zero, false
	default:
		// channel full, skip updating LRU order to avoid blocking
		return ele.Value.(*entry[K, V]).value, true
	}

	// 3. return value
	ent := ele.Value.(*entry[K, V])
	return ent.value, true
}

// Put inserts or updates the key-value pair into the cache. If the key already exists,
// its value is updated and the entry is moved to the front as most recently used. If insertion
// causes the cache to exceed its capacity, the least recently used entry is evicted.
func (m *Memoria[K, V]) Put(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}

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

	if m.closed {
		return
	}

	m.dict = make(map[K]*list.Element, m.capacity)
	m.ll.Init()
	m.len = 0
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

	if m.closed {
		return nil
	}

	keys := make([]K, 0, m.len)
	for e := m.ll.Back(); e != nil; e = e.Prev() {
		ent := e.Value.(*entry[K, V])
		keys = append(keys, ent.key)
	}
	return keys
}

func (m *Memoria[K, V]) Close() {
	m.once.Do(func() {
		m.mu.Lock()
		m.closed = true
		close(m.done)
		m.mu.Unlock()
	})
}

func (m *Memoria[K, V]) processEvents() {
	for {
		select {
		case ele := <-m.ch:
			m.mu.Lock()
			// check the node still valid & belongs to this list
			if ele.Prev() != nil {
				m.ll.MoveToFront(ele)
			}
			m.mu.Unlock()
		case <-m.done:
			m.mu.Lock()
			m.ch = nil
			m.dict = nil
			m.ll = nil
			m.len = 0
			m.mu.Unlock()
			return
		}
	}
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
