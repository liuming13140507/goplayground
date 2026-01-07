package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	w := NewWorker(5, 100, nil)
	var cnt int64
	ctx, _ := context.WithTimeout(context.Background(), time.Second * 10)
	for i := 0; i < 100; i++ {
		err := w.Submit(ctx,func(ctx context.Context) error {
			atomic.AddInt64(&cnt, 1)
			time.Sleep(time.Millisecond * 2)
			return nil
		})
		if err != nil {
			t.Errorf("submit task %d failed: %v", i, err)
			break
		}
	}
	
	w.Stop()

	if atomic.LoadInt64(&cnt) != 100 {
		t.Errorf("task count is %d, expect 100", atomic.LoadInt64(&cnt))
	}
}
