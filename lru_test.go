package lru

import (
    "fmt"
    "math/rand"
    "sync"
    "testing"
)

func TestBasicPutGet(t *testing.T) {
    cache := New[string, int](2)

    // Get on empty cache
    if v, ok := cache.Get("a"); ok {
        t.Errorf("expected not found, but got %v", v)
    }

    // Put and Get
    cache.Put("a", 1)
    if v, ok := cache.Get("a"); !ok || v != 1 {
        t.Errorf("expected found (1), got (%v, %v)", v, ok)
    }

    // Overwrite existing key
    cache.Put("a", 2)
    if v, ok := cache.Get("a"); !ok || v != 2 {
        t.Errorf("expected found (2), got (%v, %v)", v, ok)
    }

    // 2nd key
    cache.Put("b", 10)
    if v, ok := cache.Get("b"); !ok || v != 10 {
        t.Errorf("expected found (10), got (%v, %v)", v, ok)
    }
	// 3rd key
	cache.Put("c", 20)
	if v, ok := cache.Get("c"); !ok || v != 20 {
		t.Errorf("expected found (20), got (%v, %v)", v, ok)
	}

	// Now "a" should be evicted (LRU)
	if cache.Len() != 2 {
		t.Errorf("expected len 2, got %d", cache.Len())
	}
	if v, ok := cache.Get("a"); ok {
		t.Errorf("expected a to be evicted, but got %v", v)
	}
}

func TestEvictionOrder(t *testing.T) {
    cache := New[string, int](2) 
    cache.Put("a", 1)
    cache.Put("b", 2)
    // この時点で a が LRU, b が MRU

    // a を使って更新 => a が MRU, b が LRU
    if _, ok := cache.Get("a"); !ok {
        t.Fatalf("expected a to exist")
    }

    // 新しいキー c を入れると、LRU（b）が消える
    cache.Put("c", 3)

    if _, ok := cache.Get("b"); ok {
        t.Errorf("expected b to be evicted")
    }
    if v, ok := cache.Get("a"); !ok || v != 1 {
        t.Errorf("expected a to remain, got (%v, %v)", v, ok)
    }
    if v, ok := cache.Get("c"); !ok || v != 3 {
        t.Errorf("expected c to exist, got (%v, %v)", v, ok)
    }

    // 次に d を入れると、LRU（a）が消える
    cache.Put("d", 4)
    if _, ok := cache.Get("a"); ok {
        t.Errorf("expected a to be evicted")
    }
    if v, ok := cache.Get("c"); !ok || v != 3 {
        t.Errorf("expected c to remain, got (%v, %v)", v, ok)
    }
    if v, ok := cache.Get("d"); !ok || v != 4 {
        t.Errorf("expected d to exist, got (%v, %v)", v, ok)
    }
}

func TestKeysOrder(t *testing.T) {
    cache := New[string, int](3)
    cache.Put("a", 1)
    cache.Put("b", 2)
    cache.Put("c", 3)
    // LRU -> MRU: a, b, c

    keys := cache.Keys()
	// Keys should be returned in LRU -> MRU order
    want := []string{"a", "b", "c"}
    for i, k := range want {
        if keys[i] != k {
            t.Errorf("keys[%d] = %s; want %s", i, keys[i], k)
        }
    }

	// If a is accessed, it becomes MRU → new order: b, c, a
    _, _ = cache.Get("a")
    keys2 := cache.Keys()
    want2 := []string{"b", "c", "a"}
    for i, k := range want2 {
        if keys2[i] != k {
            t.Errorf("after Get(a), keys[%d] = %s; want %s", i, keys2[i], k)
        }
    }
}

func TestConcurrentAccess(t *testing.T) {
    cache := New[int, int](100) 
    const n = 1000
    var wg sync.WaitGroup
    wg.Add(2)

    // Go routine for putting values
    go func() {
        defer wg.Done()
        for i := 0; i < n; i++ {
            cache.Put(i, i*10)
            // Occasionally get
            j := rand.Intn(n)
            v, ok := cache.Get(j)
			if ok && v != j*10 {
				t.Errorf("expected %d, got %d", j*10, v)
			}
        }
    }()

    // Go routine for getting values
    go func() {
        defer wg.Done()
        for i := 0; i < n; i++ {
            j := rand.Intn(n)
            v, ok := cache.Get(j)
			if ok && v != j*10 {
				t.Errorf("expected %d, got %d", j*10, v)
			}
        }
    }()

    wg.Wait()

	// Final check, Len should not exceed capacity
    if cache.Len() > 100 {
        t.Errorf("cache.Len() = %d; want <= 100", cache.Len())
    }
}


// benchmark
func BenchmarkPutGet(b *testing.B) {
    cache := New[string, int](3000)
	b.ResetTimer() 
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            x := i % 5000
            key := fmt.Sprintf("key%d", x)
            op := rand.Intn(10)
            if op == 0 {
                // put
                cache.Put(key, x)
            } else {
                // get
                cache.Get(key)
            }
            i++
        }
    })
}

// For debugging prints (not required)
func ExampleMemoria() {
    cache := New[string, int](2) 
    cache.Put("a", 1)
    cache.Put("b", 2)
    v, ok := cache.Get("a")
    fmt.Println(v, ok)  // prints 1 true
    cache.Put("c", 3)
    _, ok2 := cache.Get("b")
    fmt.Println(ok2)    // prints false
    // Output:
    // 1 true
    // false
}