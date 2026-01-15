package signal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSyncSignal(t *testing.T) {
	s := NewSync[int]()
	var count int32

	// Test Add and Emit
	s.AddListener(func(ctx context.Context, data int) {
		atomic.AddInt32(&count, int32(data))
	}, "add")

	s.Emit(context.Background(), 10)
	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("expected count 10, got %d", count)
	}

	// Test Multiple Listeners
	s.AddListener(func(ctx context.Context, data int) {
		atomic.AddInt32(&count, int32(data))
	}) // anonymous

	s.Emit(context.Background(), 5)
	if atomic.LoadInt32(&count) != 20 { // 10 + 5 (from "add") + 5 (from anonymous)
		t.Errorf("expected count 20, got %d", count)
	}

	// Test Remove
	s.RemoveListener("add")
	s.Emit(context.Background(), 1)
	if atomic.LoadInt32(&count) != 21 { // 20 + 1 (from anonymous)
		t.Errorf("expected count 21, got %d", count)
	}

	// Test Reset
	s.Reset()
	if !s.IsEmpty() {
		t.Error("expected signal to be empty after reset")
	}
}

func TestAsyncSignal(t *testing.T) {
	s := NewAsync[string]()
	var wg sync.WaitGroup
	var received string
	var mu sync.Mutex

	wg.Add(1)
	s.AddListener(func(ctx context.Context, data string) {
		defer wg.Done()
		mu.Lock()
		received = data
		mu.Unlock()
	})

	s.Emit(context.Background(), "hello async")

	// Wait for async execution
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		if received != "hello async" {
			t.Errorf("expected 'hello async', got '%s'", received)
		}
		mu.Unlock()
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for async listener")
	}
}

func TestSignalConcurrency(t *testing.T) {
	s := NewSync[int]()
	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent additions and emissions
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(id int) {
			defer wg.Done()
			s.AddListener(func(ctx context.Context, data int) {}, fmt.Sprintf("k%d", id))
		}(i)
		go func() {
			defer wg.Done()
			s.Emit(ctx, 1)
		}()
	}
	wg.Wait()

	if s.Len() != 100 {
		t.Errorf("expected 100 listeners, got %d", s.Len())
	}
}
