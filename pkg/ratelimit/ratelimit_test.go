package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRateLimit_GetToken(t *testing.T) {
	rl := NewRateLimit(0, 100)
	var cnt int32
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			res := rl.GetToken()
			if res == true {
				atomic.AddInt32(&cnt, 1)
			}
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&cnt) != 100 {
		t.Errorf("expect %d, but got %d", 100, atomic.LoadInt32(&cnt))
	}
}

func BenchmarkRateLimit_GetToken(b *testing.B) {
	rl := NewRateLimit(1, 10^6)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.GetToken()
	}
	b.StopTimer()
}

func BenchmarkRateLimit_GetTokenParallel(b *testing.B) {
	rl := NewRateLimit(1, 10^6)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.GetToken()
		}
	})
	b.StopTimer()
}
