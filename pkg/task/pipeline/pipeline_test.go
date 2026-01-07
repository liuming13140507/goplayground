package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestPipeline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Data source
	source := make(chan int)
	go func() {
		for i := 1; i <= 10; i++ {
			source <- i
		}
		close(source)
	}()

	// 2. Build pipeline
	p1 := Start(ctx, source)

	// Step 1: convert numbers to strings (concurrency 3)
	p2 := Next(p1, 3, 5, func(ctx context.Context, i int) (string, error) {
		time.Sleep(10 * time.Millisecond) // simulate processing time
		return "val-" + strconv.Itoa(i), nil
	})

	// Step 2: add suffix to strings (concurrency 2)
	p3 := Next(p2, 2, 5, func(ctx context.Context, s string) (string, error) {
		return s + "-processed", nil
	})

	// 3. Wait and consume results
	count := 0
	err := p3.Wait(func(s string) error {
		count++
		fmt.Printf("Result: %s\n", s)
		return nil
	})

	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if count != 10 {
		t.Errorf("expected 10 results, got %d", count)
	}
}

func TestPipeline_Error(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	source := make(chan int)
	go func() {
		source <- 1
		source <- 2 // this might trigger an error
		close(source)
	}()

	p := Start(ctx, source)
	pError := Next(p, 1, 1, func(ctx context.Context, i int) (int, error) {
		if i == 2 {
			return 0, fmt.Errorf("oops error at 2")
		}
		return i, nil
	})

	err := pError.Wait(func(i int) error {
		return nil
	})

	if err == nil || err.Error() != "oops error at 2" {
		t.Errorf("expected error 'oops error at 2', got: %v", err)
	}
}

