package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestPipeline(t *testing.T) {
	inCh := make(chan int, 10)
	for i := 0; i < 10; i++ {
		inCh <- i
	}
	close(inCh)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pStart := Start(ctx, inCh)
	pN1 := Next(pStart, 10, 10, func(ctx context.Context, i int) (out int, err error) {
		return i * 2, nil
	})

	cnt := 0
	err := pN1.Wait(func(i int) (err error) {
		cnt++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if cnt != 10 {
		t.Fatal(cnt)
	}
}
