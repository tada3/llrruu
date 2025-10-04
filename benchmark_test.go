package lru_test

import (
	"fmt"
	"github.com/tada3/llrruu"
	"math/rand/v2"
	"testing"
	"time"
)

// benchmark
func BenchmarkNorm(b *testing.B) {
	cache, err := lru.New[string, int](10000)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	defer cache.Close()

	// Pre-fill cache
	for i := 9000; i < 11000; i++ {
		key := key(i)
		fmt.Printf("key = %s\n", key)
		cache.Put(key, i)
	}

	time.Sleep(1000 * time.Millisecond) // Ensure processEvents processes the pre-fill

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		seed := uint64(time.Now().UnixNano())
		src := rand.NewPCG(seed, 12345)

		rnd := rand.New(src)
		for pb.Next() {
			// repeat 100 times
			for range 100 {
				x := randNormInt(rnd)
				key := key(x)
				_, ok := cache.Get(key)
				if !ok {
					cache.Put(key, x)
				}
			}
		}
	})
}

func randNormInt(rnd *rand.Rand) int {
	const MEAN = 10000.0
	const STDDEV = 1000.0
	x := rnd.NormFloat64()*STDDEV + MEAN
	if x < 0 {
		return 0
	} else if x > 20000 {
		return 20000
	}
	return int(x)
}

func key(i int) string {
	return fmt.Sprintf("key-%05d", i)
}
